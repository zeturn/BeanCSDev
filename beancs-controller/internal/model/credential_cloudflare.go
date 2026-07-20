package model

import "time"

type CloudflareCredential struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Name            string     `gorm:"size:128;not null" json:"name"`
	APITokenEnc     []byte     `gorm:"type:bytea;not null" json:"-"`
	RefreshTokenEnc []byte     `gorm:"type:bytea" json:"-"`
	TokenExpiresAt  *time.Time `json:"token_expires_at,omitempty"`
	AuthType        string     `gorm:"size:32;not null;default:api_token" json:"auth_type"`
	ZoneID          string     `gorm:"size:128" json:"zone_id"`
	Domain          string     `gorm:"size:256" json:"domain"`
	AccountID       string     `gorm:"size:128" json:"account_id,omitempty"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`
	CreatedBy       string     `gorm:"size:128;not null;index" json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CloudflareDomainCache struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	CloudflareCredentialID uint      `gorm:"not null;uniqueIndex:idx_cf_domain_cache_credential_zone;index" json:"cloudflare_credential_id"`
	ZoneID                 string    `gorm:"size:128;not null;uniqueIndex:idx_cf_domain_cache_credential_zone" json:"zone_id"`
	Domain                 string    `gorm:"size:256;not null;index" json:"domain"`
	AccountID              string    `gorm:"size:128;index" json:"account_id,omitempty"`
	Status                 string    `gorm:"size:32" json:"status,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}
