package model

import "time"

type CloudflareCredential struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	APITokenEnc []byte    `gorm:"type:bytea;not null" json:"-"`
	ZoneID      string    `gorm:"size:128" json:"zone_id"`
	Domain      string    `gorm:"size:256" json:"domain"`
	AccountID   string    `gorm:"size:128" json:"account_id,omitempty"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedBy   string    `gorm:"size:128;not null;index" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
