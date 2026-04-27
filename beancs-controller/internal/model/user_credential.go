package model

import "time"

const (
	CredentialTypeCloudflare = "cloudflare"
	CredentialTypeGitHub     = "github"
	CredentialTypeBasaltPass = "basaltpass"
	CredentialRoleOwner      = "owner"
	CredentialRoleUser       = "user"
)

type UserCredential struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         string    `gorm:"size:128;not null;index:idx_user_cred,unique" json:"user_id"`
	CredentialType string    `gorm:"size:32;not null;index:idx_user_cred,unique" json:"credential_type"`
	CredentialID   uint      `gorm:"not null;index:idx_user_cred,unique" json:"credential_id"`
	Role           string    `gorm:"size:32;default:'owner'" json:"role"`
	CreatedAt      time.Time `json:"created_at"`
}
