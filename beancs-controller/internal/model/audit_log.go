package model

import "time"

type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"size:128;index" json:"user_id"`
	Action    string    `gorm:"size:256;not null" json:"action"`
	Resource  string    `gorm:"size:128" json:"resource"`
	Status    int       `json:"status"`
	IP        string    `gorm:"size:64" json:"ip"`
	UserAgent string    `gorm:"size:512" json:"user_agent"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}
