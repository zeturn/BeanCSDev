package migration

import (
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.CloudflareCredential{},
		&model.GitHubCredential{},
		&model.BasaltPassInstance{},
		&model.UserCredential{},
		&model.Project{},
		&model.Deployment{},
		&model.Process{},
		&model.ProcessJob{},
		&model.DNSRecord{},
		&model.ResourceQuota{},
		&model.AuditLog{},
		&model.APIKey{},
		&model.ContainerRegistry{},
		&model.ContainerImage{},
	); err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE git_hub_credentials ALTER COLUMN token_enc DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE git_hub_credentials ALTER COLUMN git_ops_repo DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE cloudflare_credentials ALTER COLUMN zone_id DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE cloudflare_credentials ALTER COLUMN domain DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE projects ALTER COLUMN basalt_pass_instance_id DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE projects ALTER COLUMN git_hub_credential_id DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE projects ALTER COLUMN git_hub_repo DROP NOT NULL").Error; err != nil {
		return err
	}
	return db.Exec("UPDATE projects SET auto_deploy = TRUE WHERE auto_deploy IS NULL").Error
}
