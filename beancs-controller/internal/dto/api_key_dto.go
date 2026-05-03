package dto

import "time"

type CreateAPIKeyRequest struct {
	Name      string   `json:"name" validate:"required,max=128"`
	Scopes    []string `json:"scopes" validate:"omitempty,dive,max=64"`
	ExpiresAt string   `json:"expires_at" validate:"omitempty"`
}

type APIKeyResponse struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CreateAPIKeyResponse struct {
	APIKeyResponse
	Key string `json:"key"`
}
