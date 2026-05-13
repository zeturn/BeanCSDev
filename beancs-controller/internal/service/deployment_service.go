package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

type DeploymentService struct {
	db          *gorm.DB
	build       *GitHubBuildService
	credentials *CredentialService
	gitops      *GitOpsService
	processes   *ProcessService
}

func NewDeploymentService(db *gorm.DB, build *GitHubBuildService, credentials *CredentialService, gitops *GitOpsService, processes *ProcessService) *DeploymentService {
	return &DeploymentService{db: db, build: build, credentials: credentials, gitops: gitops, processes: processes}
}

func (s *DeploymentService) Create(ctx context.Context, projectID uint, tag, commit, triggeredBy string) (*model.Deployment, error) {
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, projectID).Error; err != nil {
		return nil, err
	}
	dep := &model.Deployment{ProjectID: projectID, Tag: tag, CommitSHA: commit, Status: "queued", TriggeredBy: triggeredBy}
	if project.BuildSource == model.BuildSourceGitHub {
		image := buildImageReference(&project)
		dep.Tag = coalesce(tag, image)
		dep.ImageRef = image
		dep.CommitSHA = coalesce(commit, project.GitHubBranch)
	}
	if err := s.db.WithContext(ctx).Create(dep).Error; err != nil {
		return nil, err
	}
	if s.processes != nil {
		process, err := s.processes.CreateDeploymentProcess(ctx, dep, triggeredBy)
		if err != nil {
			_ = s.db.WithContext(ctx).Model(dep).Updates(map[string]any{"status": "failed", "failure_reason": truncateFailure(err.Error())}).Error
			return nil, err
		}
		s.processes.Start(process.ID)
	}
	return dep, nil
}

func (s *DeploymentService) List(ctx context.Context, projectID uint) ([]model.Deployment, error) {
	var out []model.Deployment
	err := s.db.WithContext(ctx).Where("project_id = ?", projectID).Order("created_at desc").Find(&out).Error
	return out, err
}

func (s *DeploymentService) Logs(ctx context.Context, project model.Project, deploymentID uint) (string, error) {
	var dep model.Deployment
	if err := s.db.WithContext(ctx).Where("project_id = ? AND id = ?", project.ID, deploymentID).First(&dep).Error; err != nil {
		return "", err
	}
	if dep.WorkflowRunID == 0 {
		return deploymentRecordLog(dep), nil
	}
	if project.GitHubCredentialID == 0 || strings.TrimSpace(project.GitHubRepo) == "" {
		return deploymentRecordLog(dep), nil
	}
	if s.credentials == nil {
		return deploymentRecordLog(dep), nil
	}
	var cred model.GitHubCredential
	if err := s.db.WithContext(ctx).First(&cred, project.GitHubCredentialID).Error; err != nil {
		return "", err
	}
	token, err := s.credentials.GitHubToken(ctx, cred)
	if err != nil {
		return "", err
	}
	owner, repo, ok := splitRepo(project.GitHubRepo)
	if !ok {
		return "", fmt.Errorf("github_repo must be in owner/repo format")
	}
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs/%d/logs", url.PathEscape(owner), url.PathEscape(repo), dep.WorkflowRunID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GitHub Actions logs request failed: %s", strings.TrimSpace(string(raw)))
	}
	return unzipWorkflowLogs(raw)
}

