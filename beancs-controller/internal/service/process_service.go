package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zeturn/beancs-controller/internal/config"
	"github.com/zeturn/beancs-controller/internal/k8s"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

type ProcessService struct {
	db          *gorm.DB
	build       *GitHubBuildService
	credentials *CredentialService
	gitops      *GitOpsService
	dns         *DNSService
	k8s         *k8s.Manager
}

type processJobSpec struct {
	Name        string
	DisplayName string
}

func NewProcessService(db *gorm.DB, build *GitHubBuildService, credentials *CredentialService, gitops *GitOpsService, dns *DNSService, k8sManager *k8s.Manager) *ProcessService {
	return &ProcessService{db: db, build: build, credentials: credentials, gitops: gitops, dns: dns, k8s: k8sManager}
}

func (s *ProcessService) CreateDeploymentProcess(ctx context.Context, deployment *model.Deployment, triggeredBy string) (*model.Process, error) {
	jobs := []processJobSpec{
		{Name: "validate", DisplayName: "Validate package"},
		{Name: "network", DisplayName: "Prepare Cloudflare and network"},
		{Name: "github_workflow", DisplayName: "Add GitHub workflow"},
		{Name: "github_dispatch", DisplayName: "Trigger GitHub workflow"},
		{Name: "argocd", DisplayName: "Start Argo CD sync"},
		{Name: "rollout", DisplayName: "Pull image and deploy"},
		{Name: "connectivity", DisplayName: "Verify connectivity"},
	}
	return s.createDeploymentProcess(ctx, deployment, triggeredBy, jobs)
}

func (s *ProcessService) CreateWebhookRolloutProcess(ctx context.Context, deployment *model.Deployment, triggeredBy string) (*model.Process, error) {
	jobs := []processJobSpec{
		{Name: "validate", DisplayName: "Validate package"},
		{Name: "network", DisplayName: "Prepare Cloudflare and network"},
		{Name: "argocd", DisplayName: "Start Argo CD sync"},
		{Name: "rollout", DisplayName: "Pull image and deploy"},
		{Name: "connectivity", DisplayName: "Verify connectivity"},
	}
	return s.createDeploymentProcess(ctx, deployment, triggeredBy, jobs)
}

func (s *ProcessService) createDeploymentProcess(ctx context.Context, deployment *model.Deployment, triggeredBy string, jobs []processJobSpec) (*model.Process, error) {
	if deployment == nil || deployment.ProjectID == 0 {
		return nil, fmt.Errorf("deployment is required")
	}
	process := &model.Process{
		Type:         model.ProcessTypeDeployment,
		Status:       model.ProcessStatusQueued,
		ProjectID:    deployment.ProjectID,
		DeploymentID: deployment.ID,
		Title:        fmt.Sprintf("Deployment #%d", deployment.ID),
		TriggeredBy:  triggeredBy,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(process).Error; err != nil {
			return err
		}
		for i, spec := range jobs {
			job := model.ProcessJob{ProcessID: process.ID, Name: spec.Name, DisplayName: spec.DisplayName, Status: model.ProcessStatusQueued, StepIndex: i}
			if err := tx.Create(&job).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, process.ID)
}

func (s *ProcessService) Start(processID uint) {
	go s.runDeploymentProcess(context.Background(), processID)
}

func (s *ProcessService) List(ctx context.Context, userID string) ([]model.Process, error) {
	var out []model.Process
	q := s.db.WithContext(ctx).Preload("Project").Preload("Deployment").Preload("Jobs", func(db *gorm.DB) *gorm.DB {
		return db.Order("step_index asc")
	}).Joins("LEFT JOIN projects ON projects.id = processes.project_id").Order("processes.created_at desc").Limit(100)
	if userID != "" {
		q = q.Where("projects.owner_id = ? OR processes.owner_id = ?", userID, userID)
	}
	return out, q.Find(&out).Error
}

func (s *ProcessService) Get(ctx context.Context, processID uint) (*model.Process, error) {
	var out model.Process
	err := s.db.WithContext(ctx).Preload("Project").Preload("Deployment").Preload("Jobs", func(db *gorm.DB) *gorm.DB {
		return db.Order("step_index asc")
	}).First(&out, processID).Error
	return &out, err
}

func (s *ProcessService) LatestForDeployment(ctx context.Context, deploymentID uint) (*model.Process, error) {
	var out model.Process
	err := s.db.WithContext(ctx).Preload("Jobs", func(db *gorm.DB) *gorm.DB {
		return db.Order("step_index asc")
	}).Where("deployment_id = ?", deploymentID).Order("created_at desc").First(&out).Error
	return &out, err
}

func (s *ProcessService) runDeploymentProcess(ctx context.Context, processID uint) {
	started := time.Now().UTC()
	_ = s.db.WithContext(ctx).Model(&model.Process{}).Where("id = ?", processID).Updates(map[string]any{"status": model.ProcessStatusRunning, "started_at": &started}).Error
	var process model.Process
	if err := s.db.WithContext(ctx).First(&process, processID).Error; err != nil {
		return
	}
	var deployment model.Deployment
	if err := s.db.WithContext(ctx).First(&deployment, process.DeploymentID).Error; err != nil {
		s.failProcess(ctx, processID, err)
		return
	}
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, deployment.ProjectID).Error; err != nil {
		s.failProcess(ctx, processID, err)
		return
	}
	r := &processRun{svc: s, ctx: ctx, processID: processID, process: &process, project: &project, deployment: &deployment}
	if err := r.run(); err != nil {
		s.failProcess(ctx, processID, err)
		return
	}
	finished := time.Now().UTC()
	_ = s.db.WithContext(ctx).Model(&model.Process{}).Where("id = ?", processID).Updates(map[string]any{
		"status": model.ProcessStatusSucceeded, "finished_at": &finished, "failure_reason": "",
	}).Error
}

