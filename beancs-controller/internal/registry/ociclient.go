package registry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTagsPageSize = 500

// TagsLister lists tags via OCI Distribution Specification API v2.
type TagsLister struct {
	Client *http.Client
}

type tagsListBody struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// ListTags GET /v2/<name>/tags/list with optional Bearer token retry (Docker Hub、GHCR 等)。
func (l *TagsLister) ListTags(ctx context.Context, apiBase, repository string, user, pass *string, insecureTLS bool) ([]string, error) {
	apiBase = strings.TrimRight(strings.TrimSpace(apiBase), "/")
	repo := normalizeRepoPath(repository)
	if apiBase == "" || repo == "" {
		return nil, fmt.Errorf("api base and repository are required")
	}
	client := l.Client
	if client == nil {
		client = newHTTPClient(insecureTLS)
	}
	firstURL := tagsListURL(apiBase, repo, "")
	return l.listTagsFollow(ctx, client, firstURL, apiBase, repo, user, pass, nil, nil)
}

func (l *TagsLister) listTagsFollow(ctx context.Context, client *http.Client, requestURL, apiBase, repo string, user, pass *string, bearer *string, visited *int) ([]string, error) {
	if visited == nil {
		v := 0
		visited = &v
	}
	*visited++
	if *visited > 20 {
		return nil, fmt.Errorf("too many tag list redirects or pages")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	if bearer != nil && *bearer != "" {
		req.Header.Set("Authorization", "Bearer "+*bearer)
	} else if user != nil && pass != nil && *user != "" {
		req.SetBasicAuth(*user, *pass)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var out tagsListBody
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, fmt.Errorf("decode tags list: %w", err)
		}
		tags := dedupeTags(out.Tags)
		next := resolveNextURL(apiBase, resp.Header.Get("Link"))
		if next != "" {
			more, err := l.listTagsFollow(ctx, client, next, apiBase, repo, user, pass, bearer, visited)
			if err != nil {
				return nil, err
			}
			tags = mergeTags(tags, more)
		}
		return tags, nil
	case http.StatusUnauthorized:
		if bearer != nil && *bearer != "" {
			return nil, fmt.Errorf("registry unauthorized (401): %s", trimBody(body))
		}
		tok, err := fetchBearerToken(ctx, client, resp.Header.Get("Www-Authenticate"), apiBase, repo, user, pass)
		if err != nil {
			return nil, err
		}
		if tok == "" {
			return nil, fmt.Errorf("registry unauthorized and no token realm (401): %s", trimBody(body))
		}
		return l.listTagsFollow(ctx, client, requestURL, apiBase, repo, user, pass, &tok, visited)
	default:
		return nil, fmt.Errorf("registry error %d: %s", resp.StatusCode, trimBody(body))
	}
}

func newHTTPClient(insecureTLS bool) *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if insecureTLS {
		tr.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		}
	}
	return &http.Client{Transport: tr, Timeout: 45 * time.Second}
}

func normalizeRepoPath(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	return s
}

func tagsListURL(apiBase, repository, startAfter string) string {
	parts := strings.Split(repository, "/")
	escaped := make([]string, len(parts))
	for i, p := range parts {
		escaped[i] = url.PathEscape(p)
	}
	path := strings.Join(escaped, "/")
	u := fmt.Sprintf("%s/v2/%s/tags/list?n=%d", apiBase, path, defaultTagsPageSize)
	if startAfter != "" {
		u += "&last=" + url.QueryEscape(startAfter)
	}
	return u
}

func resolveNextURL(apiBase, linkHeader string) string {
	ref := parseNextLink(linkHeader)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	base, err := url.Parse(apiBase + "/")
	if err != nil {
		return ""
	}
	next, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return base.ResolveReference(next).String()
}

func parseNextLink(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	parts := strings.Split(linkHeader, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if !strings.Contains(p, `rel="next"`) {
			continue
		}
		idx := strings.Index(p, "<")
		end := strings.Index(p, ">")
		if idx >= 0 && end > idx {
			return p[idx+1 : end]
		}
	}
	return ""
}

