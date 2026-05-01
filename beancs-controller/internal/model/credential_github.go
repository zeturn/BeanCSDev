package model

import "time"

type GitHubCredential struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"size:128;not null" json:"name"`
	AuthType       string    `gorm:"size:32;default:'pat'" json:"auth_type"`
	TokenEnc       []byte    `gorm:"type:bytea" json:"-"`
	InstallationID int64     `gorm:"index" json:"installation_id,omitempty"`
	AccountLogin   string    `gorm:"size:128" json:"account_login,omitempty"`
	Org            string    `gorm:"size:128" json:"org,omitempty"`
	GitOpsRepo     string    `gorm:"size:256" json:"gitops_repo,omitempty"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	CreatedBy      string    `gorm:"size:128;not null;index" json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
