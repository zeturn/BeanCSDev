package model

import "time"

type BasaltPassInstance struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Name               string    `gorm:"size:128;not null" json:"name"`
	BaseURL            string    `gorm:"size:512;not null" json:"base_url"`
	TenantID           string    `gorm:"size:128;index" json:"tenant_id,omitempty"`
	TenantCode         string    `gorm:"size:128;index" json:"tenant_code,omitempty"`
	ClientID           string    `gorm:"size:256" json:"client_id,omitempty"`
	ClientSecretEnc    []byte    `gorm:"type:bytea" json:"-"`
	ServiceTokenEnc    []byte    `gorm:"type:bytea" json:"-"`
	AutomationTokenEnc []byte    `gorm:"type:bytea" json:"-"`
	IsActive           bool      `gorm:"default:true" json:"is_active"`
	CreatedBy          string    `gorm:"size:128;not null;index" json:"created_by"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
