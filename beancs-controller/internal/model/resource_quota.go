package model

type ResourceQuota struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	TeamID        string `gorm:"uniqueIndex;size:128;not null" json:"team_id"`
	MaxProjects   int    `gorm:"default:10" json:"max_projects"`
	MaxCPUMillis  int    `gorm:"default:2000" json:"max_cpu_millis"`
	MaxMemoryMB   int    `gorm:"default:4096" json:"max_memory_mb"`
	UsedProjects  int    `gorm:"default:0" json:"used_projects"`
	UsedCPUMillis int    `gorm:"default:0" json:"used_cpu_millis"`
	UsedMemoryMB  int    `gorm:"default:0" json:"used_memory_mb"`
}
