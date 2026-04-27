package dto

type CreateDeploymentRequest struct {
	Tag       string `json:"tag" validate:"omitempty,max=256"`
	CommitSHA string `json:"commit_sha" validate:"omitempty,max=64"`
}

type GitHubWebhookRequest struct {
	Project string `json:"project" validate:"required,max=63"`
	Tag     string `json:"tag" validate:"omitempty,max=256"`
	Digest  string `json:"digest" validate:"omitempty,max=256"`
	Commit  string `json:"commit" validate:"omitempty,max=64"`
	Status  string `json:"status" validate:"required,oneof=success failure cancelled"`
}

type ArgoCDWebhookRequest struct {
	Project      string `json:"project" validate:"required,max=63"`
	SyncStatus   string `json:"sync_status" validate:"required"`
	HealthStatus string `json:"health_status" validate:"required"`
}
