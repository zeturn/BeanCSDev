package service

import (
	"context"
	"fmt"

	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

type DeploymentService struct {
	db *gorm.DB
}

func NewDeploymentService(db *gorm.DB) *DeploymentService { return &DeploymentService{db: db} }

func (s *DeploymentService) Create(ctx context.Context, projectID uint, tag, commit, triggeredBy string) (*model.Deployment, error) {
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
	dep := model.Deployment{ProjectID: p.ID, Tag: req.Tag, ImageDigest: req.Digest, CommitSHA: req.Commit, Status: status, TriggeredBy: "webhook"}
	return s.db.WithContext(ctx).Create(&dep).Error
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
