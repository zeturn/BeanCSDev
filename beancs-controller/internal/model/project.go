package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const (
	ExposurePublic       = "public"
	ExposurePrivate      = "private"
	ExposureInternalOnly = "internal-only"

	BuildSourceGitHub       = "github"
	BuildSourceDockerHub    = "dockerhub"
	BuildSourceGHCR         = "ghcr"
	BuildSourceRegistry     = "registry"
	BuildSourceSourceUpload = "source-upload"
)

type Project struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"uniqueIndex;size:63;not null" json:"name"`
	DisplayName string `gorm:"size:256" json:"display_name"`
	Description string `gorm:"size:2000" json:"description"`
	OwnerID     string `gorm:"size:128;not null;index" json:"owner_id"`
	TeamID      string `gorm:"size:128;index" json:"team_id"`
	TenantID    string `gorm:"size:128;index" json:"tenant_id"`
	TenantCode  string `gorm:"size:128;index" json:"tenant_code,omitempty"`

	ApplicationID       *uint      `gorm:"index" json:"application_id,omitempty"`
	ComponentName       string     `gorm:"size:128" json:"component_name,omitempty"`
	ComponentPath       string     `gorm:"size:512" json:"component_path,omitempty"`
	DependsOn           StringList `gorm:"type:jsonb" json:"depends_on,omitempty"`
	EnvFromDependencies JSONMap    `gorm:"type:jsonb" json:"env_from_dependencies,omitempty"`

	BuildSource       string `gorm:"size:32;default:'github'" json:"build_source"`
	ImageReference    string `gorm:"size:512" json:"image_reference"`
	SourceArchiveName string `gorm:"size:256" json:"source_archive_name"`

	GitHubCredentialID   uint       `gorm:"index" json:"github_credential_id"`
	GitHubRepo           string     `gorm:"size:256" json:"github_repo"`
	GitHubBranch         string     `gorm:"size:128;default:'main'" json:"github_branch"`
	GitHubInstallationID int64      `gorm:"index" json:"github_installation_id,omitempty"`
	GitHubRepoID         int64      `gorm:"index" json:"github_repo_id,omitempty"`
	GitHubRepoFullName   string     `gorm:"size:256" json:"github_repo_full_name,omitempty"`
	DockerfilePath       string     `gorm:"size:512;default:'Dockerfile'" json:"dockerfile_path"`
	BuildContext         string     `gorm:"size:512;default:'.'" json:"build_context"`
	BuildArgs            JSONMap    `gorm:"type:jsonb" json:"build_args,omitempty"`
	HealthCheck          JSONMap    `gorm:"type:jsonb" json:"health_check,omitempty"`
	Volumes              JSONMap    `gorm:"type:jsonb" json:"volumes,omitempty"`
	WatchPaths           StringList `gorm:"type:jsonb" json:"watch_paths,omitempty"`
	AutoDeploy           bool       `gorm:"default:true" json:"auto_deploy"`

	RegistryHost           string `gorm:"size:256" json:"registry_host,omitempty"`
	RegistryProject        string `gorm:"size:128" json:"registry_project,omitempty"`
	RegistryRepository     string `gorm:"size:256" json:"registry_repository,omitempty"`
	RegistryImageReference string `gorm:"size:512" json:"registry_image_reference,omitempty"`
	RegistryPullSecretName string `gorm:"size:253" json:"registry_pull_secret_name,omitempty"`

	BasaltPassInstanceID *uint  `gorm:"index" json:"basaltpass_instance_id,omitempty"`
	BasaltAppID          uint   `json:"basalt_app_id"`
	BasaltClientID       string `gorm:"size:256" json:"basalt_client_id"`
	BasaltSecretEnc      []byte `gorm:"type:bytea" json:"-"`

	CloudflareCredentialID *uint `gorm:"index" json:"cloudflare_credential_id,omitempty"`

	ExposureMode   string       `gorm:"size:32;default:'private'" json:"exposure_mode"`
	Subdomain      string       `gorm:"size:63" json:"subdomain"`
	Domain         string       `gorm:"size:256" json:"domain"`
	Namespace      string       `gorm:"size:128;not null" json:"namespace"`
	ResourcePreset string       `gorm:"size:32;default:'small'" json:"resource_preset"`
	Port           int          `gorm:"default:8080" json:"port"`
	Ports          ProjectPorts `gorm:"type:jsonb" json:"ports"`
	Replicas       int          `gorm:"default:1" json:"replicas"`
	Status         string       `gorm:"size:32;default:'active'" json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProjectPort struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol,omitempty"`
	Exposure string `json:"exposure"`
	Domain   string `json:"domain,omitempty"`
}

type ProjectPorts []ProjectPort

func (p ProjectPorts) Value() (driver.Value, error) {
	if p == nil {
		return "[]", nil
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (p *ProjectPorts) Scan(value any) error {
	if value == nil {
		*p = ProjectPorts{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported project ports value %T", value)
	}
	return json.Unmarshal(data, p)
}

type ResourceSpec struct {
	CPURequest string
	CPULimit   string
	MemRequest string
	MemLimit   string
	CPUMillis  int
	MemoryMB   int
}

var ResourcePresets = map[string]ResourceSpec{
	"nano":   {CPURequest: "50m", CPULimit: "100m", MemRequest: "64Mi", MemLimit: "128Mi", CPUMillis: 100, MemoryMB: 128},
	"small":  {CPURequest: "100m", CPULimit: "250m", MemRequest: "128Mi", MemLimit: "256Mi", CPUMillis: 250, MemoryMB: 256},
	"medium": {CPURequest: "250m", CPULimit: "500m", MemRequest: "256Mi", MemLimit: "512Mi", CPUMillis: 500, MemoryMB: 512},
	"large":  {CPURequest: "500m", CPULimit: "1000m", MemRequest: "512Mi", MemLimit: "1Gi", CPUMillis: 1000, MemoryMB: 1024},
}
