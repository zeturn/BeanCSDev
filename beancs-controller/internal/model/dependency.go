package model

import "time"

const (
	ApplicationComponentKindService    = "service"
	ApplicationComponentKindWorker     = "worker"
	ApplicationComponentKindFrontend   = "frontend"
	ApplicationComponentKindDependency = "dependency"

	DependencyDeployMethodHelm     = "helm"
	DependencyDeployMethodManifest = "manifest"
	DependencyDeployMethodOperator = "operator"
	DependencyDeployMethodExternal = "external"

	DependencyStatusCreating = "creating"
	DependencyStatusReady    = "ready"
	DependencyStatusFailed   = "failed"
	DependencyStatusDeleting = "deleting"
)

type ApplicationComponent struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ApplicationID uint       `gorm:"not null;index" json:"application_id"`
	Name          string     `gorm:"size:128;not null" json:"name"`
	Kind          string     `gorm:"size:32;not null" json:"kind"`
	ProjectID     *uint      `gorm:"index" json:"project_id,omitempty"`
	DependencyID  *uint      `gorm:"index" json:"dependency_id,omitempty"`
	DependsOn     StringList `gorm:"type:jsonb" json:"depends_on,omitempty"`
	Status        string     `gorm:"size:32;not null;default:'creating'" json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type ManagedDependency struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	ApplicationID     uint      `gorm:"not null;index" json:"application_id"`
	Name              string    `gorm:"size:128;not null;index" json:"name"`
	Type              string    `gorm:"size:64;not null;index" json:"type"`
	Version           string    `gorm:"size:64" json:"version"`
	DeployMethod      string    `gorm:"size:32;not null" json:"deploy_method"`
	Namespace         string    `gorm:"size:128;not null" json:"namespace"`
	ServiceName       string    `gorm:"size:253;not null" json:"service_name"`
	SecretName        string    `gorm:"size:253;not null" json:"secret_name"`
	DefinitionName    string    `gorm:"size:128;not null" json:"definition_name"`
	DefinitionVersion string    `gorm:"size:64;not null;default:'v1'" json:"definition_version"`
	Config            JSONMap   `gorm:"type:jsonb" json:"config,omitempty"`
	Outputs           JSONMap   `gorm:"type:jsonb" json:"outputs,omitempty"`
	Status            string    `gorm:"size:32;not null;default:'creating'" json:"status"`
	Shared            bool      `gorm:"not null;default:false" json:"shared"`
	External          bool      `gorm:"not null;default:false" json:"external"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type DependencyCredential struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	DependencyID uint      `gorm:"not null;index" json:"dependency_id"`
	Name         string    `gorm:"size:128;not null;index" json:"name"`
	Description  string    `gorm:"size:512" json:"description,omitempty"`
	Config       JSONMap   `gorm:"type:jsonb" json:"config,omitempty"`
	Outputs      JSONMap   `gorm:"type:jsonb" json:"outputs,omitempty"`
	Status       string    `gorm:"size:32;not null;default:'ready'" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type DependencyDefinitionRecord struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;not null;index" json:"name"`
	Version     string    `gorm:"size:64;not null;default:'v1'" json:"version"`
	ContentYAML string    `gorm:"type:text;not null" json:"content_yaml"`
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
