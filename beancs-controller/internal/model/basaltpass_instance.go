package model

import "time"

type BasaltPassInstance struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"size:128;not null" json:"name"`
	BaseURL         string    `gorm:"size:512;not null" json:"base_url"`
	ClientID        string    `gorm:"size:256;not null" json:"client_id"`
	ClientSecretEnc []byte    `gorm:"type:bytea;not null" json:"-"`
	ServiceTokenEnc []byte    `gorm:"type:bytea" json:"-"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatedBy       string    `gorm:"size:128;not null;index" json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