func dedupeTags(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func mergeTags(a, b []string) []string {
	return dedupeTags(append(append([]string{}, a...), b...))
}

func trimBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 400 {
		return s[:400] + "..."
	}
	return s
}

func fetchBearerToken(ctx context.Context, client *http.Client, wwwAuth, apiBase, repo string, user, pass *string) (string, error) {
	params, err := parseBearerChallenge(wwwAuth)
	if err != nil {
		return "", err
	}
	realmURL, err := url.Parse(params.Realm)
	if err != nil {
		return "", err
	}
	q := realmURL.Query()
	if params.Service != "" {
		q.Set("service", params.Service)
	}
	if params.Scope == "" {
		q.Set("scope", fmt.Sprintf("repository:%s:pull", repo))
	} else {
		q.Set("scope", params.Scope)
	}
	realmURL.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, realmURL.String(), nil)
	if err != nil {
		return "", err
	}
	if user != nil && pass != nil && *user != "" {
		req.SetBasicAuth(*user, *pass)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint %d: %s", resp.StatusCode, trimBody(body))
	}
	var tok struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", err
	}
	if tok.AccessToken != "" {
		return tok.AccessToken, nil
	}
	return tok.Token, nil
}

type bearerParams struct {
	Realm, Service, Scope string
}

func parseBearerChallenge(header string) (*bearerParams, error) {
	if header == "" {
		return nil, errors.New("empty challenge")
	}
	lower := strings.ToLower(header)
	idx := strings.Index(lower, "bearer ")
	if idx < 0 {
		return nil, fmt.Errorf("not a bearer challenge")
	}
	rest := header[idx+len("Bearer "):]
	out := &bearerParams{}
	for _, part := range splitAuthParts(rest) {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(strings.ToLower(k))
		v = strings.Trim(strings.TrimSpace(v), `"`)
		switch k {
		case "realm":
			out.Realm = v
		case "service":
			out.Service = v
		case "scope":
			out.Scope = v
		}
	}
	if out.Realm == "" {
		return nil, fmt.Errorf("missing realm")
	}
	return out, nil
}

func splitAuthParts(s string) []string {
	var parts []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			inQuote = !inQuote
			cur.WriteByte(c)
			continue
		}
		if c == ',' && !inQuote {
			parts = append(parts, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

// ResolveAPIBase 将用户选择的类型与输入地址解析为 OCI API 根 URL（含 scheme）。
func ResolveAPIBase(kind, host string) (string, error) {
	kind = strings.TrimSpace(strings.ToLower(kind))
	h := strings.TrimSpace(host)
	if h == "" {
		return "", fmt.Errorf("host is required")
	}
	if !strings.Contains(h, "://") {
		h = "https://" + h
	}
	u, err := url.Parse(h)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("invalid scheme")
	}
	hostName := strings.ToLower(u.Hostname())
	if kind == "dockerhub" || hostName == "docker.io" || hostName == "registry.hub.docker.com" {
		return "https://registry-1.docker.io", nil
	}
	return strings.TrimRight(u.String(), "/"), nil
}

// DefaultExampleHost 用于 UI 占位
func DefaultExampleHost(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ghcr":
		return "ghcr.io"
	case "dockerhub":
		return "docker.io"
	case "gitlab":
		return "registry.gitlab.com"
	case "quay":
		return "quay.io"
	case "harbor", "docker_registry":
		return "harbor.example.com"
	case "ecr":
		return "123456789012.dkr.ecr.us-east-1.amazonaws.com"
	case "gar":
		return "us-docker.pkg.dev"
	case "acr":
		return "myregistry.azurecr.io"
	case "aliyun":
		return "registry.cn-hangzhou.aliyuncs.com"
	default:
		return "registry.example.com"
	}
}

// ValidateKind 是否为支持的预设类型（含 custom）
func ValidateKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ghcr", "dockerhub", "gitlab", "quay", "harbor", "docker_registry", "ecr", "gar", "acr", "aliyun", "custom":
		return true
	default:
		return false
	}
}
