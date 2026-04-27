package model

import "time"

type DNSRecord struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	ProjectID              uint      `gorm:"not null;index" json:"project_id"`
	CloudflareCredentialID uint      `gorm:"not null;index" json:"cloudflare_credential_id"`
	CloudflareRecordID     string    `gorm:"size:128;not null" json:"cloudflare_record_id"`
	Name                   string    `gorm:"size:256;not null" json:"name"`
	Type                   string    `gorm:"size:16;default:'A'" json:"type"`
	Content                string    `gorm:"size:256;not null" json:"content"`
	Proxied                bool      `gorm:"default:true" json:"proxied"`
	CreatedAt              time.Time `json:"created_at"`
}
