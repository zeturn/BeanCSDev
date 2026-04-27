package migration

import (
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.CloudflareCredential{},
		&model.GitHubCredential{},
		&model.BasaltPassInstance{},
		&model.UserCredential{},
		&model.Project{},
		&model.Deployment{},
		&model.DNSRecord{},
		&model.ResourceQuota{},
		&model.AuditLog{},
	)
}
