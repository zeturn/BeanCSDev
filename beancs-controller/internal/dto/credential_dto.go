package dto

type CreateCloudflareCredentialRequest struct {
	Name      string `json:"name" validate:"required,max=128"`
	APIToken  string `json:"api_token" validate:"required"`
	ZoneID    string `json:"zone_id" validate:"required,max=128"`
	Domain    string `json:"domain" validate:"required,fqdn"`
	AccountID string `json:"account_id" validate:"omitempty,max=128"`
}

type UpdateCloudflareCredentialRequest struct {
	Name      *string `json:"name" validate:"omitempty,max=128"`
	APIToken  *string `json:"api_token"`
	ZoneID    *string `json:"zone_id" validate:"omitempty,max=128"`
	Domain    *string `json:"domain" validate:"omitempty,fqdn"`
	AccountID *string `json:"account_id" validate:"omitempty,max=128"`
	IsActive  *bool   `json:"is_active"`
}

type CreateGitHubCredentialRequest struct {
	Name       string `json:"name" validate:"required,max=128"`
	Token      string `json:"token" validate:"required"`
	Org        string `json:"org" validate:"omitempty,max=128"`
	GitOpsRepo string `json:"gitops_repo" validate:"required,max=256"`
}

type UpdateGitHubCredentialRequest struct {
	Name       *string `json:"name" validate:"omitempty,max=128"`
	Token      *string `json:"token"`
	Org        *string `json:"org" validate:"omitempty,max=128"`
	GitOpsRepo *string `json:"gitops_repo" validate:"omitempty,max=256"`
	IsActive   *bool   `json:"is_active"`
}

type CreateBasaltPassCredentialRequest struct {
	Name         string `json:"name" validate:"required,max=128"`
	BaseURL      string `json:"base_url" validate:"required,url"`
	ClientID     string `json:"client_id" validate:"required,max=256"`
	ClientSecret string `json:"client_secret" validate:"required"`
	ServiceToken string `json:"service_token" validate:"omitempty"`
}

type UpdateBasaltPassCredentialRequest struct {
	Name         *string `json:"name" validate:"omitempty,max=128"`
	BaseURL      *string `json:"base_url" validate:"omitempty,url"`
	ClientID     *string `json:"client_id" validate:"omitempty,max=256"`
	ClientSecret *string `json:"client_secret"`
	ServiceToken *string `json:"service_token"`
	IsActive     *bool   `json:"is_active"`
}

type ShareCredentialRequest struct {
	UserID string `json:"user_id" validate:"required,max=128"`
	Role   string `json:"role" validate:"omitempty,oneof=owner user"`
}
