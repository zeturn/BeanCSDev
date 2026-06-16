package basaltpass

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HTTPClient struct {
	baseURL            string
	apiBaseURL         string
	clientID           string
	clientSecret       string
	staticServiceToken string
	adminIdentifier    string
	adminPassword      string
	httpClient         *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewHTTPClient(baseURL, clientID, clientSecret string) *HTTPClient {
	return NewHTTPClientWithServiceToken(baseURL, clientID, clientSecret, "")
}

func NewHTTPClientWithServiceToken(baseURL, clientID, clientSecret, serviceToken string) *HTTPClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &HTTPClient{
		baseURL:            baseURL,
		apiBaseURL:         basaltAPIBaseURL(baseURL),
		clientID:           clientID,
		clientSecret:       clientSecret,
		staticServiceToken: strings.TrimSpace(serviceToken),
		httpClient:         &http.Client{Timeout: 5 * time.Minute},
	}
}

func NewHTTPClientWithAdminCredentials(baseURL, identifier, password string) *HTTPClient {
	client := NewHTTPClientWithServiceToken(baseURL, "", "", "")
	client.adminIdentifier = strings.TrimSpace(identifier)
	client.adminPassword = password
	return client
}

func (c *HTTPClient) IntrospectToken(ctx context.Context, token string) (*IntrospectionResult, error) {
	form := url.Values{}
	form.Set("token", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBaseURL+"/oauth/introspect", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.clientID+":"+c.clientSecret)))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("introspection failed: %s", parseAPIError(body))
	}
	var result IntrospectionResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *HTTPClient) CreateTenant(ctx context.Context, reqBody *CreateTenantRequest) (*CreateTenantResponse, error) {
	token, err := c.serviceToken(ctx)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBaseURL+"/admin/tenants", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create tenant failed: %s", parseAPIError(body))
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	data, _ := raw["data"].(map[string]any)
	if data == nil {
		data = raw
	}
	out := &CreateTenantResponse{
		ID:    uint(numberValue(data["id"])),
		Name:  stringValue(data["name"]),
		Code:  firstString(data["code"], data["tenant_code"], data["tenantCode"]),
		Token: firstString(data["automation_token"], data["service_token"], data["token"]),
	}
	if out.ID == 0 || out.Code == "" {
		return nil, fmt.Errorf("create tenant returned incomplete response")
	}
	return out, nil
}

func (c *HTTPClient) RegisterApp(ctx context.Context, reqBody *RegisterAppRequest) (*RegisterAppResponse, error) {
	if strings.HasPrefix(strings.TrimSpace(c.staticServiceToken), "bpk_") {
		return c.registerAppWithManualAPIKey(ctx, reqBody)
	}
	token, err := c.serviceToken(ctx)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBaseURL+"/tenant/apps", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("register app failed: %s", parseAPIError(body))
	}
	var out RegisterAppResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.Data.ID == 0 || out.Data.Status != "active" || len(out.Data.OAuthClients) == 0 {
		return nil, fmt.Errorf("register app returned incomplete response")
	}
	return &out, nil
}

func (c *HTTPClient) registerAppWithManualAPIKey(ctx context.Context, reqBody *RegisterAppRequest) (*RegisterAppResponse, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBaseURL+"/manual/apps", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", strings.TrimSpace(c.staticServiceToken))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("register app failed: %s", parseAPIError(body))
	}
	var raw struct {
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
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	out := &RegisterAppResponse{}
	out.Data.ID = raw.Data.ID
	out.Data.TenantID = raw.Data.TenantID
	out.Data.Name = raw.Data.Name
	out.Data.Status = raw.Data.Status
	out.Data.OAuthClients = raw.Data.OAuthClients
	if out.Data.ID == 0 || out.Data.Status != "active" || len(out.Data.OAuthClients) == 0 {
		return nil, fmt.Errorf("register app returned incomplete response")
	}
	return out, nil
}

func firstString(values ...any) string {
	for _, value := range values {
		if s := stringValue(value); s != "" {
			return s
		}
	}
	return ""
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func numberValue(value any) uint64 {
	switch v := value.(type) {
	case float64:
		return uint64(v)
	case int:
		return uint64(v)
	case uint:
		return uint64(v)
	case string:
		n, _ := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		return n
	default:
		return 0
	}
}

func (c *HTTPClient) DeleteApp(ctx context.Context, appID uint) error {
	if strings.HasPrefix(strings.TrimSpace(c.staticServiceToken), "bpk_") {
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%s/manual/apps/%d", c.apiBaseURL, appID), nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-API-Key", strings.TrimSpace(c.staticServiceToken))
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return nil
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			return fmt.Errorf("delete app failed: %s", parseAPIError(body))
		}
		return nil
	}
	token, err := c.serviceToken(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%s/tenant/apps/%d", c.apiBaseURL, appID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("delete app failed: %s", parseAPIError(body))
	}
	return nil
}

func (c *HTTPClient) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBaseURL+"/health", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("health check returned %d", resp.StatusCode)
	}
	return &HealthStatus{Status: "healthy", CheckedAt: time.Now().UTC()}, nil
}

func (c *HTTPClient) serviceToken(ctx context.Context) (string, error) {
	if c.staticServiceToken != "" {
		return c.staticServiceToken, nil
	}
	c.mu.Lock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-30*time.Second)) {
		token := c.accessToken
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	if c.adminIdentifier != "" && c.adminPassword != "" {
		return c.loginAdminToken(ctx)
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", "openid profile tenant.admin")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBaseURL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.clientID+":"+c.clientSecret)))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("service token failed: %s", parseAPIError(body))
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", err
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("service token response missing access_token")
	}
	if tr.ExpiresIn <= 0 {
		tr.ExpiresIn = 300
	}
	c.mu.Lock()
	c.accessToken = tr.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	c.mu.Unlock()
	return tr.AccessToken, nil
}

func (c *HTTPClient) loginAdminToken(ctx context.Context) (string, error) {
	payload, err := json.Marshal(map[string]string{
		"identifier": c.adminIdentifier,
		"password":   c.adminPassword,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBaseURL+"/auth/login", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Scope", "admin")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("admin login failed: %s", parseAPIError(body))
	}
	var out struct {
		AccessToken string `json:"access_token"`
		Data        struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	token := firstString(out.AccessToken, out.Data.Token)
	if token == "" {
		return "", fmt.Errorf("admin login returned no access token")
	}
	c.mu.Lock()
	c.accessToken = token
	c.tokenExpiry = time.Now().Add(10 * time.Minute)
	c.mu.Unlock()
	return token, nil
}

func basaltAPIBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + "/api/v1"
	}
	if strings.HasSuffix(strings.TrimRight(parsed.Path, "/"), "/api/v1") {
		return baseURL
	}
	return baseURL + "/api/v1"
}

func parseAPIError(body []byte) string {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err == nil {
		if code, ok := obj["code"].(string); ok && code != "" {
			if msg, ok := obj["error"].(string); ok && msg != "" {
				return code + ": " + msg
			}
			return code
		}
		if msg, ok := obj["error"].(string); ok && msg != "" {
			return msg
		}
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		return "upstream returned an error"
	}
	return msg
}
