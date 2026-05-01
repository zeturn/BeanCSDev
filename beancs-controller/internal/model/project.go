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
)

type Project struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"uniqueIndex;size:63;not null" json:"name"`
	DisplayName string `gorm:"size:256" json:"display_name"`
	Description string `gorm:"size:2000" json:"description"`
	OwnerID     string `gorm:"size:128;not null;index" json:"owner_id"`
	TeamID      string `gorm:"size:128;index" json:"team_id"`
	TenantID    string `gorm:"size:128;index" json:"tenant_id"`

	GitHubCredentialID uint   `gorm:"not null;index" json:"github_credential_id"`
	GitHubRepo         string `gorm:"size:256;not null" json:"github_repo"`
	GitHubBranch       string `gorm:"size:128;default:'main'" json:"github_branch"`
	DockerfilePath     string `gorm:"size:512;default:'Dockerfile'" json:"dockerfile_path"`

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