func (s *ProcessService) failProcess(ctx context.Context, processID uint, err error) {
	finished := time.Now().UTC()
	msg := truncateFailure(err.Error())
	_ = s.db.WithContext(ctx).Model(&model.Process{}).Where("id = ?", processID).Updates(map[string]any{"status": model.ProcessStatusFailed, "finished_at": &finished, "failure_reason": msg}).Error
	var job model.ProcessJob
	if e := s.db.WithContext(ctx).Where("process_id = ? AND status = ?", processID, model.ProcessStatusRunning).Order("step_index desc").First(&job).Error; e == nil {
		_ = s.finishJob(ctx, &job, model.ProcessStatusFailed, msg)
	}
}

func (s *ProcessService) startJob(ctx context.Context, processID uint, name string) (*model.ProcessJob, error) {
	now := time.Now().UTC()
	var job model.ProcessJob
	if err := s.db.WithContext(ctx).Where("process_id = ? AND name = ?", processID, name).First(&job).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&job).Updates(map[string]any{"status": model.ProcessStatusRunning, "started_at": &now}).Error; err != nil {
		return nil, err
	}
	job.Status = model.ProcessStatusRunning
	job.StartedAt = &now
	return &job, nil
}

func (s *ProcessService) appendJobLog(ctx context.Context, job *model.ProcessJob, line string) {
	if job == nil {
		return
	}
	entry := fmt.Sprintf("[%s] %s\n", time.Now().UTC().Format(time.RFC3339), strings.TrimRight(line, "\n"))
	job.Logs += entry
	_ = s.db.WithContext(ctx).Model(job).Update("logs", job.Logs).Error
}

func (s *ProcessService) finishJob(ctx context.Context, job *model.ProcessJob, status, failure string) error {
	if job == nil {
		return nil
	}
	now := time.Now().UTC()
	if failure != "" {
		s.appendJobLog(ctx, job, "ERROR: "+failure)
	}
	return s.db.WithContext(ctx).Model(job).Updates(map[string]any{"status": status, "finished_at": &now, "failure_reason": truncateFailure(failure)}).Error
}

func (s *ProcessService) failJob(ctx context.Context, job *model.ProcessJob, failure string) error {
	_ = s.finishJob(ctx, job, model.ProcessStatusFailed, failure)
	return fmt.Errorf("%s", failure)
}

