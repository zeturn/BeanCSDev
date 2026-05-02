package service

import (
	"context"
	"errors"
	"fmt"
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
}

func NewDeploymentService(db *gorm.DB, build *GitHubBuildService, credentials *CredentialService, gitops *GitOpsService) *DeploymentService {
	return &DeploymentService{db: db, build: build, credentials: credentials, gitops: gitops}
}

func (s *DeploymentService) Create(ctx context.Context, projectID uint, tag, commit, triggeredBy string) (*model.Deployment, error) {
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, projectID).Error; err == nil && s.build != nil && project.BuildSource == model.BuildSourceGitHub {
		return s.build.Start(ctx, &project, triggeredBy)
	}
	dep := &model.Deployment{ProjectID: projectID, Tag: tag, CommitSHA: commit, Status: "pending", TriggeredBy: triggeredBy}
	return dep, s.db.WithContext(ctx).Create(dep).Error
}

func (s *DeploymentService) List(ctx context.Context, projectID uint) ([]model.Deployment, error) {
	var out []model.Deployment
	err := s.db.WithContext(ctx).Where("project_id = ?", projectID).Order("created_at desc").Find(&out).Error
	return out, err
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
		return s.gitops.CommitProjectManifests(ctx, token, cred, &p)
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