func (s *DeploymentService) HandleGitHubWebhook(ctx context.Context, req dto.GitHubWebhookRequest) error {
	var p model.Project
	if err := s.db.WithContext(ctx).Where("name = ?", req.Project).First(&p).Error; err != nil {
		return err
	}
	status := "deploying"
	if req.Status != "success" {
		status = "failed"
	}
	imageRef := webhookImageReference(p, req.Tag)
	var dep model.Deployment
	tx := s.db.WithContext(ctx).Where("project_id = ? AND status = ? AND (image_ref = ? OR tag = ? OR commit_sha = ?)", p.ID, "building", imageRef, req.Tag, req.Commit).Order("created_at desc").First(&dep)
	if tx.Error != nil {
		if !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return tx.Error
		}
		if !p.AutoDeploy {
			return nil
		}
		dep = model.Deployment{ProjectID: p.ID, Tag: req.Tag, ImageDigest: req.Digest, CommitSHA: req.Commit, ImageRef: imageRef, Status: status, TriggeredBy: "webhook"}
		if req.Status != "success" {
			dep.FailureReason = "github workflow " + req.Status
		}
		if err := s.db.WithContext(ctx).Create(&dep).Error; err != nil {
			return err
		}
	} else {
		updates := map[string]any{"status": status, "tag": req.Tag, "image_digest": req.Digest, "commit_sha": req.Commit, "image_ref": imageRef}
		if req.Status != "success" {
			updates["failure_reason"] = "github workflow " + req.Status
		}
		if err := s.db.WithContext(ctx).Model(&dep).Updates(updates).Error; err != nil {
			return err
		}
	}
	if req.Status == "success" && imageRef != "" && s.credentials != nil && s.gitops != nil && p.GitHubCredentialID != 0 {
		p.ImageReference = imageRef
		if err := s.db.WithContext(ctx).Model(&p).Update("image_reference", imageRef).Error; err != nil {
			return err
		}
		var cred model.GitHubCredential
		if err := s.db.WithContext(ctx).First(&cred, p.GitHubCredentialID).Error; err != nil {
			return err
		}
		token, err := s.credentials.GitHubToken(ctx, cred)
		if err != nil {
			return err
		}
		return s.gitops.UpdateImageTag(ctx, token, cred, &p, imageRef)
	}
	return nil
}

func webhookImageReference(project model.Project, tag string) string {
	imageRef := strings.TrimSpace(tag)
	if imageRef == "" || strings.Contains(imageRef, "/") {
		return imageRef
	}
	base := strings.TrimSpace(project.ImageReference)
	if base == "" && project.GitHubRepo != "" {
		base = "ghcr.io/" + strings.ToLower(project.GitHubRepo)
	}
	if base == "" {
		return imageRef
	}
	if strings.Contains(base, "@") {
		base = strings.Split(base, "@")[0]
	}
	lastSlash := strings.LastIndex(base, "/")
	lastColon := strings.LastIndex(base, ":")
	if lastColon > lastSlash {
		base = base[:lastColon]
	}
	return base + ":" + imageRef
}

func deploymentRecordLog(dep model.Deployment) string {
	lines := []string{
		fmt.Sprintf("deployment_id=%d", dep.ID),
		"status=" + coalesce(dep.Status, "pending"),
		"tag=" + coalesce(dep.Tag, "-"),
		"image=" + coalesce(dep.ImageRef, "-"),
		"commit=" + coalesce(dep.CommitSHA, "-"),
	}
	if dep.WorkflowURL != "" {
		lines = append(lines, "workflow="+dep.WorkflowURL)
	}
	if dep.FailureReason != "" {
		lines = append(lines, "error="+dep.FailureReason)
	}
	if dep.WorkflowRunID == 0 {
		lines = append(lines, "note=no GitHub Actions workflow run is attached to this deployment record")
	}
	return strings.Join(lines, "\n") + "\n"
}

func unzipWorkflowLogs(raw []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return string(raw), nil
	}
	var out strings.Builder
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			fmt.Fprintf(&out, "==> %s <==\nlog file unavailable: %s\n\n", file.Name, err.Error())
			continue
		}
		content, _ := io.ReadAll(io.LimitReader(rc, 2<<20))
		_ = rc.Close()
		fmt.Fprintf(&out, "==> %s <==\n%s\n\n", file.Name, strings.TrimRight(string(content), "\n"))
	}
	if out.Len() == 0 {
		return "GitHub Actions log archive was empty.\n", nil
	}
	return out.String(), nil
}

func (s *DeploymentService) HandleArgoCDWebhook(ctx context.Context, req dto.ArgoCDWebhookRequest) error {
	var p model.Project
	if err := s.db.WithContext(ctx).Where("name = ?", req.Project).First(&p).Error; err != nil {
		return err
	}
	status := "deploying"
	if req.SyncStatus == "Synced" && req.HealthStatus == "Healthy" {
		status = "running"
	}
	tx := s.db.WithContext(ctx).Model(&model.Deployment{}).Where("project_id = ?", p.ID).Order("created_at desc").Limit(1).Update("status", status)
	if tx.RowsAffected == 0 {
		return fmt.Errorf("deployment not found")
	}
	return tx.Error
}
