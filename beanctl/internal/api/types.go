package api

import "time"

type ListResponse[T any] struct {
	Data []T `json:"data"`
}

type Credential struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	Domain     string    `json:"domain"`
	ZoneID     string    `json:"zone_id"`
	Org        string    `json:"org"`
	GitOpsRepo string    `json:"gitops_repo"`
	BaseURL    string    `json:"base_url"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

type Project struct {
	ID             uint          `json:"id"`
	Name           string        `json:"name"`
	Status         string        `json:"status"`
	Domain         string        `json:"domain"`
	ExposureMode   string        `json:"exposure_mode"`
	ResourcePreset string        `json:"resource_preset"`
	Replicas       int           `json:"replicas"`
	Port           int           `json:"port"`
	Ports          []ProjectPort `json:"ports"`
	GitHubRepo     string        `json:"github_repo"`
	CreatedAt      time.Time     `json:"created_at"`
}

type ProjectPort struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol,omitempty"`
	Exposure string `json:"exposure"`
	Domain   string `json:"domain,omitempty"`
}

type Deployment struct {
	ID          uint      `json:"id"`
	Tag         string    `json:"tag"`
	Status      string    `json:"status"`
	ImageDigest string    `json:"image_digest"`
	CommitSHA   string    `json:"commit_sha"`
	TriggeredBy string    `json:"triggered_by"`
	CreatedAt   time.Time `json:"created_at"`
}
