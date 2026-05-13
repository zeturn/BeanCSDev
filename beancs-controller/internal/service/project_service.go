package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/zeturn/beancs-controller/internal/basaltpass"
	"github.com/zeturn/beancs-controller/internal/config"
	cryptoutil "github.com/zeturn/beancs-controller/internal/crypto"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/k8s"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

var portNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,13}[a-z0-9])?$`)

type ProjectService struct {
	db          *gorm.DB
	credentials *CredentialService
	quota       *QuotaService
	dns         *DNSService
	gitops      *GitOpsService
	build       *GitHubBuildService
	k8s         *k8s.Manager
	registry    *basaltpass.ClientRegistry
	cipher      cryptoutil.Cipher
	cfg         *config.Config
}

func NewProjectService(db *gorm.DB, credentials *CredentialService, quota *QuotaService, dns *DNSService, gitops *GitOpsService, build *GitHubBuildService, k8sManager *k8s.Manager, registry *basaltpass.ClientRegistry, cipher cryptoutil.Cipher, cfg *config.Config) *ProjectService {
	return &ProjectService{db: db, credentials: credentials, quota: quota, dns: dns, gitops: gitops, build: build, k8s: k8sManager, registry: registry, cipher: cipher, cfg: cfg}
}

func (s *ProjectService) AnalyzeRepository(ctx context.Context, userID string, req dto.AnalyzeProjectRepositoryRequest) (*dto.AnalyzeProjectRepositoryResponse, error) {
	if err := s.credentials.RequireAccess(userID, model.CredentialTypeGitHub, req.GitHubCredentialID, false); err != nil {
		return nil, err
	}
	var ghCred model.GitHubCredential
	if err := s.db.WithContext(ctx).First(&ghCred, req.GitHubCredentialID).Error; err != nil {
		return nil, err
	}
	token, err := s.credentials.GitHubToken(ctx, ghCred)
	if err != nil {
		return nil, err
	}
	owner, repo, ok := splitRepo(req.GitHubRepo)
	if !ok {
		return nil, fmt.Errorf("github_repo must be in owner/repo format")
	}
	branch := strings.TrimSpace(req.GitHubBranch)
	if branch == "" {
		branch = "main"
	}

	out := &dto.AnalyzeProjectRepositoryResponse{DefaultPort: 8080}
	if meta, ok, err := githubBasaltAppMeta(ctx, token, owner, repo, branch); err != nil {
		return nil, err
	} else if ok {
		out.Signals = append(out.Signals, ".basalt/app.json found")
		if len(meta.ServicePorts) > 0 {
			out.Ports = meta.ServicePorts
			out.DefaultPort = meta.ServicePorts[0]
		}
	}
	for _, candidate := range dockerfileCandidates() {
		exists, err := githubContentExists(ctx, token, owner, repo, candidate, branch)
		if err != nil {
			return nil, err
		}
		if exists {
			out.Deployable = true
			out.Containerized = true
			out.DockerfilePath = candidate
			out.Signals = append(out.Signals, candidate+" found")
			break
		}
	}
	for _, candidate := range composeFileCandidates() {
		exists, err := githubContentExists(ctx, token, owner, repo, candidate, branch)
		if err != nil {
			return nil, err
		}
		if exists {
			if !out.Containerized {
				out.Deployable = true
				out.Containerized = true
				out.ComposePath = candidate
			}
			out.Signals = append(out.Signals, candidate+" found")
			break
		}
	}
	sourceSignals := 0
	for _, candidate := range sourceFileCandidates() {
		exists, err := githubContentExists(ctx, token, owner, repo, candidate, branch)
		if err != nil {
			return nil, err
		}
		if exists {
			sourceSignals++
			out.Signals = append(out.Signals, candidate+" found")
		}
	}
	if !out.Containerized && sourceSignals > 0 {
		out.Scaffoldable = true
		out.Warnings = append(out.Warnings, "Source layout detected, but no container recipe was found. Add a Dockerfile or Docker Compose file before deploying to avoid image build failures.")
	}
	if out.ComposePath != "" && out.DockerfilePath == "" {
		out.Deployable = false
		out.Warnings = append(out.Warnings, "Docker Compose was detected, but BeanCS GitHub Actions builds currently require a Dockerfile or Containerfile path.")
	}
	if !out.Containerized {
		out.Warnings = append(out.Warnings, "No Dockerfile, Containerfile, or Docker Compose file was found in the repository root or common app directories.")
	}
	return out, nil
}

func (s *ProjectService) CreateProject(ctx context.Context, userID, tenantID, tenantCode string, req dto.CreateProjectRequest) (*model.Project, error) {
	normalizeProjectRequest(&req)
	if err := validateProjectPorts(req.Name, req.Ports); err != nil {
		return nil, err
	}
	req.ExposureMode = aggregateExposureMode(req.Ports)
	primaryPort := req.Ports[0]
	if err := validateProjectSource(&req); err != nil {
		return nil, err
	}
	if err := s.ensureProjectNameAvailable(ctx, req.Name); err != nil {
		return nil, err
	}
	if req.ExposureMode == model.ExposurePublic {
		if req.CloudflareCredentialID == nil {
			return nil, fmt.Errorf("cloudflare_credential_id is required when any port is public")
		}
	}
	if _, ok := model.ResourcePresets[req.ResourcePreset]; !ok {
		return nil, fmt.Errorf("unknown resource preset")
	}

	if req.GitHubCredentialID != 0 {
		if err := s.credentials.RequireAccess(userID, model.CredentialTypeGitHub, req.GitHubCredentialID, false); err != nil {
			return nil, err
		}
	}
	if req.BasaltPassInstanceID != nil {
		if err := s.credentials.RequireAccess(userID, model.CredentialTypeBasaltPass, *req.BasaltPassInstanceID, false); err != nil {
			return nil, err
		}
	}
	if req.CloudflareCredentialID != nil {
		if err := s.credentials.RequireAccess(userID, model.CredentialTypeCloudflare, *req.CloudflareCredentialID, false); err != nil {
			return nil, err
		}
	}

	quotaKey := req.TeamID
	if quotaKey == "" {
		quotaKey = tenantID
	}
	if quotaKey == "" {
		quotaKey = userID
	}
	if err := s.quota.Reserve(ctx, quotaKey, req.ResourcePreset); err != nil {
		return nil, err
	}
	quotaReserved := true
	var rollbacks []func()
	rollbacks = append(rollbacks, func() {
		if quotaReserved {
			_ = s.quota.ReleaseStandalone(context.Background(), quotaKey, req.ResourcePreset)
		}
	})
	rollback := func() {
		for i := len(rollbacks) - 1; i >= 0; i-- {
			rollbacks[i]()
		}
	}

	var ghCred model.GitHubCredential
	var ghToken string
	var ghRepoMeta githubRepositoryMeta
	if req.GitHubCredentialID != 0 {
		if err := s.db.WithContext(ctx).First(&ghCred, req.GitHubCredentialID).Error; err != nil {
			rollback()
			return nil, err
		}
		var err error
		ghToken, err = s.credentials.GitHubToken(ctx, ghCred)
		if err != nil {
			rollback()
			return nil, err
		}
		if req.BuildSource == model.BuildSourceGitHub {
			dockerfilePath, err := s.resolveGitHubDockerfilePath(ctx, ghToken, req)
			if err != nil {
				rollback()
				return nil, err
			}
			req.DockerfilePath = dockerfilePath
			owner, repo, ok := splitRepo(req.GitHubRepo)
			if !ok {
				rollback()
				return nil, fmt.Errorf("github_repo must be in owner/repo format")
			}
			ghRepoMeta, err = fetchGitHubRepositoryMeta(ctx, ghToken, owner, repo)
			if err != nil {
				rollback()
				return nil, err
			}
			if req.GitHubBranch == "" && ghRepoMeta.DefaultBranch != "" {
				req.GitHubBranch = ghRepoMeta.DefaultBranch
			}
		}
	}

	var cfCred model.CloudflareCredential
	var cfToken string
	var err error
	if req.CloudflareCredentialID != nil {
		cfCred, err = s.credentials.CloudflareCredentialForDomain(ctx, *req.CloudflareCredentialID, req.CloudflareZoneID, firstRoutableDomain(req.Ports))
		if err != nil {
			rollback()
			return nil, err
		}
		cfToken, err = s.credentials.DecryptCloudflareToken(cfCred)
		if err != nil {
			rollback()
			return nil, err
		}
	}

	project := &model.Project{
		Name:                   req.Name,
		DisplayName:            req.DisplayName,
		Description:            req.Description,
		OwnerID:                userID,
		TeamID:                 req.TeamID,
		TenantID:               tenantID,
		TenantCode:             tenantCode,
		BuildSource:            req.BuildSource,
		ImageReference:         req.ImageReference,
		SourceArchiveName:      req.SourceArchiveName,
		GitHubCredentialID:     req.GitHubCredentialID,
		GitHubRepo:             req.GitHubRepo,
		GitHubBranch:           req.GitHubBranch,
		GitHubInstallationID:   ghCred.InstallationID,
		GitHubRepoID:           ghRepoMeta.ID,
		GitHubRepoFullName:     coalesce(ghRepoMeta.FullName, req.GitHubRepo),
		DockerfilePath:         req.DockerfilePath,
		AutoDeploy:             req.AutoDeploy == nil || *req.AutoDeploy,
		BasaltPassInstanceID:   req.BasaltPassInstanceID,
		CloudflareCredentialID: req.CloudflareCredentialID,
		ExposureMode:           req.ExposureMode,
		Subdomain:              req.Subdomain,
		Namespace:              projectNamespace(req),
		ResourcePreset:         req.ResourcePreset,
		Port:                   primaryPort.Port,
		Ports:                  req.Ports,
		Replicas:               req.Replicas,
		Status:                 "active",
	}
	project.Domain = firstRoutableDomain(project.Ports)
	if project.BuildSource == model.BuildSourceGitHub {
		if err := configureBeanCSRegistry(project, s.cfg, tenantCode); err != nil {
			rollback()
			return nil, err
		}
		if err := ensureHarborProject(ctx, s.cfg, project.RegistryProject); err != nil {
			rollback()
			return nil, err
		}
	}

	var secret string
	if req.BasaltPassInstanceID != nil {
		bpClient, err := s.registry.GetClientForInstance(*req.BasaltPassInstanceID)
		if err != nil {
			rollback()
			return nil, err
		}
		appResp, err := bpClient.RegisterApp(ctx, &basaltpass.RegisterAppRequest{
			Name:           req.Name,
			Description:    coalesce(req.Description, "BeanCS managed tenant app"),
			HomepageURL:    projectHome(project),
			RedirectURIs:   []string{projectHome(project) + "/callback"},
			AllowedOrigins: []string{projectHome(project)},
			Scopes:         []string{"openid", "profile", "email"},
		})
		if err != nil {
			rollback()
			return nil, err
		}
		project.BasaltAppID = appResp.Data.ID
		project.BasaltClientID = appResp.Data.OAuthClients[0].ClientID
		secret = appResp.Data.OAuthClients[0].ClientSecret
		rollbacks = append(rollbacks, func() { _ = bpClient.DeleteApp(context.Background(), project.BasaltAppID) })

		project.BasaltSecretEnc, err = s.cipher.EncryptString(secret)
		if err != nil {
			rollback()
			return nil, err
		}
	}

	if err := s.k8s.CreateNamespace(ctx, project.Namespace, project.Name); err != nil {
		rollback()
		return nil, err
	}
	rollbacks = append(rollbacks, func() { _ = s.k8s.DeleteNamespace(context.Background(), project.Namespace) })

	if project.BasaltClientID != "" {
		if err := s.k8s.UpsertSecret(ctx, project.Namespace, "basaltpass-keys", project.Name, map[string]string{"client_id": project.BasaltClientID, "client_secret": secret}); err != nil {
			rollback()
			return nil, err
		}
	}
	if project.BuildSource == model.BuildSourceGitHub && project.RegistryImageReference != "" {
		creds, err := ensureHarborPullRobot(ctx, s.cfg, project)
		if err != nil {
			rollback()
			return nil, err
		}
		if err := s.k8s.UpsertRegistryPullSecretWithCredentials(ctx, project.Namespace, project.Name, project.RegistryPullSecretName, creds.Host, creds.Username, creds.Token); err != nil {
			rollback()
			return nil, err
		}
	}
	if err := s.k8s.UpsertSecret(ctx, project.Namespace, "app-env-vars", project.Name, req.Env); err != nil {
		rollback()
		return nil, err
	}
	if err := s.k8s.ApplyNetworkPoliciesForPorts(ctx, project.Namespace, project.Name, project.Ports); err != nil {
		rollback()
		return nil, err
	}
	if err := s.k8s.ApplyServicePorts(ctx, project.Namespace, project.Name, project.Ports); err != nil {
		rollback()
		return nil, err
	}
	if hasPublicPorts(project.Ports) {
		if err := s.k8s.ApplyProjectCertificateIssuer(ctx, project.Namespace, project.Name, cfToken); err != nil {
			rollback()
			return nil, err
		}
	}

	dnsRecords := []model.DNSRecord{}
	for _, p := range project.Ports {
		if p.Exposure != model.ExposurePublic {
			continue
		}
		dnsRecord, err := s.dns.CreateRecordForHost(ctx, cfToken, cfCred, project.Name, p.Domain)
		if err != nil {
			rollback()
			return nil, err
		}
		dnsRecords = append(dnsRecords, *dnsRecord)
		dnsRecords[len(dnsRecords)-1].CloudflareZoneID = cfCred.ZoneID
		recordID := dnsRecord.CloudflareRecordID
		rollbacks = append(rollbacks, func() {
			_ = s.dns.DeleteRecord(context.Background(), cfToken, cfCred.ZoneID, recordID)
		})
	}

	if err := s.k8s.ApplyIngressPorts(ctx, project.Namespace, project.Name, project.Ports); err != nil {
		rollback()
		return nil, err
	}
	if project.ImageReference != "" && project.BuildSource != model.BuildSourceGitHub {
		resources := model.ResourcePresets[project.ResourcePreset]
		if err := s.k8s.ApplyDeploymentPorts(ctx, project.Namespace, project.Name, project.ImageReference, project.Ports, int32(project.Replicas), resources.CPURequest, resources.CPULimit, resources.MemRequest, resources.MemLimit); err != nil {
			rollback()
			return nil, err
		}
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(project).Error; err != nil {
			return err
		}
		for i := range dnsRecords {
			dnsRecords[i].ProjectID = project.ID
			if err := tx.Create(&dnsRecords[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		rollback()
		return nil, err
	}
	deploymentRef := req.GitHubBranch
	if deploymentRef == "" {
		deploymentRef = project.ImageReference
	}
	deployment := &model.Deployment{ProjectID: project.ID, Tag: "initial", CommitSHA: deploymentRef, Status: "provisioning", TriggeredBy: userID}
	if err := s.db.WithContext(ctx).Create(deployment).Error; err != nil {
		rollback()
		return nil, err
	}
	rollbacks = append(rollbacks, func() {
		_ = s.db.WithContext(context.Background()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("project_id = ?", project.ID).Delete(&model.DNSRecord{}).Error; err != nil {
				return err
			}
			if err := tx.Where("project_id = ?", project.ID).Delete(&model.Deployment{}).Error; err != nil {
				return err
			}
			return tx.Delete(&model.Project{}, project.ID).Error
		})
	})

	if err := s.gitops.CommitProjectManifests(ctx, ghToken, ghCred, project); err != nil {
		rollback()
		return nil, err
	}
	if project.BuildSource == model.BuildSourceGitHub && project.AutoDeploy && ghCred.GitOpsRepo != "" {
		if err := s.k8s.ApplyArgoCDApplication(ctx, project.Name, gitOpsRepoURL(ghCred), fmt.Sprintf("apps/%s/overlays/dev", project.Name), project.Namespace); err != nil {
			rollback()
			return nil, err
		}
	}
	_ = s.db.WithContext(ctx).Model(deployment).Updates(map[string]any{"status": "provisioned"}).Error
	if s.build != nil && project.BuildSource == model.BuildSourceGitHub && project.AutoDeploy {
		if _, err := s.build.Start(ctx, project, userID); err != nil {
			rollback()
			return nil, err
		}
	}
	quotaReserved = false
	return project, nil
}

func (s *ProjectService) ListProjects(ctx context.Context, userID string) ([]model.Project, error) {
	var out []model.Project
	err := s.db.WithContext(ctx).Where("owner_id = ?", userID).Order("created_at desc").Find(&out).Error
	return out, err
}

func (s *ProjectService) UpdateProject(ctx context.Context, project *model.Project, req dto.UpdateProjectRequest, triggeredBy string) (*model.Project, error) {
	updates := map[string]any{}
	activateBuild := false
	autoDeployChanged := false
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ResourcePreset != nil {
		updates["resource_preset"] = *req.ResourcePreset
	}
	if req.Port != nil {
		updates["port"] = *req.Port
	}
	if req.Replicas != nil {
		updates["replicas"] = *req.Replicas
	}
	if req.Status != nil {
		updates["status"] = *req.Status
		activateBuild = *req.Status == "active" && project.Status != "active"
	}
	if req.AutoDeploy != nil {
		updates["auto_deploy"] = *req.AutoDeploy
		autoDeployChanged = *req.AutoDeploy != project.AutoDeploy
	}
	if len(updates) == 0 {
		return project, nil
	}
	if err := s.db.WithContext(ctx).Model(project).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(project, project.ID).Error; err != nil {
		return nil, err
	}
	if autoDeployChanged && s.build != nil && project.BuildSource == model.BuildSourceGitHub {
		if err := s.build.EnsureProjectWorkflow(ctx, project); err != nil {
			return nil, err
		}
	}
	if activateBuild && project.AutoDeploy && s.build != nil && project.BuildSource == model.BuildSourceGitHub {
		if _, err := s.build.Start(ctx, project, triggeredBy); err != nil {
			return nil, err
		}
	}
	return project, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, project *model.Project) error {
	var cleanupErrs []error
	var cfCred model.CloudflareCredential
	var cfToken string
	if project.CloudflareCredentialID != nil {
		if err := s.db.WithContext(ctx).First(&cfCred, *project.CloudflareCredentialID).Error; err == nil {
			cfToken, _ = s.credentials.DecryptCloudflareToken(cfCred)
		}
	}
	var dnsRecords []model.DNSRecord
	_ = s.db.WithContext(ctx).Where("project_id = ?", project.ID).Find(&dnsRecords).Error
	for _, rec := range dnsRecords {
		if cfToken != "" {
			zoneID := rec.CloudflareZoneID
			if zoneID == "" {
				zoneID = cfCred.ZoneID
			}
			if err := s.dns.DeleteRecord(ctx, cfToken, zoneID, rec.CloudflareRecordID); err != nil {
				cleanupErrs = append(cleanupErrs, fmt.Errorf("delete DNS record %s: %w", rec.Name, err))
			}
		}
	}
	if err := s.k8s.DeleteProjectResources(ctx, project.Namespace, project.Name); err != nil {
		cleanupErrs = append(cleanupErrs, fmt.Errorf("delete Kubernetes resources for %s/%s: %w", project.Namespace, project.Name, err))
	}
	if project.BasaltPassInstanceID != nil && project.BasaltAppID != 0 {
		if client, err := s.registry.GetClientForInstance(*project.BasaltPassInstanceID); err == nil {
			if err := client.DeleteApp(ctx, project.BasaltAppID); err != nil {
				cleanupErrs = append(cleanupErrs, fmt.Errorf("delete BasaltPass app %d: %w", project.BasaltAppID, err))
			}
		} else {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("load BasaltPass client: %w", err))
		}
	}
	// Clean up GitOps manifests from the gitops repo
	if project.GitHubCredentialID != 0 && s.credentials != nil && s.gitops != nil {
		var ghCred model.GitHubCredential
		if err := s.db.WithContext(ctx).First(&ghCred, project.GitHubCredentialID).Error; err == nil {
			if ghToken, err := s.credentials.GitHubToken(ctx, ghCred); err == nil {
				if err := s.gitops.DeleteProjectManifests(ctx, ghToken, ghCred, project.Name); err != nil {
					cleanupErrs = append(cleanupErrs, fmt.Errorf("delete GitOps manifests for %s: %w", project.Name, err))
				}
			}
		}
	}
	// Delete Argo CD Application CR
	if err := s.k8s.DeleteArgoCDApplication(ctx, project.Name); err != nil {
		cleanupErrs = append(cleanupErrs, fmt.Errorf("delete Argo CD Application %s: %w", project.Name, err))
	}
	if len(cleanupErrs) > 0 {
		return fmt.Errorf("project cleanup failed; database record retained for retry: %v", cleanupErrs)
	}
	quotaKey := project.TeamID
	if quotaKey == "" {
		quotaKey = project.TenantID
	}
	if quotaKey == "" {
		quotaKey = project.OwnerID
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		projectProcesses := tx.Model(&model.Process{}).Select("id").Where("project_id = ?", project.ID)
		if err := tx.Where("process_id IN (?)", projectProcesses).Delete(&model.ProcessJob{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", project.ID).Delete(&model.Process{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", project.ID).Delete(&model.DNSRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", project.ID).Delete(&model.Deployment{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Project{}, project.ID).Error; err != nil {
			return err
		}
		return s.quota.Release(ctx, tx, quotaKey, project.ResourcePreset)
	})
}

func projectNamespace(req dto.CreateProjectRequest) string {
	namespace := strings.TrimSpace(req.Namespace)
	if namespace == "" {
		return "proj-" + req.Name
	}
	return namespace
}

func normalizeProjectRequest(req *dto.CreateProjectRequest) {
	req.BuildSource = strings.ToLower(strings.TrimSpace(req.BuildSource))
	if req.BuildSource == "" {
		req.BuildSource = model.BuildSourceGitHub
	}
	if req.AutoDeploy == nil {
		value := true
		req.AutoDeploy = &value
	}
	req.ImageReference = strings.TrimSpace(req.ImageReference)
	req.SourceArchiveName = strings.TrimSpace(req.SourceArchiveName)
	req.GitHubRepo = strings.TrimSpace(req.GitHubRepo)
	if req.GitHubBranch == "" {
		req.GitHubBranch = "main"
	}
	if req.DockerfilePath == "" {
		req.DockerfilePath = "Dockerfile"
	}
	req.Namespace = strings.TrimSpace(req.Namespace)
	if req.ResourcePreset == "" {
		req.ResourcePreset = "small"
	}
	if req.Port == 0 {
		req.Port = 8080
	}
	if req.Replicas == 0 {
		req.Replicas = 1
	}
	if req.ExposureMode == "" {
		req.ExposureMode = model.ExposurePrivate
	}
	for i := range req.Ports {
		req.Ports[i].Name = strings.ToLower(strings.TrimSpace(req.Ports[i].Name))
		req.Ports[i].Protocol = strings.ToLower(strings.TrimSpace(req.Ports[i].Protocol))
		req.Ports[i].Exposure = strings.ToLower(strings.TrimSpace(req.Ports[i].Exposure))
		req.Ports[i].Domain = strings.ToLower(strings.TrimSpace(req.Ports[i].Domain))
	}
	if req.Env == nil {
		req.Env = map[string]string{}
	}
}

func validateProjectSource(req *dto.CreateProjectRequest) error {
	switch req.BuildSource {
	case model.BuildSourceGitHub:
		if req.GitHubCredentialID == 0 {
			return fmt.Errorf("github_credential_id is required for github installs")
		}
		if _, _, ok := splitRepo(req.GitHubRepo); !ok {
			return fmt.Errorf("github_repo must be in owner/repo format")
		}
		if req.ImageReference == "" {
			req.ImageReference = "ghcr.io/" + strings.ToLower(req.GitHubRepo) + ":latest"
		}
	case model.BuildSourceDockerHub:
		if err := validateImageReference(req.ImageReference); err != nil {
			return fmt.Errorf("docker hub image_reference: %w", err)
		}
		if strings.HasPrefix(strings.ToLower(req.ImageReference), "ghcr.io/") {
			return fmt.Errorf("docker hub image_reference should not start with ghcr.io")
		}
	case model.BuildSourceGHCR:
		if err := validateImageReference(req.ImageReference); err != nil {
			return fmt.Errorf("ghcr image_reference: %w", err)
		}
		if !strings.HasPrefix(strings.ToLower(req.ImageReference), "ghcr.io/") {
			return fmt.Errorf("ghcr image_reference must start with ghcr.io/")
		}
	case model.BuildSourceRegistry:
		if err := validateImageReference(req.ImageReference); err != nil {
			return fmt.Errorf("registry image_reference: %w", err)
		}
	case model.BuildSourceSourceUpload:
		if req.SourceArchiveName == "" {
			return fmt.Errorf("source_archive_name is required for source uploads")
		}
		if err := validateImageReference(req.ImageReference); err != nil {
			return fmt.Errorf("source upload image_reference: %w", err)
		}
	default:
		return fmt.Errorf("build_source must be github, dockerhub, ghcr, registry, or source-upload")
	}
	return nil
}

func validateImageReference(image string) error {
	image = strings.TrimSpace(image)
	if image == "" {
		return fmt.Errorf("is required")
	}
	if strings.ContainsAny(image, " \t\r\n") || strings.Contains(image, "://") {
		return fmt.Errorf("must be a container image reference")
	}
	if strings.HasPrefix(image, "-") || strings.Contains(image, "..") {
		return fmt.Errorf("must be a valid container image reference")
	}
	return nil
}

func gitOpsRepoURL(cred model.GitHubCredential) string {
	repo := strings.TrimSpace(cred.GitOpsRepo)
	if repo == "" {
		return ""
	}
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "git@") {
		return repo
	}
	owner, name, ok := splitRepo(repo)
	if !ok && cred.Org != "" {
		owner, name, ok = cred.Org, repo, true
	}
	if !ok {
		return repo
	}
	return fmt.Sprintf("https://github.com/%s/%s.git", owner, name)
}

func composeFileCandidates() []string {
	names := []string{
		"compose.yaml",
		"compose.yml",
		"compose.prod.yaml",
		"compose.prod.yml",
		"docker-compose.yaml",
		"docker-compose.yml",
		"docker-compose.prod.yaml",
		"docker-compose.prod.yml",
		"docker-compose.production.yaml",
		"docker-compose.production.yml",
	}
	return pathCandidates(names)
}

func dockerfileCandidates() []string {
	return pathCandidates([]string{"Dockerfile", "dockerfile", "Containerfile"})
}

func (s *ProjectService) resolveGitHubDockerfilePath(ctx context.Context, token string, req dto.CreateProjectRequest) (string, error) {
	owner, repo, ok := splitRepo(req.GitHubRepo)
	if !ok {
		return "", fmt.Errorf("github_repo must be in owner/repo format")
	}
	branch := coalesce(req.GitHubBranch, "main")
	if path := strings.TrimSpace(req.DockerfilePath); path != "" {
		exists, err := githubContentExists(ctx, token, owner, repo, path, branch)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("dockerfile_path %q was not found in %s on branch %s", path, req.GitHubRepo, branch)
		}
		return path, nil
	}
	for _, candidate := range dockerfileCandidates() {
		exists, err := githubContentExists(ctx, token, owner, repo, candidate, branch)
		if err != nil {
			return "", err
		}
		if exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no Dockerfile or Containerfile was found in %s on branch %s; add one or set dockerfile_path to the correct path before deploying", req.GitHubRepo, branch)
}

func sourceFileCandidates() []string {
	return pathCandidates([]string{"package.json", "go.mod", "pyproject.toml", "requirements.txt", "Cargo.toml"})
}

func pathCandidates(names []string) []string {
	dirs := []string{"", "backend", "frontend", "server", "client", "app", "api", "web", "cmd"}
	out := make([]string, 0, len(names)*len(dirs))
	for _, dir := range dirs {
		for _, name := range names {
			if dir == "" {
				out = append(out, name)
			} else {
				out = append(out, dir+"/"+name)
			}
		}
	}
	return out
}

func validateProjectPorts(projectName string, ports model.ProjectPorts) error {
	if len(ports) == 0 {
		return fmt.Errorf("ports json is required")
	}
	names := map[string]bool{}
	numbers := map[int]bool{}
	domains := map[string]bool{}
	for _, p := range ports {
		if !portNamePattern.MatchString(p.Name) {
			return fmt.Errorf("port name %q must be a valid dns label of 1-15 chars", p.Name)
		}
		if len(projectName)+1+len(p.Name) > 63 {
			return fmt.Errorf("port name %q is too long for project %q", p.Name, projectName)
		}
		if p.Port < 1 || p.Port > 65535 {
			return fmt.Errorf("port %q must be between 1 and 65535", p.Name)
		}
		if p.Protocol != "" && p.Protocol != "http" {
			return fmt.Errorf("port %q protocol %q is not supported; only http is currently routable", p.Name, p.Protocol)
		}
		switch p.Exposure {
		case model.ExposurePublic, model.ExposurePrivate:
			if p.Domain == "" {
				return fmt.Errorf("port %q requires domain for %s exposure", p.Name, p.Exposure)
			}
		case model.ExposureInternalOnly:
		default:
			return fmt.Errorf("port %q exposure must be public, private, or internal-only", p.Name)
		}
		if names[p.Name] {
			return fmt.Errorf("duplicate port name %q", p.Name)
		}
		names[p.Name] = true
		if numbers[p.Port] {
			return fmt.Errorf("duplicate port number %d", p.Port)
		}
		numbers[p.Port] = true
		if p.Domain != "" {
			if domains[p.Domain] {
				return fmt.Errorf("duplicate port domain %q", p.Domain)
			}
			domains[p.Domain] = true
		}
	}
	return nil
}

func (s *ProjectService) ensureProjectNameAvailable(ctx context.Context, name string) error {
	var existing model.Project
	err := s.db.WithContext(ctx).Select("id").Where("name = ?", name).First(&existing).Error
	if err == nil {
		return fmt.Errorf("project name %q already exists; choose a different project name", name)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func aggregateExposureMode(ports model.ProjectPorts) string {
	out := model.ExposureInternalOnly
	for _, p := range ports {
		if p.Exposure == model.ExposurePublic {
			return model.ExposurePublic
		}
		if p.Exposure == model.ExposurePrivate {
			out = model.ExposurePrivate
		}
	}
	return out
}

func firstRoutableDomain(ports model.ProjectPorts) string {
	for _, p := range ports {
		if p.Domain != "" {
			return p.Domain
		}
	}
	return ""
}

func hasPublicPorts(ports model.ProjectPorts) bool {
	for _, p := range ports {
		if p.Exposure == model.ExposurePublic {
			return true
		}
	}
	return false
}

func projectHome(p *model.Project) string {
	if p.Domain != "" {
		return "https://" + p.Domain
	}
	return "http://" + p.Name + "." + p.Namespace + ".svc.cluster.local"
}

func coalesce(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

func githubContentExists(ctx context.Context, token, owner, repo, filePath, ref string) (bool, error) {
	body, ok, err := githubContentRead(ctx, token, owner, repo, filePath, ref)
	if err != nil || !ok {
		return ok, err
	}
	var payload struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, fmt.Errorf("GitHub returned invalid JSON")
	}
	return payload.Type == "file", nil
}

type basaltAppMeta struct {
	ServicePorts []int `json:"service_ports"`
}

func githubBasaltAppMeta(ctx context.Context, token, owner, repo, ref string) (basaltAppMeta, bool, error) {
	body, ok, err := githubContentRead(ctx, token, owner, repo, ".basalt/app.json", ref)
	if err != nil || !ok {
		return basaltAppMeta{}, ok, err
	}
	var payload struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return basaltAppMeta{}, false, fmt.Errorf("GitHub returned invalid JSON")
	}
	if payload.Type != "file" {
		return basaltAppMeta{}, false, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(payload.Content, "\n", ""))
	if err != nil {
		return basaltAppMeta{}, false, fmt.Errorf("Basalt app metadata was not base64 content")
	}
	var meta basaltAppMeta
	if err := json.Unmarshal(decoded, &meta); err != nil {
		return basaltAppMeta{}, false, fmt.Errorf("Basalt app metadata was invalid JSON")
	}
	return meta, true, nil
}

func githubContentRead(ctx context.Context, token, owner, repo, filePath, ref string) ([]byte, bool, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", url.PathEscape(owner), url.PathEscape(repo), strings.TrimLeft(filePath, "/"), url.QueryEscape(ref))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false, fmt.Errorf("GitHub content check failed: %s", strings.TrimSpace(string(body)))
	}
	return body, true, nil
}
