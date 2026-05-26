package dto

import "github.com/zeturn/beancs-controller/internal/model"

type CreateManagedDependencyRequest struct {
	Name         string        `json:"name" validate:"required,hostname_rfc1123,max=63"`
	Type         string        `json:"type" validate:"required,max=64"`
	Version      string        `json:"version" validate:"omitempty,max=64"`
	DeployMethod string        `json:"deploy_method" validate:"omitempty,oneof=helm manifest operator external"`
	Config       model.JSONMap `json:"config" validate:"omitempty"`
}

type LinkProjectDependencyRequest struct {
	Dependency string         `json:"dependency" validate:"required,max=128"`
	Preset     string         `json:"preset" validate:"omitempty,max=128"`
	Mappings   map[string]any `json:"mappings" validate:"omitempty"`
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
	Outputs model.JSONMap `json:"outputs,omitempty"`
}