type processRun struct {
	svc            *ProcessService
	ctx            context.Context
	processID      uint
	process        *model.Process
	project        *model.Project
	deployment     *model.Deployment
	token          string
	cred           model.GitHubCredential
	owner          string
	repo           string
	image          string
	runID          int64
	gitOpsRevision string
}

func (r *processRun) run() error {
	if err := r.validate(); err != nil {
		return err
	}
	if err := r.network(); err != nil {
		return err
	}
	if r.project.BuildSource == model.BuildSourceGitHub && r.hasJob("github_dispatch") {
		if err := r.githubWorkflow(); err != nil {
			return err
		}
		if err := r.githubDispatch(); err != nil {
			return err
		}
	}
	if err := r.argocd(); err != nil {
		return err
	}
	if err := r.rollout(); err != nil {
		return err
	}
	return r.connectivity()
}

func (r *processRun) hasJob(name string) bool {
	var count int64
	err := r.svc.db.WithContext(r.ctx).Model(&model.ProcessJob{}).Where("process_id = ? AND name = ?", r.processID, name).Count(&count).Error
	return err == nil && count > 0
}

func (r *processRun) validate() error {
	job, err := r.svc.startJob(r.ctx, r.processID, "validate")
	if err != nil {
		return err
	}
	r.svc.appendJobLog(r.ctx, job, fmt.Sprintf("project=%s namespace=%s source=%s", r.project.Name, r.project.Namespace, r.project.BuildSource))
	if r.project.Namespace == "" {
		return r.svc.failJob(r.ctx, job, "project namespace is empty")
	}
	if r.project.BuildSource == model.BuildSourceGitHub {
		if err := r.prepareGitHub(job); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		if err := configureBeanCSRegistry(r.project, r.svc.build.cfg, r.project.TenantCode); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		if err := ensureHarborProject(r.ctx, r.svc.build.cfg, r.project.RegistryProject); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		if err := r.svc.db.WithContext(r.ctx).Model(r.project).Updates(map[string]any{
			"image_reference":           r.project.ImageReference,
			"registry_host":             r.project.RegistryHost,
			"registry_project":          r.project.RegistryProject,
			"registry_repository":       r.project.RegistryRepository,
			"registry_image_reference":  r.project.RegistryImageReference,
			"registry_pull_secret_name": r.project.RegistryPullSecretName,
		}).Error; err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		path, err := r.resolveDockerfilePath(job)
		if err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		r.project.DockerfilePath = path
		r.svc.appendJobLog(r.ctx, job, "dockerfile_path="+path)
	}
	if r.svc.k8s != nil {
		if err := r.svc.k8s.CreateNamespace(r.ctx, r.project.Namespace, r.project.Name); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		r.svc.appendJobLog(r.ctx, job, "namespace exists or was created")
		if r.project.BuildSource == model.BuildSourceGitHub && r.project.RegistryImageReference != "" {
			creds, err := ensureHarborPullRobot(r.ctx, r.svc.build.cfg, r.project)
			if err != nil {
				return r.svc.failJob(r.ctx, job, err.Error())
			}
			if err := r.svc.k8s.UpsertRegistryPullSecretWithCredentials(r.ctx, r.project.Namespace, r.project.Name, r.project.RegistryPullSecretName, creds.Host, creds.Username, creds.Token); err != nil {
				return r.svc.failJob(r.ctx, job, err.Error())
			}
			r.svc.appendJobLog(r.ctx, job, "registry pull secret reconciled")
		}
	}
	r.image = strings.TrimSpace(r.deployment.ImageRef)
	if r.image == "" {
		r.image = strings.TrimSpace(r.deployment.Tag)
	}
	if r.project.BuildSource == model.BuildSourceGitHub && r.hasJob("github_dispatch") {
		r.image = buildImageReference(r.project)
	}
	if r.project.BuildSource != model.BuildSourceGitHub && r.image == "" {
		r.image = strings.TrimSpace(r.project.ImageReference)
	}
	if r.image == "" {
		return r.svc.failJob(r.ctx, job, "image reference could not be resolved")
	}
	r.svc.appendJobLog(r.ctx, job, "resolved image="+r.image)
	return r.svc.finishJob(r.ctx, job, model.ProcessStatusSucceeded, "")
}

