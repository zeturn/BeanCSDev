package dto

import "github.com/zeturn/beancs-controller/internal/model"

type CreateMonorepoApplicationRequest struct {
	Name                   string                           `json:"name" validate:"required,hostname_rfc1123,max=63"`
	DisplayName            string                           `json:"display_name" validate:"omitempty,max=256"`
	Description            string                           `json:"description" validate:"omitempty,max=2000"`
	TeamID                 string                           `json:"team_id" validate:"omitempty,max=128"`
	GitHubCredentialID     uint                             `json:"github_credential_id" validate:"required"`
	GitHubRepo             string                           `json:"github_repo" validate:"required,max=256"`
	GitHubBranch           string                           `json:"github_branch" validate:"omitempty,max=128"`
	AutoDeploy             *bool                            `json:"auto_deploy" validate:"omitempty"`
	Namespace              string                           `json:"namespace" validate:"omitempty,hostname_rfc1123,max=63"`
	BasaltPassInstanceID   *uint                            `json:"basaltpass_instance_id"`
	CloudflareCredentialID *uint                            `json:"cloudflare_credential_id"`
	CloudflareZoneID       string                           `json:"cloudflare_zone_id" validate:"omitempty,max=128"`
	ResourcePreset         string                           `json:"resource_preset" validate:"omitempty,oneof=nano small medium large"`
	Dependencies           []CreateManagedDependencyRequest `json:"dependencies" validate:"omitempty,dive"`
	Components             []MonorepoComponentRequest       `json:"components" validate:"required,min=1,dive"`
}

type MonorepoComponentRequest struct {
	Name                string                     `json:"name" validate:"required,hostname_rfc1123,max=63"`
	Kind                string                     `json:"kind" validate:"omitempty,oneof=service worker frontend"`
	ProjectName         string                     `json:"project_name" validate:"required,hostname_rfc1123,max=63"`
	DisplayName         string                     `json:"display_name" validate:"omitempty,max=256"`
	Description         string                     `json:"description" validate:"omitempty,max=2000"`
	DockerfilePath      string                     `json:"dockerfile_path" validate:"required,max=512"`
	BuildContext        string                     `json:"build_context" validate:"omitempty,max=512"`
	BuildArgs           model.JSONMap              `json:"build_args" validate:"omitempty"`
	ComponentPath       string                     `json:"component_path" validate:"omitempty,max=512"`
	Namespace           string                     `json:"namespace" validate:"omitempty,hostname_rfc1123,max=63"`
	ExposureMode        string                     `json:"exposure_mode" validate:"omitempty,oneof=public private internal-only"`
	Subdomain           string                     `json:"subdomain" validate:"omitempty,hostname_rfc1123,max=63"`
	Ports               model.ProjectPorts         `json:"ports" validate:"omitempty"`
	Replicas            int                        `json:"replicas" validate:"omitempty,min=1,max=20"`
	ResourcePreset      string                     `json:"resource_preset" validate:"omitempty,oneof=nano small medium large"`
	Env                 map[string]string          `json:"env" validate:"omitempty"`
	DependsOn           []string                   `json:"depends_on" validate:"omitempty,dive,max=128"`
	EnvFromDependencies []EnvFromDependencyRequest `json:"env_from_dependencies" validate:"omitempty,dive"`
	HealthCheck         model.JSONMap              `json:"health_check" validate:"omitempty"`
	Volumes             model.JSONMap              `json:"volumes" validate:"omitempty"`
	WatchPaths          []string                   `json:"watch_paths" validate:"omitempty,dive,max=512"`
}

type EnvFromDependencyRequest struct {
	Dependency   string         `json:"dependency" validate:"required_without=DependencyID,max=128"`
	DependencyID uint           `json:"dependency_id" validate:"omitempty"`
	Credential   string         `json:"credential" validate:"omitempty,max=128"`
	CredentialID uint           `json:"credential_id" validate:"omitempty"`
	Preset       string         `json:"preset" validate:"omitempty,max=128"`
	Mappings     map[string]any `json:"mappings" validate:"omitempty"`
}

type ApplicationResponse struct {
	model.Application
	Projects     []model.Project              `json:"projects,omitempty"`
	Dependencies []model.ManagedDependency    `json:"dependencies,omitempty"`
	Components   []model.ApplicationComponent `json:"components,omitempty"`
}
