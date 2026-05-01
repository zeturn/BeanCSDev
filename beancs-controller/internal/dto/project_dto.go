package dto

import "github.com/zeturn/beancs-controller/internal/model"

type CreateProjectRequest struct {
	Name                   string             `json:"name" validate:"required,hostname_rfc1123,max=63"`
	DisplayName            string             `json:"display_name" validate:"omitempty,max=256"`
	Description            string             `json:"description" validate:"omitempty,max=2000"`
	TeamID                 string             `json:"team_id" validate:"omitempty,max=128"`
	GitHubCredentialID     uint               `json:"github_credential_id" validate:"required"`
	GitHubRepo             string             `json:"github_repo" validate:"required,max=256"`
	GitHubBranch           string             `json:"github_branch" validate:"omitempty,max=128"`
	DockerfilePath         string             `json:"dockerfile_path" validate:"omitempty,max=512"`
	BasaltPassInstanceID   uint               `json:"basaltpass_instance_id" validate:"required"`
	CloudflareCredentialID *uint              `json:"cloudflare_credential_id"`
	ExposureMode           string             `json:"exposure_mode" validate:"required,oneof=public private internal-only"`
	Subdomain              string             `json:"subdomain" validate:"omitempty,hostname_rfc1123,max=63"`
	ResourcePreset         string             `json:"resource_preset" validate:"omitempty,oneof=nano small medium large"`
	Port                   int                `json:"port" validate:"omitempty,min=1,max=65535"`
	Ports                  model.ProjectPorts `json:"ports" validate:"required,min=1"`
	Replicas               int                `json:"replicas" validate:"omitempty,min=1,max=20"`
	Env                    map[string]string  `json:"env" validate:"omitempty"`
}

type AnalyzeProjectRepositoryRequest struct {
	GitHubCredentialID uint   `json:"github_credential_id" validate:"required"`
	GitHubRepo         string `json:"github_repo" validate:"required,max=256"`
	GitHubBranch       string `json:"github_branch" validate:"omitempty,max=128"`
}

type AnalyzeProjectRepositoryResponse struct {
	Deployable     bool     `json:"deployable"`
	Containerized  bool     `json:"containerized"`
	DockerfilePath string   `json:"dockerfile_path,omitempty"`
	DefaultPort    int      `json:"default_port"`
	Signals        []string `json:"signals"`
	Warnings       []string `json:"warnings"`
}

type UpdateProjectRequest struct {
	DisplayName    *string `json:"display_name" validate:"omitempty,max=256"`
	Description    *string `json:"description" validate:"omitempty,max=2000"`
	ResourcePreset *string `json:"resource_preset" validate:"omitempty,oneof=nano small medium large"`
	Port           *int    `json:"port" validate:"omitempty,min=1,max=65535"`
	Replicas       *int    `json:"replicas" validate:"omitempty,min=1,max=20"`
	Status         *string `json:"status" validate:"omitempty,oneof=active suspended deleted"`
}

type ScaleProjectRequest struct {
	Replicas int `json:"replicas" validate:"required,min=0,max=20"`
}
