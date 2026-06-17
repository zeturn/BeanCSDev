package model

import "time"

const (
	ProcessTypeDeployment           = "deployment"
	ProcessTypeBasaltPassDeployment = "basaltpass_deployment"

	ProcessStatusQueued    = "queued"
	ProcessStatusRunning   = "running"
	ProcessStatusSucceeded = "succeeded"
	ProcessStatusFailed    = "failed"
)

type Process struct {
	ID            uint         `gorm:"primaryKey" json:"id"`
	Type          string       `gorm:"size:64;not null;index" json:"type"`
	Status        string       `gorm:"size:32;not null;index" json:"status"`
	ProjectID     uint         `gorm:"index" json:"project_id"`
	DeploymentID  uint         `gorm:"index" json:"deployment_id"`
	OwnerID       string       `gorm:"size:128;index" json:"owner_id,omitempty"`
	Title         string       `gorm:"size:256" json:"title"`
	TriggeredBy   string       `gorm:"size:128" json:"triggered_by"`
	FailureReason string       `gorm:"size:1000" json:"failure_reason"`
	StartedAt     *time.Time   `json:"started_at,omitempty"`
	FinishedAt    *time.Time   `json:"finished_at,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	Project       Project      `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Deployment    Deployment   `gorm:"foreignKey:DeploymentID" json:"deployment,omitempty"`
	Jobs          []ProcessJob `gorm:"foreignKey:ProcessID" json:"jobs,omitempty"`
}

type ProcessJob struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ProcessID     uint       `gorm:"not null;index" json:"process_id"`
	Name          string     `gorm:"size:128;not null" json:"name"`
	DisplayName   string     `gorm:"size:256" json:"display_name"`
	Status        string     `gorm:"size:32;not null;index" json:"status"`
	StepIndex     int        `gorm:"not null" json:"step_index"`
	Logs          string     `gorm:"type:text" json:"logs"`
	FailureReason string     `gorm:"size:1000" json:"failure_reason"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
