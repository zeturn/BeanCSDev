package model

import "time"

type Deployment struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProjectID     uint      `gorm:"not null;index" json:"project_id"`
	Tag           string    `gorm:"size:256" json:"tag"`
	ImageDigest   string    `gorm:"size:256" json:"image_digest"`
	CommitSHA     string    `gorm:"size:64" json:"commit_sha"`
	ImageRef      string    `gorm:"size:512" json:"image_ref"`
	WorkflowRunID int64     `gorm:"index" json:"workflow_run_id"`
	WorkflowURL   string    `gorm:"size:512" json:"workflow_url"`
	FailureReason string    `gorm:"size:1000" json:"failure_reason"`
	Status        string    `gorm:"size:32" json:"status"`
	TriggeredBy   string    `gorm:"size:128" json:"triggered_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
