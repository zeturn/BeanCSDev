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
		httpClient:         &http.Client{Timeout: 15 * time.Second},
	}
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

func (c *HTTPClient) RegisterApp(ctx context.Context, reqBody *RegisterAppRequest) (*RegisterAppResponse, error) {
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

func (c *HTTPClient) DeleteApp(ctx context.Context, appID uint) error {
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
