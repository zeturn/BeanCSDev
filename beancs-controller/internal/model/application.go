package model

import "time"

const (
	ApplicationTypeSingle   = "single"
	ApplicationTypeMonorepo = "monorepo"

	ApplicationStatusCreating       = "creating"
	ApplicationStatusActive         = "active"
	ApplicationStatusPartialFailed  = "partial_failed"
	ApplicationStatusDeleting       = "deleting"
	ApplicationStatusPartialDeleted = "partial_deleted"
	ApplicationStatusDeleted        = "deleted"
)

type Application struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"uniqueIndex;size:63;not null" json:"name"`
	DisplayName  string    `gorm:"size:256" json:"display_name"`
	Type         string    `gorm:"size:32;not null;default:'single'" json:"type"`
	GitHubRepo   string    `gorm:"size:256" json:"github_repo"`
	GitHubBranch string    `gorm:"size:128;default:'main'" json:"github_branch"`
	Namespace    string    `gorm:"size:128" json:"namespace,omitempty"`
	SpecPath     string    `gorm:"size:512" json:"spec_path,omitempty"`
	SpecRaw      JSONMap   `gorm:"type:jsonb" json:"spec_raw,omitempty"`
	SpecHash     string    `gorm:"size:128" json:"spec_hash,omitempty"`
	OwnerID      string    `gorm:"size:128;not null;index" json:"owner_id"`
	TenantID     string    `gorm:"size:128;index" json:"tenant_id"`
	TenantCode   string    `gorm:"size:128;index,omitempty" json:"tenant_code,omitempty"`
	Status       string    `gorm:"size:32;not null;default:'creating'" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
