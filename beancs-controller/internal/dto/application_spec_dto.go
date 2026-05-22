package dto

import appspec "github.com/zeturn/beancs-controller/internal/application/spec"

type ApplicationSpecRepoRequest struct {
	GitHubCredentialID     uint   `json:"github_credential_id" validate:"required"`
	GitHubRepo             string `json:"github_repo" validate:"required,max=256"`
	GitHubBranch           string `json:"github_branch" validate:"omitempty,max=128"`
	ConfigPath             string `json:"config_path" validate:"omitempty,max=512"`
	DryRun                 bool   `json:"dry_run"`
	BasaltPassInstanceID   *uint  `json:"basaltpass_instance_id"`
	CloudflareCredentialID *uint  `json:"cloudflare_credential_id"`
	CloudflareZoneID       string `json:"cloudflare_zone_id" validate:"omitempty,max=128"`
}

type ApplicationSpecResponse struct {
	ConfigPath string                           `json:"config_path,omitempty"`
	Found      bool                             `json:"found"`
	Document   *appspec.ApplicationSpecDocument `json:"document,omitempty"`
	Validation appspec.ValidationResult         `json:"validation"`
	Plan       *appspec.ApplicationPlan         `json:"plan,omitempty"`
	Warnings   []string                         `json:"warnings,omitempty"`
}
