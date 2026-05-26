package dto

import "github.com/zeturn/beancs-controller/internal/model"

type CreateManagedDependencyRequest struct {
	ApplicationName      string                             `json:"application_name" validate:"omitempty,hostname_rfc1123,max=63"`
	DisplayName          string                             `json:"display_name" validate:"omitempty,max=256"`
	Namespace            string                             `json:"namespace" validate:"omitempty,hostname_rfc1123,max=63"`
	GitHubCredentialID   uint                               `json:"github_credential_id" validate:"omitempty"`
	Name                 string                             `json:"name" validate:"required,hostname_rfc1123,max=63"`
	Type                 string                             `json:"type" validate:"omitempty,max=64"`
	Version              string                             `json:"version" validate:"omitempty,max=64"`
	DeployMethod         string                             `json:"deploy_method" validate:"omitempty,oneof=helm manifest operator external"`
	Config               model.JSONMap                      `json:"config" validate:"omitempty"`
	Shared               bool                               `json:"shared"`
	External             bool                               `json:"external"`
	ExistingDependencyID uint                               `json:"existing_dependency_id" validate:"omitempty"`
	Credential           *CreateDependencyCredentialRequest `json:"credential" validate:"omitempty"`
}

type LinkProjectDependencyRequest struct {
	Dependency   string         `json:"dependency" validate:"required_without=DependencyID,max=128"`
	DependencyID uint           `json:"dependency_id" validate:"omitempty"`
	Credential   string         `json:"credential" validate:"omitempty,max=128"`
	CredentialID uint           `json:"credential_id" validate:"omitempty"`
	Preset       string         `json:"preset" validate:"omitempty,max=128"`
	Mappings     map[string]any `json:"mappings" validate:"omitempty"`
}

type CreateDependencyCredentialRequest struct {
	Name        string        `json:"name" validate:"required,max=128"`
	Description string        `json:"description" validate:"omitempty,max=512"`
	Config      model.JSONMap `json:"config" validate:"omitempty"`
}

type DependencyDefinitionSummary struct {
	Name                   string   `json:"name"`
	DisplayName            string   `json:"display_name"`
	Category               string   `json:"category"`
	Type                   string   `json:"type"`
	SupportedDeployMethods []string `json:"supported_deploy_methods"`
	DefaultDeployMethod    string   `json:"default_deploy_method"`
}

type ManagedDependencyResponse struct {
	model.ManagedDependency
	Outputs     model.JSONMap                `json:"outputs,omitempty"`
	Credentials []model.DependencyCredential `json:"credentials,omitempty"`
}
