package service

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	jobs := []processJobSpec{
		{Name: "validate", DisplayName: "Validate package"},
		{Name: "network", DisplayName: "Prepare Cloudflare and network"},
		{Name: "github_workflow", DisplayName: "Add GitHub workflow"},
		{Name: "github_dispatch", DisplayName: "Trigger GitHub workflow"},
		{Name: "argocd", DisplayName: "Start Argo CD sync"},
		{Name: "rollout", DisplayName: "Pull image and deploy"},
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
	}).Order("processes.created_at desc").Limit(100)
	if userID != "" {
		q = q.Joins("JOIN projects ON projects.id = processes.project_id").Where("projects.owner_id = ?", userID)
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
	r := &processRun{svc: s, ctx: ctx, processID: processID, project: &project, deployment: &deployment}
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
	svc        *ProcessService
	ctx        context.Context
	processID  uint
	project    *model.Project
	deployment *model.Deployment
	token      string
	cred       model.GitHubCredential
	owner      string
	repo       string
	image      string
	runID      int64
}

func (r *processRun) run() error {
	if err := r.validate(); err != nil {
		return err
	}
	if err := r.network(); err != nil {
		return err
	}
	if r.project.BuildSource == model.BuildSourceGitHub {
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
	return r.rollout()
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
			if err := r.svc.k8s.UpsertRegistryPullSecret(r.ctx, r.project.Namespace, r.project.Name, r.project.RegistryPullSecretName); err != nil {
				return r.svc.failJob(r.ctx, job, err.Error())
			}
			r.svc.appendJobLog(r.ctx, job, "registry pull secret reconciled")
		}
	}
	r.image = strings.TrimSpace(r.deployment.ImageRef)
	if r.image == "" {
		r.image = strings.TrimSpace(r.deployment.Tag)
	}
	if r.image == "" && r.project.BuildSource == model.BuildSourceGitHub {
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
		if err := r.svc.k8s.ApplyNetworkPoliciesForPorts(r.ctx, r.project.Namespace, r.project.Name, r.project.Ports); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		if err := r.svc.k8s.ApplyServicePorts(r.ctx, r.project.Namespace, r.project.Name, r.project.Ports); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		if err := r.svc.k8s.ApplyIngressPorts(r.ctx, r.project.Namespace, r.project.Name, r.project.Ports); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
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
			token, err := r.svc.credentials.DecryptCloudflareToken(cred)
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
	r.svc.appendJobLog(r.ctx, job, "workflow file ensured: "+beancsBuildWorkflowPath)
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
	run, err := r.svc.build.waitForRun(r.ctx, r.token, r.owner, r.repo, branch, dispatchedAt)
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
	r.project.ImageReference = r.image
	if err := r.svc.db.WithContext(r.ctx).Model(r.project).Update("image_reference", r.image).Error; err != nil {
		return r.svc.failJob(r.ctx, job, err.Error())
	}
	if r.svc.gitops != nil {
		if err := r.svc.gitops.CommitProjectManifests(r.ctx, r.token, r.cred, r.project); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		r.svc.appendJobLog(r.ctx, job, "GitOps manifests committed")
	}
	if r.svc.k8s != nil && r.cred.GitOpsRepo != "" {
		if err := r.svc.k8s.ApplyArgoCDApplication(r.ctx, r.project.Name, gitOpsRepoURL(r.cred), fmt.Sprintf("apps/%s/overlays/dev", r.project.Name), r.project.Namespace); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		r.svc.appendJobLog(r.ctx, job, "Argo CD application reconciled")
	}
	_ = r.svc.db.WithContext(r.ctx).Model(r.deployment).Updates(map[string]any{"status": "deploying", "image_ref": r.image, "failure_reason": ""}).Error
	return r.svc.finishJob(r.ctx, job, model.ProcessStatusSucceeded, "")
}

func (r *processRun) rollout() error {
	job, err := r.svc.startJob(r.ctx, r.processID, "rollout")
	if err != nil {
		return err
	}
	resources := model.ResourcePresets[r.project.ResourcePreset]
	if resources.CPURequest == "" {
		resources = model.ResourcePresets["small"]
	}
	if r.svc.k8s != nil {
		if err := r.svc.k8s.ApplyDeploymentPortsWithPullSecret(r.ctx, r.project.Namespace, r.project.Name, r.image, r.project.Ports, int32(r.project.Replicas), resources.CPURequest, resources.CPULimit, resources.MemRequest, resources.MemLimit, r.project.RegistryPullSecretName); err != nil {
			return r.svc.failJob(r.ctx, job, err.Error())
		}
		r.svc.appendJobLog(r.ctx, job, fmt.Sprintf("deployment applied %s/%s image=%s", r.project.Namespace, r.project.Name, r.image))
	}
	_ = r.svc.db.WithContext(r.ctx).Model(r.deployment).Updates(map[string]any{"status": "running", "image_ref": r.image, "failure_reason": ""}).Error
	return r.svc.finishJob(r.ctx, job, model.ProcessStatusSucceeded, "")
}
