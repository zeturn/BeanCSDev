package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zeturn/beancs-controller/internal/config"
	"github.com/zeturn/beancs-controller/internal/model"
)

type githubRepositoryMeta struct {
	ID            int64  `json:"id"`
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
}

func fetchGitHubRepositoryMeta(ctx context.Context, token, owner, repo string) (githubRepositoryMeta, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s", url.PathEscape(owner), url.PathEscape(repo))
	var out githubRepositoryMeta
	if err := githubJSON(ctx, http.MethodGet, endpoint, token, &out); err != nil {
		return out, err
	}
	if strings.TrimSpace(out.FullName) == "" {
		out.FullName = owner + "/" + repo
	}
	return out, nil
}

func configureBeanCSRegistry(project *model.Project, cfg *config.Config, tenantCode string) error {
	if project == nil || cfg == nil || strings.TrimSpace(cfg.RegistryHost) == "" || project.GitHubRepo == "" {
		return nil
	}
	_, repo, ok := splitRepo(project.GitHubRepo)
	if !ok {
		return fmt.Errorf("github_repo must be in owner/repo format")
	}
	host := normalizeRegistryHost(cfg.RegistryHost)
	registryProject := harborName(coalesce(tenantCode, project.TenantCode))
	registryRepo := harborName(repo)
	if registryProject == "" {
		return fmt.Errorf("BasaltPass tenant_code is required to create BeanCS registry projects")
	}
	project.RegistryHost = host
	project.RegistryProject = registryProject
	project.RegistryRepository = registryRepo
	project.RegistryPullSecretName = strings.TrimSpace(cfg.RegistryPullSecret)
	if project.RegistryPullSecretName == "" {
		project.RegistryPullSecretName = "beancs-registry-pull"
	}
	project.RegistryImageReference = host + "/" + registryProject + "/" + registryRepo
	project.ImageReference = project.RegistryImageReference
	return nil
}

func harborName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.'
		if !allowed {
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
			continue
		}
		b.WriteRune(r)
		lastDash = r == '-'
	}
	return strings.Trim(b.String(), "-.")
}

func ghcrImageBase(project *model.Project) string {
	if project == nil || strings.TrimSpace(project.GitHubRepoFullName) == "" {
		return ""
	}
	return "ghcr.io/" + strings.ToLower(project.GitHubRepoFullName)
}

func normalizeRegistryHost(raw string) string {
	raw = strings.TrimSpace(strings.TrimRight(raw, "/"))
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	return raw
}

func ensureHarborProject(ctx context.Context, cfg *config.Config, projectName string) error {
	if cfg == nil || strings.TrimSpace(cfg.RegistryHost) == "" || strings.TrimSpace(cfg.RegistryUsername) == "" || strings.TrimSpace(cfg.RegistryToken) == "" || strings.TrimSpace(projectName) == "" {
		return nil
	}
	base := "https://" + normalizeRegistryHost(cfg.RegistryHost)
	client := &http.Client{Timeout: 20 * time.Second}
	getURL := fmt.Sprintf("%s/api/v2.0/projects/%s", base, url.PathEscape(projectName))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.RegistryUsername, cfg.RegistryToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	_ = resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("Harbor project check failed: %s", strings.TrimSpace(string(body)))
	}
	payload, _ := json.Marshal(map[string]any{
		"project_name": projectName,
		"metadata": map[string]string{
			"public": "false",
		},
	})
	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v2.0/projects", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	createReq.SetBasicAuth(cfg.RegistryUsername, cfg.RegistryToken)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := client.Do(createReq)
	if err != nil {
		return err
	}
	defer createResp.Body.Close()
	createBody, _ := io.ReadAll(io.LimitReader(createResp.Body, 1<<20))
	if createResp.StatusCode == http.StatusConflict || (createResp.StatusCode >= 200 && createResp.StatusCode < 300) {
		return nil
	}
	return fmt.Errorf("Harbor project create failed: %s", strings.TrimSpace(string(createBody)))
}