func (r *processRun) network() error {
	job, err := r.svc.startJob(r.ctx, r.processID, "network")
	if err != nil {
		return err
	}
	if r.svc.k8s != nil {
		result, err := r.svc.k8s.EnsureTraefikPodNetwork(r.ctx)
		if err != nil {
			return r.svc.failJob(r.ctx, job, "traefik reconcile failed: "+err.Error())
		}
		r.svc.appendJobLog(r.ctx, job, fmt.Sprintf("traefik=%s/%s updated=%t: %s", result.Namespace, result.Name, result.Updated, result.Message))
		if err := r.svc.k8s.ApplyNetworkPoliciesForPorts(r.ctx, r.project.Namespace, r.project.Name, r.project.Ports); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		if len(r.project.Ports) > 0 {
			if err := r.svc.k8s.ApplyServicePorts(r.ctx, r.project.Namespace, r.project.Name, r.project.Ports); err != nil {
				return r.svc.failJob(r.ctx, job, err.Error())
			}
			if err := r.svc.k8s.ApplyIngressPorts(r.ctx, r.project.Namespace, r.project.Name, r.project.Ports); err != nil {
				return r.svc.failJob(r.ctx, job, err.Error())
			}
		}
	}
	if r.project.CloudflareCredentialID != nil && r.svc.dns != nil && r.svc.credentials != nil && r.project.Domain != "" {
		var existing model.DNSRecord
		err := r.svc.db.WithContext(r.ctx).Where("project_id = ? AND name = ?", r.project.ID, r.project.Domain).First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		if err == gorm.ErrRecordNotFound {
			var cred model.CloudflareCredential
			if err := r.svc.db.WithContext(r.ctx).First(&cred, *r.project.CloudflareCredentialID).Error; err != nil {
				return r.svc.failJob(r.ctx, job, err.Error())
			}
			token, err := r.svc.credentials.CloudflareToken(r.ctx, cred)
			if err != nil {
				return r.svc.failJob(r.ctx, job, err.Error())
			}
			record, err := r.svc.dns.CreateRecordForHost(r.ctx, token, cred, r.project.Name, r.project.Domain)
			if err != nil {
				return r.svc.failJob(r.ctx, job, err.Error())
			}
			record.ProjectID = r.project.ID
			if err := r.svc.db.WithContext(r.ctx).Create(record).Error; err != nil {
				return r.svc.failJob(r.ctx, job, err.Error())
			}
			r.svc.appendJobLog(r.ctx, job, "cloudflare_dns_record="+record.Name)
		} else {
			r.svc.appendJobLog(r.ctx, job, "cloudflare_dns_record already exists: "+existing.Name)
			if existing.Proxied {
				var cred model.CloudflareCredential
				if err := r.svc.db.WithContext(r.ctx).First(&cred, *r.project.CloudflareCredentialID).Error; err != nil {
					return r.svc.failJob(r.ctx, job, err.Error())
				}
				token, err := r.svc.credentials.CloudflareToken(r.ctx, cred)
				if err != nil {
					return r.svc.failJob(r.ctx, job, err.Error())
				}
				if err := r.svc.dns.EnsureRecordDNSOnly(r.ctx, token, cred, existing); err != nil {
					return r.svc.failJob(r.ctx, job, "cloudflare_dns_record dns-only update failed: "+err.Error())
				}
				if err := r.svc.db.WithContext(r.ctx).Model(&existing).Update("proxied", false).Error; err != nil {
					return r.svc.failJob(r.ctx, job, err.Error())
				}
				r.svc.appendJobLog(r.ctx, job, "cloudflare_dns_record set to DNS only: "+existing.Name)
			}
		}
	}
	r.svc.appendJobLog(r.ctx, job, "service, ingress, and network policies reconciled")
	if r.project.Domain != "" {
		r.svc.appendJobLog(r.ctx, job, "route="+r.project.Domain)
	}
	return r.svc.finishJob(r.ctx, job, model.ProcessStatusSucceeded, "")
}

