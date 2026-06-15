package migration

import (
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.CloudflareCredential{},
		&model.CloudflareDomainCache{},
		&model.GitHubCredential{},
		&model.BasaltPassInstance{},
		&model.UserCredential{},
		&model.Application{},
		&model.ManagedDependency{},
		&model.DependencyCredential{},
		&model.ApplicationComponent{},
		&model.DependencyDefinitionRecord{},
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
	if err := db.Exec("ALTER TABLE dns_records ALTER COLUMN cloudflare_zone_id DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE projects ALTER COLUMN basalt_pass_instance_id DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE basalt_pass_instances ALTER COLUMN client_id DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE basalt_pass_instances ALTER COLUMN client_secret_enc DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE projects ALTER COLUMN git_hub_credential_id DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE projects ALTER COLUMN git_hub_repo DROP NOT NULL").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE dns_records ALTER COLUMN proxied SET DEFAULT FALSE").Error; err != nil {
		return err
	}
	if err := db.Exec("UPDATE managed_dependencies SET controlled = TRUE WHERE external = FALSE").Error; err != nil {
		return err
	}
	if err := db.Exec("UPDATE managed_dependencies SET controlled = FALSE WHERE external = TRUE AND (config->>'admin_password' IS NULL OR config->>'admin_password' = '')").Error; err != nil {
		return err
	}
	return db.Exec("UPDATE projects SET auto_deploy = TRUE WHERE auto_deploy IS NULL").Error
}
