package model

import "time"

type APIKey struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     string     `gorm:"size:128;not null;index" json:"user_id"`
	TenantID   string     `gorm:"size:128;index" json:"tenant_id,omitempty"`
	Name       string     `gorm:"size:128;not null" json:"name"`
	Prefix     string     `gorm:"size:32;not null;uniqueIndex" json:"prefix"`
	Hash       string     `gorm:"size:64;not null" json:"-"`
	Scopes     string     `gorm:"size:512" json:"scopes"`
	LastUsedAt *time.Time `gorm:"index" json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `gorm:"index" json:"expires_at,omitempty"`
	RevokedAt  *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