func (r *processRun) prepareGitHub(job *model.ProcessJob) error {
	if r.svc.build == nil || r.svc.credentials == nil {
		return fmt.Errorf("GitHub build service is not configured")
	}
	if r.project.GitHubCredentialID == 0 || strings.TrimSpace(r.project.GitHubRepo) == "" {
		return fmt.Errorf("project GitHub credential and repo are required")
	}
	if r.token != "" {
		return nil
	}
	if err := r.svc.db.WithContext(r.ctx).First(&r.cred, r.project.GitHubCredentialID).Error; err != nil {
		return err
	}
	token, err := r.svc.credentials.GitHubToken(r.ctx, r.cred)
	if err != nil {
		return err
	}
	owner, repo, ok := splitRepo(r.project.GitHubRepo)
	if !ok {
		return fmt.Errorf("github_repo must be in owner/repo format")
	}
	r.token, r.owner, r.repo = token, owner, repo
	r.svc.appendJobLog(r.ctx, job, "repo="+r.project.GitHubRepo)
	return nil
}

func (r *processRun) resolveDockerfilePath(job *model.ProcessJob) (string, error) {
	branch := coalesce(r.project.GitHubBranch, "main")
	if path := strings.TrimSpace(r.project.DockerfilePath); path != "" {
		exists, err := githubContentExists(r.ctx, r.token, r.owner, r.repo, path, branch)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("dockerfile_path %q was not found in %s on branch %s", path, r.project.GitHubRepo, branch)
		}
		return path, nil
	}
	for _, candidate := range dockerfileCandidates() {
		exists, err := githubContentExists(r.ctx, r.token, r.owner, r.repo, candidate, branch)
		if err != nil {
			return "", err
		}
		if exists {
			if err := r.svc.db.WithContext(r.ctx).Model(r.project).Update("dockerfile_path", candidate).Error; err != nil {
				return "", err
			}
			r.svc.appendJobLog(r.ctx, job, "discovered dockerfile_path="+candidate)
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no Dockerfile or Containerfile was found in %s on branch %s; add one or set dockerfile_path to the correct path before deploying", r.project.GitHubRepo, branch)
}

func (r *processRun) githubWorkflow() error {
	job, err := r.svc.startJob(r.ctx, r.processID, "github_workflow")
	if err != nil {
		return err
	}
	if err := r.prepareGitHub(job); err != nil {
		return r.svc.failJob(r.ctx, job, err.Error())
	}
	if err := r.svc.build.ensureWorkflow(r.ctx, r.token, r.owner, r.repo, r.project); err != nil {
		return r.svc.failJob(r.ctx, job, err.Error())
	}
	r.svc.appendJobLog(r.ctx, job, "workflow file ensured: "+beancsBuildWorkflowPath(r.project))
	return r.svc.finishJob(r.ctx, job, model.ProcessStatusSucceeded, "")
}

func (r *processRun) githubDispatch() error {
	job, err := r.svc.startJob(r.ctx, r.processID, "github_dispatch")
	if err != nil {
		return err
	}
	if err := r.prepareGitHub(job); err != nil {
		return r.svc.failJob(r.ctx, job, err.Error())
	}
	branch := coalesce(r.deployment.CommitSHA, coalesce(r.project.GitHubBranch, "main"))
	dispatchedAt := time.Now().UTC()
	if err := r.svc.build.dispatchWorkflow(r.ctx, r.token, r.owner, r.repo, branch, r.project, r.image); err != nil {
		return r.svc.failJob(r.ctx, job, err.Error())
	}
	r.svc.appendJobLog(r.ctx, job, "workflow dispatch accepted")
	run, err := r.svc.build.waitForRun(r.ctx, r.token, r.owner, r.repo, beancsBuildWorkflowFile(r.project), branch, dispatchedAt)
	if err != nil {
		return r.svc.failJob(r.ctx, job, err.Error())
	}
	r.runID = run.ID
	_ = r.svc.db.WithContext(r.ctx).Model(r.deployment).Updates(map[string]any{"status": "building", "image_ref": r.image, "workflow_run_id": run.ID, "workflow_url": run.HTMLURL}).Error
	r.svc.appendJobLog(r.ctx, job, fmt.Sprintf("workflow_run_id=%d", run.ID))
	for {
		current, err := r.svc.build.workflowRun(r.ctx, r.token, r.owner, r.repo, run.ID)
		if err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		r.svc.appendJobLog(r.ctx, job, fmt.Sprintf("github_status=%s conclusion=%s", current.Status, coalesce(current.Conclusion, "-")))
		if current.Status == "completed" {
			if current.Conclusion != "success" {
				return r.svc.failJob(r.ctx, job, "github workflow concluded with "+coalesce(current.Conclusion, "unknown"))
			}
			return r.svc.finishJob(r.ctx, job, model.ProcessStatusSucceeded, "")
		}
		select {
		case <-r.ctx.Done():
			return r.ctx.Err()
		case <-time.After(15 * time.Second):
		}
	}
}

func (r *processRun) argocd() error {
	job, err := r.svc.startJob(r.ctx, r.processID, "argocd")
	if err != nil {
		return err
	}
	if r.project.BuildSource != model.BuildSourceGitHub {
		r.svc.appendJobLog(r.ctx, job, "registry deployment does not need Argo CD")
		return r.svc.finishJob(r.ctx, job, model.ProcessStatusSucceeded, "")
	}
	if err := r.prepareGitHub(job); err != nil {
		return r.svc.failJob(r.ctx, job, err.Error())
	}
	if !UsesGitOps(r.project, r.cred) {
		r.svc.appendJobLog(r.ctx, job, "GitOps repo not configured; deployment will use direct Kubernetes apply path")
		_ = r.svc.db.WithContext(r.ctx).Model(r.deployment).Updates(map[string]any{"status": "deploying", "image_ref": r.image, "failure_reason": ""}).Error
		return r.svc.finishJob(r.ctx, job, model.ProcessStatusSucceeded, "")
	}
	r.project.ImageReference = r.image
	if err := r.svc.db.WithContext(r.ctx).Model(r.project).Update("image_reference", r.image).Error; err != nil {
		return r.svc.failJob(r.ctx, job, err.Error())
	}
	if err := r.syncApplicationDependencies(job); err != nil {
		return r.svc.failJob(r.ctx, job, err.Error())
	}
	if r.svc.gitops != nil {
		var revision string
		var err error
		if r.isWebhookRollout() {
			revision, err = r.svc.gitops.UpdateImageTag(r.ctx, r.token, r.cred, r.project, r.image)
		} else {
			revision, err = r.svc.gitops.CommitProjectManifests(r.ctx, r.token, r.cred, r.project)
		}
		if err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		r.gitOpsRevision = revision
		r.svc.appendJobLog(r.ctx, job, "GitOps manifests committed revision="+revision)
	}
	if r.svc.k8s != nil && r.cred.GitOpsRepo != "" {
		var cfg *config.Config
		if r.svc.build != nil {
			cfg = r.svc.build.cfg
		}
		appName, appPath, appNamespace, err := r.gitOpsApplicationTarget()
		if err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		if err := ensureArgoCDGitOpsRepository(r.ctx, r.svc.k8s, r.svc.gitops, r.svc.credentials, cfg, r.token, r.cred, appName); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		r.svc.appendJobLog(r.ctx, job, "Argo CD GitOps repository credentials reconciled")
		if err := r.svc.k8s.ApplyArgoCDApplication(r.ctx, appName, gitOpsRepoURL(r.cred), appPath, appNamespace); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		r.svc.appendJobLog(r.ctx, job, "Argo CD application reconciled")
	}
	_ = r.svc.db.WithContext(r.ctx).Model(r.deployment).Updates(map[string]any{"status": "deploying", "image_ref": r.image, "failure_reason": ""}).Error
	return r.svc.finishJob(r.ctx, job, model.ProcessStatusSucceeded, "")
}

func (r *processRun) syncApplicationDependencies(job *model.ProcessJob) error {
	if r.project.ApplicationID == nil || r.svc.gitops == nil {
		return nil
	}
	var app model.Application
	if err := r.svc.db.WithContext(r.ctx).First(&app, *r.project.ApplicationID).Error; err != nil {
		return err
	}
	var deps []model.ManagedDependency
	if err := r.svc.db.WithContext(r.ctx).Where("application_id = ?", app.ID).Order("name asc").Find(&deps).Error; err != nil {
		return err
	}
	for _, dep := range deps {
		if dep.DeployMethod == model.DependencyDeployMethodExternal {
			continue
		}
		if r.svc.k8s != nil {
			if err := r.svc.k8s.CreateNamespace(r.ctx, dep.Namespace, app.Name); err != nil {
				return err
			}
			if err := r.svc.k8s.UpsertSecret(r.ctx, dep.Namespace, dep.SecretName, app.Name, dependencySecretRuntimeData(dep)); err != nil {
				return err
			}
		}
		if err := r.svc.gitops.CommitDependencyManifests(r.ctx, r.token, r.cred, app, dep); err != nil {
			return err
		}
		if r.svc.k8s != nil && r.cred.GitOpsRepo != "" {
			if err := r.svc.k8s.ApplyArgoCDApplication(r.ctx, dependencyRootArgoApplicationName(app.Name, dep.Name), gitOpsRepoURL(r.cred), fmt.Sprintf("apps/%s/dependencies/%s", app.Name, dep.Name), dep.Namespace); err != nil {
				return err
			}
		}
		r.svc.appendJobLog(r.ctx, job, fmt.Sprintf("dependency reconciled %s/%s", dep.Namespace, dep.Name))
	}
	return nil
}

func (r *processRun) isWebhookRollout() bool {
	return r != nil && r.process != nil && r.process.TriggeredBy == "webhook"
}

func (r *processRun) gitOpsApplicationTarget() (string, string, string, error) {
	appName := r.project.Name
	namespace := r.project.Namespace
	if r.isWebhookRollout() && r.project.ApplicationID != nil && *r.project.ApplicationID != 0 {
		var app model.Application
		if err := r.svc.db.WithContext(r.ctx).First(&app, *r.project.ApplicationID).Error; err != nil {
			return "", "", "", err
		}
		if strings.TrimSpace(app.Name) != "" {
			appName = app.Name
		}
		if strings.TrimSpace(app.Namespace) != "" {
			namespace = app.Namespace
		}
	}
	return appName, fmt.Sprintf("apps/%s/overlays/dev", appName), namespace, nil
}

func (r *processRun) rollout() error {
	job, err := r.svc.startJob(r.ctx, r.processID, "rollout")
	if err != nil {
		return err
	}
	if UsesGitOps(r.project, r.cred) {
		if err := r.rolloutWithArgoCD(job); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
	} else {
		if err := r.rolloutDirectly(job); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
	}
	_ = r.svc.db.WithContext(r.ctx).Model(r.deployment).Updates(map[string]any{"status": "deploying", "image_ref": r.image, "failure_reason": ""}).Error
	return r.svc.finishJob(r.ctx, job, model.ProcessStatusSucceeded, "")
}

func (r *processRun) rolloutWithArgoCD(job *model.ProcessJob) error {
	if r.svc.k8s == nil {
		return fmt.Errorf("kubernetes manager is not configured")
	}
	if strings.TrimSpace(r.gitOpsRevision) == "" {
		return fmt.Errorf("gitops revision is empty")
	}
	appName, _, _, err := r.gitOpsApplicationTarget()
	if err != nil {
		return err
	}
	if r.isWebhookRollout() && appName != r.project.Name {
		if err := r.svc.k8s.WaitForArgoCDApplicationSync(r.ctx, appName, r.gitOpsRevision, 5*time.Minute); err != nil {
			return err
		}
	} else {
		if err := r.svc.k8s.WaitForArgoCDApplication(r.ctx, appName, r.gitOpsRevision, 5*time.Minute); err != nil {
			return err
		}
	}
	r.svc.appendJobLog(r.ctx, job, fmt.Sprintf("Argo CD synced revision=%s", r.gitOpsRevision))
	if err := r.svc.k8s.WaitForDeploymentRollout(r.ctx, r.project.Namespace, r.project.Name, 5*time.Minute); err != nil {
		return err
	}
	r.svc.appendJobLog(r.ctx, job, fmt.Sprintf("deployment rollout completed %s/%s", r.project.Namespace, r.project.Name))
	return nil
}

func (r *processRun) rolloutDirectly(job *model.ProcessJob) error {
	if r.svc.k8s == nil {
		return nil
	}
	resources := resolveResourcePreset(r.project.ResourcePreset)
	if err := r.svc.k8s.ApplyProjectDeployment(r.ctx, *r.project, r.image, resources); err != nil {
		return err
	}
	r.svc.appendJobLog(r.ctx, job, fmt.Sprintf("deployment applied %s/%s image=%s", r.project.Namespace, r.project.Name, r.image))
	return nil
}

func resolveResourcePreset(name string) model.ResourceSpec {
	resources := model.ResourcePresets[name]
	if resources.CPURequest == "" {
		resources = model.ResourcePresets["small"]
	}
	return resources
}

func (r *processRun) connectivity() error {
	job, err := r.svc.startJob(r.ctx, r.processID, "connectivity")
	if err != nil {
		return err
	}
	if r.svc.k8s != nil {
		checks, err := r.svc.k8s.WaitForProjectConnectivity(r.ctx, r.project.Namespace, r.project.Name, r.project.Ports, 2*time.Minute)
		for _, check := range checks {
			r.svc.appendJobLog(r.ctx, job, fmt.Sprintf("%s %s %s: %s", check.Status, check.Name, check.Target, check.Message))
		}
		if err != nil {
			_ = r.svc.db.WithContext(r.ctx).Model(r.deployment).Updates(map[string]any{"status": "failed", "failure_reason": truncateFailure(err.Error())}).Error
			return r.svc.failJob(r.ctx, job, err.Error())
		}
	}
	for _, publicURL := range r.publicRouteURLs() {
		status, err := waitForPublicRoute(r.ctx, publicURL, 2*time.Minute)
		r.svc.appendJobLog(r.ctx, job, fmt.Sprintf("public-route %s: %s", publicURL, status))
		if err != nil {
			_ = r.svc.db.WithContext(r.ctx).Model(r.deployment).Updates(map[string]any{"status": "failed", "failure_reason": truncateFailure(err.Error())}).Error
			return r.svc.failJob(r.ctx, job, err.Error())
		}
	}
	_ = r.svc.db.WithContext(r.ctx).Model(r.deployment).Updates(map[string]any{"status": "running", "failure_reason": ""}).Error
	return r.svc.finishJob(r.ctx, job, model.ProcessStatusSucceeded, "")
}

func (r *processRun) publicRouteURLs() []string {
	var urls []string
	for _, port := range r.project.Ports {
		if port.Exposure == model.ExposurePublic && strings.TrimSpace(port.Domain) != "" {
			urls = append(urls, "https://"+strings.TrimSpace(port.Domain)+"/")
		}
	}
	return urls
}

func waitForPublicRoute(ctx context.Context, url string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	var lastStatus string
	var lastErr error
	for {
		status, err := checkPublicRoute(ctx, url)
		if err == nil {
			return status, nil
		}
		lastStatus = status
		lastErr = err
		if time.Now().After(deadline) {
			if lastStatus != "" {
				return lastStatus, lastErr
			}
			return "", lastErr
		}
		select {
		case <-ctx.Done():
			return lastStatus, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func checkPublicRoute(ctx context.Context, url string) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	targets := []string{strings.TrimRight(url, "/") + "/health", url}
	var lastStatus string
	var lastErr error
	for _, target := range targets {
		status, err := checkHTTPRoute(reqCtx, target)
		if err == nil {
			return target + " " + status, nil
		}
		lastStatus = target + " " + status
		lastErr = err
	}
	return lastStatus, lastErr
}

func checkHTTPRoute(ctx context.Context, target string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 BeanCS connectivity check")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return resp.Status, fmt.Errorf("public route returned %s", resp.Status)
	}
	return resp.Status, nil
}
