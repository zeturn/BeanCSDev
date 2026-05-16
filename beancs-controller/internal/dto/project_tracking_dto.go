package dto

import "time"

type ProjectTrackingResponse struct {
	ProjectID         uint                    `json:"project_id"`
	ProjectName       string                  `json:"project_name"`
	DisplayName       string                  `json:"display_name,omitempty"`
	ProjectStatus     string                  `json:"project_status"`
	BuildSource       string                  `json:"build_source"`
	GitHubRepo        string                  `json:"github_repo,omitempty"`
	GitHubBranch      string                  `json:"github_branch,omitempty"`
	Namespace         string                  `json:"namespace"`
	Domain            string                  `json:"domain,omitempty"`
	CurrentImage      string                  `json:"current_image,omitempty"`
	CurrentVersion    string                  `json:"current_version,omitempty"`
	LatestStatus      string                  `json:"latest_status,omitempty"`
	LatestDeployment  *DeploymentHistoryItem  `json:"latest_deployment,omitempty"`
	RunningDeployment *DeploymentHistoryItem  `json:"running_deployment,omitempty"`
	History           []DeploymentHistoryItem `json:"history"`
	Summary           ProjectTrackingSummary  `json:"summary"`
}

type ProjectTrackingSummary struct {
	Total      int `json:"total"`
	Running    int `json:"running"`
	Deploying  int `json:"deploying"`
	Building   int `json:"building"`
	Queued     int `json:"queued"`
	Failed     int `json:"failed"`
	Successful int `json:"successful"`
}

type DeploymentHistoryItem struct {
	ID            uint       `json:"id"`
	Version       string     `json:"version,omitempty"`
	Tag           string     `json:"tag,omitempty"`
	ImageRef      string     `json:"image_ref,omitempty"`
	ImageDigest   string     `json:"image_digest,omitempty"`
	CommitSHA     string     `json:"commit_sha,omitempty"`
	Status        string     `json:"status"`
	TriggeredBy   string     `json:"triggered_by,omitempty"`
	FailureReason string     `json:"failure_reason,omitempty"`
	WorkflowRunID int64      `json:"workflow_run_id,omitempty"`
	WorkflowURL   string     `json:"workflow_url,omitempty"`
	ProcessID     uint       `json:"process_id,omitempty"`
	ProcessStatus string     `json:"process_status,omitempty"`
	ProcessTitle  string     `json:"process_title,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
}
