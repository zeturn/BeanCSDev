package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/zeturn/beancs-controller/internal/basaltpass"
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
	k8s         *k8s.Manager
	registry    *basaltpass.ClientRegistry
	cipher      cryptoutil.Cipher
}

func NewProjectService(db *gorm.DB, credentials *CredentialService, quota *QuotaService, dns *DNSService, gitops *GitOpsService, k8sManager *k8s.Manager, registry *basaltpass.ClientRegistry, cipher cryptoutil.Cipher) *ProjectService {
	return &ProjectService{db: db, credentials: credentials, quota: quota, dns: dns, gitops: gitops, k8s: k8sManager, registry: registry, cipher: cipher}
}

func (s *ProjectService) CreateProject(ctx context.Context, userID, tenantID string, req dto.CreateProjectRequest) (*model.Project, error) {
	normalizeProjectRequest(&req)
	if err := validateProjectPorts(req.Name, req.Ports); err != nil {
		return nil, err
	}
	req.ExposureMode = aggregateExposureMode(req.Ports)
	primaryPort := req.Ports[0]
	if req.ExposureMode == model.ExposurePublic {
		if req.CloudflareCredentialID == nil {
			return nil, fmt.Errorf("cloudflare_credential_id is required when any port is public")
		}
	}
	if _, ok := model.ResourcePresets[req.ResourcePreset]; !ok {
		return nil, fmt.Errorf("unknown resource preset")
	}

	if err := s.credentials.RequireAccess(userID, model.CredentialTypeGitHub, req.GitHubCredentialID, false); err != nil {
		return nil, err
	}
	if err := s.credentials.RequireAccess(userID, model.CredentialTypeBasaltPass, req.BasaltPassInstanceID, false); err != nil {
		return nil, err
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
	if err := s.db.WithContext(ctx).First(&ghCred, req.GitHubCredentialID).Error; err != nil {
		rollback()
		return nil, err
	}
	ghToken, err := s.credentials.DecryptGitHubToken(ghCred)
	if err != nil {
		rollback()
		return nil, err
	}

	var cfCred model.CloudflareCredential
	var cfToken string
	if req.CloudflareCredentialID != nil {
		if err := s.db.WithContext(ctx).First(&cfCred, *req.CloudflareCredentialID).Error; err != nil {
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
		GitHubCredentialID:     req.GitHubCredentialID,
		GitHubRepo:             req.GitHubRepo,
		GitHubBranch:           req.GitHubBranch,
		DockerfilePath:         req.DockerfilePath,
		BasaltPassInstanceID:   req.BasaltPassInstanceID,
		CloudflareCredentialID: req.CloudflareCredentialID,
		ExposureMode:           req.ExposureMode,
		Subdomain:              req.Subdomain,
		Namespace:              "proj-" + req.Name,
		ResourcePreset:         req.ResourcePreset,
		Port:                   primaryPort.Port,
		Ports:                  req.Ports,
		Replicas:               req.Replicas,
		Status:                 "active",
	}
	project.Domain = firstRoutableDomain(project.Ports)

	bpClient, err := s.registry.GetClientForInstance(req.BasaltPassInstanceID)
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
	secret := appResp.Data.OAuthClients[0].ClientSecret
	rollbacks = append(rollbacks, func() { _ = bpClient.DeleteApp(context.Background(), project.BasaltAppID) })

	project.BasaltSecretEnc, err = s.cipher.EncryptString(secret)
	if err != nil {
		rollback()
		return nil, err
	}

	if err := s.k8s.CreateNamespace(ctx, project.Namespace, project.Name); err != nil {
		rollback()
		return nil, err
	}
	rollbacks = append(rollbacks, func() { _ = s.k8s.DeleteNamespace(context.Background(), project.Namespace) })

	if err := s.k8s.UpsertSecret(ctx, project.Namespace, "basaltpass-keys", project.Name, map[string]string{"client_id": project.BasaltClientID, "client_secret": secret}); err != nil {
		rollback()
		return nil, err
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
		recordID := dnsRecord.CloudflareRecordID
		rollbacks = append(rollbacks, func() {
			_ = s.dns.DeleteRecord(context.Background(), cfToken, cfCred.ZoneID, recordID)
		})
	}

	if err := s.k8s.ApplyIngressPorts(ctx, project.Namespace, project.Name, project.Ports); err != nil {
		rollback()
		return nil, err
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
	quotaReserved = false
	return project, nil
}

func (s *ProjectService) ListProjects(ctx context.Context, userID string) ([]model.Project, error) {
	var out []model.Project
	err := s.db.WithContext(ctx).Where("owner_id = ?", userID).Order("created_at desc").Find(&out).Error
	return out, err
}

func (s *ProjectService) UpdateProject(ctx context.Context, project *model.Project, req dto.UpdateProjectRequest) (*model.Project, error) {
	updates := map[string]any{}
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
	}
	if len(updates) == 0 {
		return project, nil
	}
	if err := s.db.WithContext(ctx).Model(project).Updates(updates).Error; err != nil {
		return nil, err
	}
	return project, s.db.WithContext(ctx).First(project, project.ID).Error
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
			if err := s.dns.DeleteRecord(ctx, cfToken, cfCred.ZoneID, rec.CloudflareRecordID); err != nil {
				cleanupErrs = append(cleanupErrs, fmt.Errorf("delete DNS record %s: %w", rec.Name, err))
			}
		}
	}
	if err := s.k8s.DeleteNamespace(ctx, project.Namespace); err != nil {
		cleanupErrs = append(cleanupErrs, fmt.Errorf("delete namespace %s: %w", project.Namespace, err))
	}
	if client, err := s.registry.GetClientForInstance(project.BasaltPassInstanceID); err == nil && project.BasaltAppID != 0 {
		if err := client.DeleteApp(ctx, project.BasaltAppID); err != nil {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("delete BasaltPass app %d: %w", project.BasaltAppID, err))
		}
	} else if err != nil {
		cleanupErrs = append(cleanupErrs, fmt.Errorf("load BasaltPass client: %w", err))
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

func normalizeProjectRequest(req *dto.CreateProjectRequest) {
	if req.GitHubBranch == "" {
		req.GitHubBranch = "main"
	}
	if req.DockerfilePath == "" {
		req.DockerfilePath = "Dockerfile"
	}
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
