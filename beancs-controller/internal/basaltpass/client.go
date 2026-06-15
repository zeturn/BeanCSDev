package basaltpass

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Client interface {
	IntrospectToken(ctx context.Context, token string) (*IntrospectionResult, error)
	CreateTenant(ctx context.Context, req *CreateTenantRequest) (*CreateTenantResponse, error)
	RegisterApp(ctx context.Context, req *RegisterAppRequest) (*RegisterAppResponse, error)
	DeleteApp(ctx context.Context, appID uint) error
	HealthCheck(ctx context.Context) (*HealthStatus, error)
}

type Actor struct {
	Sub      string `json:"sub"`
	ClientID string `json:"client_id"`
}

type IntrospectionResult struct {
	Active     bool   `json:"active"`
	Sub        string `json:"sub"`
	ClientID   string `json:"client_id"`
	Scope      string `json:"scope"`
	TenantID   string `json:"tenant_id"`
	TenantCode string `json:"tenant_code"`
	Exp        int64  `json:"exp"`
	Act        *Actor `json:"act,omitempty"`
}

func (r *IntrospectionResult) UnmarshalJSON(data []byte) error {
	var raw struct {
		Active     bool            `json:"active"`
		Sub        any             `json:"sub"`
		ClientID   any             `json:"client_id"`
		Scope      any             `json:"scope"`
		TenantID   any             `json:"tenant_id"`
		TenantCode any             `json:"tenant_code"`
		Tenant     any             `json:"tenant"`
		Exp        any             `json:"exp"`
		Act        json.RawMessage `json:"act"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Active = raw.Active
	r.Sub = stringifyJSONValue(raw.Sub)
	r.ClientID = stringifyJSONValue(raw.ClientID)
	r.Scope = scopeString(raw.Scope)
	r.TenantID = stringifyJSONValue(raw.TenantID)
	r.TenantCode = coalesceString(stringifyJSONValue(raw.TenantCode), tenantCodeFromValue(raw.Tenant))
	r.Exp = int64JSONValue(raw.Exp)
	r.Act = nil
	if len(raw.Act) > 0 && string(raw.Act) != "null" {
		var actRaw struct {
			Sub      any `json:"sub"`
			ClientID any `json:"client_id"`
		}
		if err := json.Unmarshal(raw.Act, &actRaw); err != nil {
			return err
		}
		r.Act = &Actor{Sub: stringifyJSONValue(actRaw.Sub), ClientID: stringifyJSONValue(actRaw.ClientID)}
	}
	return nil
}

func coalesceString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func tenantCodeFromValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case map[string]any:
		return coalesceString(
			stringifyJSONValue(t["code"]),
			stringifyJSONValue(t["tenant_code"]),
			stringifyJSONValue(t["tenantCode"]),
			stringifyJSONValue(t["slug"]),
			stringifyJSONValue(t["name"]),
		)
	default:
		return stringifyJSONValue(t)
	}
}

func stringifyJSONValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprint(t)
	}
}

func scopeString(v any) string {
	switch t := v.(type) {
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			if s := stringifyJSONValue(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	default:
		return stringifyJSONValue(t)
	}
}

func int64JSONValue(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		return 0
	}
}

type RegisterAppRequest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	IconURL           string   `json:"icon_url,omitempty"`
	LogoURL           string   `json:"logo_url,omitempty"`
	HomepageURL       string   `json:"homepage_url,omitempty"`
	PrivacyPolicyURL  string   `json:"privacy_policy_url,omitempty"`
	TermsOfServiceURL string   `json:"terms_of_service_url,omitempty"`
	IsVerified        bool     `json:"is_verified,omitempty"`
	RedirectURIs      []string `json:"redirect_uris"`
	Scopes            []string `json:"scopes,omitempty"`
	AllowedOrigins    []string `json:"allowed_origins,omitempty"`
}

type CreateTenantRequest struct {
	Name             string `json:"name"`
	Code             string `json:"code"`
	Description      string `json:"description,omitempty"`
	OwnerEmail       string `json:"owner_email"`
	MaxApps          int    `json:"max_apps"`
	MaxUsers         int    `json:"max_users"`
	MaxTokensPerHour int    `json:"max_tokens_per_hour"`
}

type CreateTenantResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Code  string `json:"code"`
	Token string `json:"token,omitempty"`
}

type RegisterAppResponse struct {
	Data struct {
		ID           uint   `json:"id"`
		TenantID     uint   `json:"tenant_id"`
		Name         string `json:"name"`
		Status       string `json:"status"`
		OAuthClients []struct {
			ID           uint     `json:"id"`
			ClientID     string   `json:"client_id"`
			ClientSecret string   `json:"client_secret"`
			RedirectURIs []string `json:"redirect_uris"`
			Scopes       []string `json:"scopes"`
			IsActive     bool     `json:"is_active"`
		} `json:"oauth_clients"`
	} `json:"data"`
	Message string `json:"message"`
}

type HealthStatus struct {
	Status    string    `json:"status"`
	CheckedAt time.Time `json:"checked_at"`
}
