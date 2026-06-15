package dto

type CreateCloudflareCredentialRequest struct {
	Name      string `json:"name" validate:"omitempty,max=128"`
	APIToken  string `json:"api_token" validate:"required"`
	ZoneID    string `json:"zone_id" validate:"omitempty,max=128"`
	Domain    string `json:"domain" validate:"omitempty,fqdn"`
	AccountID string `json:"account_id" validate:"omitempty,max=128"`
}

type CloudflareDomainResponse struct {
	CredentialID uint   `json:"credential_id"`
	Credential   string `json:"credential"`
	ZoneID       string `json:"zone_id"`
	Domain       string `json:"domain"`
	AccountID    string `json:"account_id,omitempty"`
	Status       string `json:"status,omitempty"`
	Active       bool   `json:"active"`
}

type CloudflareDNSRecordResponse struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Proxied  bool   `json:"proxied"`
	Comment  string `json:"comment,omitempty"`
	Modified string `json:"modified_on,omitempty"`
}

type CreateCloudflareDNSRecordRequest struct {
	Type    string `json:"type" validate:"required,oneof=A AAAA CNAME TXT MX"`
	Name    string `json:"name" validate:"required,max=256"`
	Content string `json:"content" validate:"required,max=2048"`
	TTL     int    `json:"ttl" validate:"omitempty,min=1"`
	Proxied bool   `json:"proxied"`
	Comment string `json:"comment" validate:"omitempty,max=500"`
}

type UpdateCloudflareDNSRecordRequest struct {
	Type    string `json:"type" validate:"required,oneof=A AAAA CNAME TXT MX"`
	Name    string `json:"name" validate:"required,max=256"`
	Content string `json:"content" validate:"required,max=2048"`
	TTL     int    `json:"ttl" validate:"omitempty,min=1"`
	Proxied bool   `json:"proxied"`
	Comment string `json:"comment" validate:"omitempty,max=500"`
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
	Name       string `json:"name" validate:"omitempty,max=128"`
	Token      string `json:"token" validate:"required"`
	Org        string `json:"org" validate:"omitempty,max=128"`
	GitOpsRepo string `json:"gitops_repo" validate:"omitempty,max=256"`
}

type StartGitHubAppInstallRequest struct {
	GitOpsRepo string `json:"gitops_repo" validate:"omitempty,max=256"`
}

type UpdateGitHubCredentialRequest struct {
	Name       *string `json:"name" validate:"omitempty,max=128"`
	Token      *string `json:"token"`
	Org        *string `json:"org" validate:"omitempty,max=128"`
	GitOpsRepo *string `json:"gitops_repo" validate:"omitempty,max=256"`
	IsActive   *bool   `json:"is_active"`
}

type CreateBasaltPassCredentialRequest struct {
	Name            string `json:"name" validate:"required,max=128"`
	BaseURL         string `json:"base_url" validate:"required,url"`
	TenantID        string `json:"tenant_id" validate:"omitempty,max=128"`
	TenantCode      string `json:"tenant_code" validate:"omitempty,max=128"`
	AutomationToken string `json:"automation_token" validate:"omitempty"`
	ClientID        string `json:"client_id" validate:"omitempty,max=256"`
	ClientSecret    string `json:"client_secret" validate:"omitempty"`
	ServiceToken    string `json:"service_token" validate:"omitempty"`
}

type DeployBasaltPassRequest struct {
	Name                   string `json:"name" validate:"required,max=128"`
	BaseURL                string `json:"base_url" validate:"required,url"`
	TenantName             string `json:"tenant_name" validate:"omitempty,max=128"`
	TenantCode             string `json:"tenant_code" validate:"omitempty,max=128"`
	Namespace              string `json:"namespace" validate:"omitempty,max=128"`
	BackendImage           string `json:"backend_image" validate:"required,max=512"`
	FrontendImage          string `json:"frontend_image" validate:"required,max=512"`
	GitHubCredentialID     uint   `json:"github_credential_id" validate:"omitempty"`
	GitHubRepo             string `json:"github_repo" validate:"omitempty,max=256"`
	GitHubBranch           string `json:"github_branch" validate:"omitempty,max=128"`
	PublicHost             string `json:"public_host" validate:"omitempty,max=256"`
	ExposureMode           string `json:"exposure_mode" validate:"omitempty,oneof=public private"`
	JWTSecret              string `json:"jwt_secret" validate:"omitempty"`
	CORSAllowOrigins       string `json:"cors_allow_origins" validate:"omitempty,max=2048"`
	PlatformAdminEmail     string `json:"platform_admin_email" validate:"required,email"`
	PlatformAdminUsername  string `json:"platform_admin_username" validate:"required,max=64"`
	PlatformAdminPassword  string `json:"platform_admin_password" validate:"required"`
	CloudflareCredentialID uint   `json:"cloudflare_credential_id" validate:"omitempty"`
	CloudflareZoneID       string `json:"cloudflare_zone_id" validate:"omitempty,max=128"`
	DatabaseDependencyID   uint   `json:"database_dependency_id" validate:"required"`
	DatabaseCredentialID   uint   `json:"database_credential_id" validate:"required"`
	OwnerEmail             string `json:"owner_email" validate:"required,email"`
	OwnerUsername          string `json:"owner_username" validate:"required,max=64"`
	OwnerPassword          string `json:"owner_password" validate:"required"`
	Description            string `json:"description" validate:"omitempty,max=500"`
	MaxApps                int    `json:"max_apps" validate:"omitempty,min=1"`
	MaxUsers               int    `json:"max_users" validate:"omitempty,min=1"`
	MaxTokensPerHour       int    `json:"max_tokens_per_hour" validate:"omitempty,min=1"`
	AutomationToken        string `json:"automation_token" validate:"omitempty"`
	ClientID               string `json:"client_id" validate:"omitempty,max=256"`
	ClientSecret           string `json:"client_secret" validate:"omitempty"`
	ServiceToken           string `json:"service_token" validate:"omitempty"`
}

type UpdateBasaltPassCredentialRequest struct {
	Name            *string `json:"name" validate:"omitempty,max=128"`
	BaseURL         *string `json:"base_url" validate:"omitempty,url"`
	TenantID        *string `json:"tenant_id" validate:"omitempty,max=128"`
	TenantCode      *string `json:"tenant_code" validate:"omitempty,max=128"`
	AutomationToken *string `json:"automation_token"`
	ClientID        *string `json:"client_id" validate:"omitempty,max=256"`
	ClientSecret    *string `json:"client_secret"`
	ServiceToken    *string `json:"service_token"`
	IsActive        *bool   `json:"is_active"`
}

type ShareCredentialRequest struct {
	UserID string `json:"user_id" validate:"required,max=128"`
	Role   string `json:"role" validate:"omitempty,oneof=owner user"`
}

type GitHubRepositoryResponse struct {
	FullName      string `json:"full_name"`
	Name          string `json:"name"`
	Owner         string `json:"owner"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
}
