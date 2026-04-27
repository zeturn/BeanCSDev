package model

import "time"

type GitHubCredential struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"size:128;not null" json:"name"`
	TokenEnc   []byte    `gorm:"type:bytea;not null" json:"-"`
	Org        string    `gorm:"size:128" json:"org,omitempty"`
	GitOpsRepo string    `gorm:"size:256;not null" json:"gitops_repo"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedBy  string    `gorm:"size:128;not null;index" json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
