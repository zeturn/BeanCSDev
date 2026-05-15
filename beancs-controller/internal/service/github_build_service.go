package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/zeturn/beancs-controller/internal/config"
	"github.com/zeturn/beancs-controller/internal/model"
	"golang.org/x/crypto/nacl/box"
	"gorm.io/gorm"
)

const beancsLegacyBuildWorkflowPath = ".github/workflows/beancs-build.yml"

type GitHubBuildService struct {
	db          *gorm.DB
	cfg         *config.Config
	credentials *CredentialService
	gitops      *GitOpsService
}

func NewGitHubBuildService(db *gorm.DB, cfg *config.Config, credentials *CredentialService, gitops *GitOpsService) *GitHubBuildService {
	return &GitHubBuildService{db: db, cfg: cfg, credentials: credentials, gitops: gitops}
}

func (s *GitHubBuildService) Start(ctx context.Context, project *model.Project, triggeredBy string) (*model.Deployment, error) {
	if project == nil || project.BuildSource != model.BuildSourceGitHub || project.GitHubCredentialID == 0 || strings.TrimSpace(project.GitHubRepo) == "" {
		return nil, nil
	}
	var cred model.GitHubCredential
	if err := s.db.WithContext(ctx).First(&cred, project.GitHubCredentialID).Error; err != nil {
		return nil, err
	}
	token, err := s.credentials.GitHubToken(ctx, cred)
	if err != nil {
		return nil, err
	}
	owner, repo, ok := splitRepo(project.GitHubRepo)
	if !ok {
		return nil, fmt.Errorf("github_repo must be in owner/repo format")
	}
	branch := strings.TrimSpace(project.GitHubBranch)
	if branch == "" {
		branch = "main"
	}
	image := buildImageReference(project)
	if err := s.ensureWorkflow(ctx, token, owner, repo, project); err != nil {
		return nil, err
	}
	dispatchedAt := time.Now().UTC()
	if err := s.dispatchWorkflow(ctx, token, owner, repo, branch, project, image); err != nil {
		return nil, err
	}
	deployment := &model.Deployment{
		ProjectID:   project.ID,
		Tag:         image,
		ImageRef:    image,
		CommitSHA:   branch,
		Status:      "building",
		TriggeredBy: triggeredBy,
	}
	if err := s.db.WithContext(ctx).Create(deployment).Error; err != nil {
		return nil, err
	}
	go s.watchBuild(context.Background(), deployment.ID, dispatchedAt)
	return deployment, nil
}

func (s *GitHubBuildService) EnsureProjectWorkflow(ctx context.Context, project *model.Project) error {
	if project == nil || project.BuildSource != model.BuildSourceGitHub || project.GitHubCredentialID == 0 || strings.TrimSpace(project.GitHubRepo) == "" {
		return nil
	}
	var cred model.GitHubCredential
	if err := s.db.WithContext(ctx).First(&cred, project.GitHubCredentialID).Error; err != nil {
		return err
	}
	token, err := s.credentials.GitHubToken(ctx, cred)
	if err != nil {
		return err
	}
	owner, repo, ok := splitRepo(project.GitHubRepo)
	if !ok {
		return fmt.Errorf("github_repo must be in owner/repo format")
	}
	return s.ensureWorkflow(ctx, token, owner, repo, project)
}

func (s *GitHubBuildService) ensureWorkflow(ctx context.Context, token, owner, repo string, project *model.Project) error {
	client := github.NewClient(nil).WithAuthToken(token)
	webhookURL := s.webhookURL()
	if project.AutoDeploy && webhookURL == "" {
		return fmt.Errorf("BEANCS_WEBHOOK_HOST or BEANCS_PUBLIC_HOST is required for automatic GitHub push deployment")
	}
	callbackEnabled := false
	if webhookURL != "" {
		var err error
		callbackEnabled, err = s.ensureWebhookSecrets(ctx, token, owner, repo, webhookURL)
		if err != nil {
			if !isGitHubSecretsPermissionError(err) {
				return err
			}
			callbackEnabled = false
		}
	}
	if err := s.ensureRegistrySecrets(ctx, token, owner, repo); err != nil {
		return err
	}
	workflowPath := beancsBuildWorkflowPath(project)
	if err := putFile(ctx, client, owner, repo, workflowPath, beancsBuildWorkflow(project, callbackEnabled), "beancs: add "+project.Name+" build workflow"); err != nil {
		return githubWorkflowFilePermissionError(owner, repo, workflowPath, err)
	}
	if workflowPath != beancsLegacyBuildWorkflowPath {
		_ = deleteFileIfExists(ctx, client, owner, repo, beancsLegacyBuildWorkflowPath, "beancs: remove legacy build workflow")
	}
	return nil
}

func (s *GitHubBuildService) DeleteProjectWorkflow(ctx context.Context, project *model.Project) error {
	if project == nil || s == nil || s.credentials == nil || project.GitHubCredentialID == 0 || strings.TrimSpace(project.GitHubRepo) == "" {
		return nil
	}
	var cred model.GitHubCredential
	if err := s.db.WithContext(ctx).First(&cred, project.GitHubCredentialID).Error; err != nil {
		return err
	}
	token, err := s.credentials.GitHubToken(ctx, cred)
	if err != nil {
		return err
	}
	owner, repo, ok := splitRepo(project.GitHubRepo)
	if !ok {
		return fmt.Errorf("github_repo must be in owner/repo format")
	}
	client := github.NewClient(nil).WithAuthToken(token)
	workflowPath := beancsBuildWorkflowPath(project)
	if err := deleteFileIfExists(ctx, client, owner, repo, workflowPath, "beancs: remove "+project.Name+" build workflow"); err != nil {
		return githubWorkflowFilePermissionError(owner, repo, workflowPath, err)
	}
	return nil
}

func (s *GitHubBuildService) ensureRegistrySecrets(ctx context.Context, token, owner, repo string) error {
	if s.cfg == nil || strings.TrimSpace(s.cfg.RegistryHost) == "" {
		return nil
	}
	if strings.TrimSpace(s.cfg.RegistryUsername) == "" || strings.TrimSpace(s.cfg.RegistryToken) == "" {
		return fmt.Errorf("BEANCS_REGISTRY_USERNAME and BEANCS_REGISTRY_TOKEN are required to write the BeanCS registry push secrets")
	}
	secrets := map[string]string{
		"BEANCS_REGISTRY_HOST":     normalizeRegistryHost(s.cfg.RegistryHost),
		"BEANCS_REGISTRY_USERNAME": strings.TrimSpace(s.cfg.RegistryUsername),
		"BEANCS_REGISTRY_TOKEN":    s.cfg.RegistryToken,
	}
	for name, value := range secrets {
		if err := s.putRepositorySecret(ctx, token, owner, repo, name, value); err != nil {
			return githubSecretsPermissionError(owner, repo, err)
		}
	}
	return nil
}

func (s *GitHubBuildService) ensureWebhookSecrets(ctx context.Context, token, owner, repo, webhookURL string) (bool, error) {
	if err := s.putRepositorySecret(ctx, token, owner, repo, "BEANCS_WEBHOOK_URL", webhookURL); err != nil {
		return false, githubSecretsPermissionError(owner, repo, err)
	}
	if s.cfg != nil && s.cfg.WebhookSecret != "" {
		if err := s.putRepositorySecret(ctx, token, owner, repo, "BEANCS_WEBHOOK_SECRET", s.cfg.WebhookSecret); err != nil {
			return false, githubSecretsPermissionError(owner, repo, err)
		}
	}
	return true, nil
}

func (s *GitHubBuildService) dispatchWorkflow(ctx context.Context, token, owner, repo, branch string, project *model.Project, image string) error {
	inputs := map[string]any{
		"image":           image,
		"ghcr_image":      ghcrDispatchImage(project, image),
		"dockerfile_path": coalesce(project.DockerfilePath, "Dockerfile"),
		"context":         ".",
	}
	body, _ := json.Marshal(map[string]any{"ref": branch, "inputs": inputs})
	workflowFile := beancsBuildWorkflowFile(project)
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(workflowFile))
	var lastErr error
	for attempt := 0; attempt < 12; attempt++ {
		if err := githubRequest(ctx, http.MethodPost, endpoint, token, body, nil); err != nil {
			if !isGitHubAPIStatus(err, http.StatusNotFound) {
				return githubActionsPermissionError(owner, repo, err)
			}
			lastErr = err
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
			}
			continue
		}
		return nil
	}
	return githubWorkflowDispatchNotFoundError(owner, repo, branch, beancsBuildWorkflowPath(project), workflowFile, lastErr)
}

func (s *GitHubBuildService) watchBuild(ctx context.Context, deploymentID uint, dispatchedAt time.Time) {
	if err := s.reconcileDeployment(ctx, deploymentID, dispatchedAt); err != nil {
		_ = s.db.WithContext(ctx).Model(&model.Deployment{}).Where("id = ?", deploymentID).Updates(map[string]any{"status": "failed", "failure_reason": truncateFailure(err.Error())}).Error
	}
}

func (s *GitHubBuildService) ReconcileBuilding(ctx context.Context) error {
	var deployments []model.Deployment
	if err := s.db.WithContext(ctx).Where("status = ?", "building").Order("created_at asc").Limit(20).Find(&deployments).Error; err != nil {
		return err
	}
	for _, deployment := range deployments {
		if err := s.reconcileDeployment(ctx, deployment.ID, deployment.CreatedAt); err != nil {
			_ = s.db.WithContext(ctx).Model(&deployment).Updates(map[string]any{"status": "failed", "failure_reason": truncateFailure(err.Error())}).Error
		}
	}
	return nil
}

func (s *GitHubBuildService) StartReconciler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.ReconcileBuilding(ctx)
		}
	}
}

func (s *GitHubBuildService) reconcileDeployment(ctx context.Context, deploymentID uint, dispatchedAt time.Time) error {
	var deployment model.Deployment
	if err := s.db.WithContext(ctx).First(&deployment, deploymentID).Error; err != nil {
		return err
	}
	if deployment.Status != "building" {
		return nil
	}
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, deployment.ProjectID).Error; err != nil {
		return err
	}
	var cred model.GitHubCredential
	if err := s.db.WithContext(ctx).First(&cred, project.GitHubCredentialID).Error; err != nil {
		return err
	}
	token, err := s.credentials.GitHubToken(ctx, cred)
	if err != nil {
		return err
	}
	owner, repo, ok := splitRepo(project.GitHubRepo)
	if !ok {
		return fmt.Errorf("github_repo must be in owner/repo format")
	}
	branch := coalesce(project.GitHubBranch, "main")
	runID := deployment.WorkflowRunID
	if runID == 0 {
		run, err := s.waitForRun(ctx, token, owner, repo, beancsBuildWorkflowFile(&project), branch, dispatchedAt)
		if err != nil {
			return err
		}
		runID = run.ID
		_ = s.db.WithContext(ctx).Model(&deployment).Updates(map[string]any{"workflow_run_id": run.ID, "workflow_url": run.HTMLURL}).Error
	}
	run, err := s.workflowRun(ctx, token, owner, repo, runID)
	if err != nil {
		return err
	}
	if run.Status != "completed" {
		return nil
	}
	if run.Conclusion != "success" {
		return fmt.Errorf("github workflow concluded with %s", coalesce(run.Conclusion, "unknown"))
	}
	image := strings.TrimSpace(deployment.ImageRef)
	if image == "" {
		image = strings.TrimSpace(deployment.Tag)
	}
	if image == "" {
		image = buildImageReference(&project)
	}
	project.ImageReference = image
	if err := s.db.WithContext(ctx).Model(&project).Updates(map[string]any{"image_reference": image}).Error; err != nil {
		return err
	}
	if err := s.gitops.UpdateImageTag(ctx, token, cred, &project, image); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&deployment).Updates(map[string]any{
		"status":         "deploying",
		"commit_sha":     branch,
		"image_ref":      image,
		"failure_reason": "",
	}).Error
}

type workflowRun struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HTMLURL    string `json:"html_url"`
	CreatedAt  string `json:"created_at"`
}

func (s *GitHubBuildService) waitForRun(ctx context.Context, token, owner, repo, workflowFile, branch string, dispatchedAt time.Time) (workflowRun, error) {
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		runs, err := s.workflowRuns(ctx, token, owner, repo, workflowFile, branch)
		if err != nil {
			if isGitHubAPIStatus(err, http.StatusNotFound) {
				time.Sleep(5 * time.Second)
				continue
			}
			return workflowRun{}, err
		}
		for _, run := range runs {
			created, _ := time.Parse(time.RFC3339, run.CreatedAt)
			if run.ID != 0 && !created.Before(dispatchedAt.Add(-15*time.Second)) {
				return run, nil
			}
		}
		time.Sleep(5 * time.Second)
	}
	return workflowRun{}, fmt.Errorf("github workflow run was not created")
}

func (s *GitHubBuildService) waitForConclusion(ctx context.Context, token, owner, repo string, runID int64) (string, error) {
	deadline := time.Now().Add(45 * time.Minute)
	for time.Now().Before(deadline) {
		run, err := s.workflowRun(ctx, token, owner, repo, runID)
		if err != nil {
			return "", err
		}
		if run.Status == "completed" {
			return run.Conclusion, nil
		}
		time.Sleep(15 * time.Second)
	}
	return "", fmt.Errorf("github workflow run timed out")
}

func (s *GitHubBuildService) workflowRuns(ctx context.Context, token, owner, repo, workflowFile, branch string) ([]workflowRun, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/workflows/%s/runs?event=workflow_dispatch&branch=%s&per_page=10", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(workflowFile), url.QueryEscape(branch))
	var out struct {
		WorkflowRuns []workflowRun `json:"workflow_runs"`
	}
	if err := githubJSON(ctx, http.MethodGet, endpoint, token, &out); err != nil {
		return nil, err
	}
	return out.WorkflowRuns, nil
}

func (s *GitHubBuildService) workflowRun(ctx context.Context, token, owner, repo string, runID int64) (workflowRun, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs/%d", url.PathEscape(owner), url.PathEscape(repo), runID)
	var out workflowRun
	if err := githubJSON(ctx, http.MethodGet, endpoint, token, &out); err != nil {
		return workflowRun{}, err
	}
	return out, nil
}

func buildImageReference(project *model.Project) string {
	base := strings.TrimSpace(project.ImageReference)
	if base == "" {
		base = strings.TrimSpace(project.RegistryImageReference)
	}
	if base == "" {
		base = "ghcr.io/" + strings.ToLower(project.GitHubRepo)
	}
	if strings.Contains(base, "@") {
		base = strings.Split(base, "@")[0]
	}
	lastSlash := strings.LastIndex(base, "/")
	lastColon := strings.LastIndex(base, ":")
	if lastColon > lastSlash {
		base = base[:lastColon]
	}
	return base + ":beancs-" + time.Now().UTC().Format("20060102150405")
}

func ghcrDispatchImage(project *model.Project, image string) string {
	base := ghcrImageBase(project)
	if base == "" {
		base = "ghcr.io/" + strings.ToLower(project.GitHubRepo)
	}
	tag := "latest"
	if i := strings.LastIndex(image, ":"); i > strings.LastIndex(image, "/") {
		tag = image[i+1:]
	}
	return base + ":" + tag
}

func (s *GitHubBuildService) webhookURL() string {
	if s.cfg == nil || s.cfg.WebhookBaseURL() == "" {
		return ""
	}
	return s.cfg.WebhookBaseURL() + "/api/v1/webhooks/github"
}

func (s *GitHubBuildService) putRepositorySecret(ctx context.Context, token, owner, repo, name, value string) error {
	var key struct {
		KeyID string `json:"key_id"`
		Key   string `json:"key"`
	}
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/secrets/public-key", url.PathEscape(owner), url.PathEscape(repo))
	if err := githubJSON(ctx, http.MethodGet, endpoint, token, &key); err != nil {
		return err
	}
	encrypted, err := encryptGitHubSecret(key.Key, value)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"encrypted_value": encrypted, "key_id": key.KeyID})
	secretURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/secrets/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(name))
	return githubRequest(ctx, http.MethodPut, secretURL, token, body, nil)
}

func encryptGitHubSecret(publicKey, value string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return "", err
	}
	if len(decoded) != 32 {
		return "", fmt.Errorf("invalid GitHub repository public key")
	}
	var key [32]byte
	copy(key[:], decoded)
	encrypted, err := box.SealAnonymous(nil, []byte(value), &key, rand.Reader)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func githubRequest(ctx context.Context, method, endpoint, token string, body []byte, out any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &githubAPIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(raw))}
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("GitHub returned invalid JSON")
		}
	}
	return nil
}

type githubAPIError struct {
	StatusCode int
	Body       string
}

func (e *githubAPIError) Error() string {
	return fmt.Sprintf("GitHub API request failed: %s", e.Body)
}

func isGitHubAPIStatus(err error, status int) bool {
	var apiErr *githubAPIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == status
}

func githubSecretsPermissionError(owner, repo string, err error) error {
	var apiErr *githubAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden && strings.Contains(apiErr.Body, "Resource not accessible by integration") {
		return fmt.Errorf("GitHub App installation for %s/%s cannot manage repository Actions secrets. Update the GitHub App permissions to include Repository permissions: Contents read/write, Workflows read/write, Actions read/write, and Secrets read/write, then reinstall or approve the app installation", owner, repo)
	}
	return err
}

func isGitHubSecretsPermissionError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *githubAPIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusForbidden && strings.Contains(apiErr.Body, "Resource not accessible by integration")
	}
	message := err.Error()
	return strings.Contains(message, "cannot manage repository Actions secrets") ||
		(strings.Contains(message, "Resource not accessible by integration") && strings.Contains(message, "actions/secrets"))
}

func githubActionsPermissionError(owner, repo string, err error) error {
	var apiErr *githubAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden && strings.Contains(apiErr.Body, "Resource not accessible by integration") {
		return fmt.Errorf("GitHub App installation for %s/%s cannot dispatch GitHub Actions workflows. Update the GitHub App permissions to include Repository permissions: Contents read/write, Workflows read/write, and Actions read/write, then reinstall or approve the app installation", owner, repo)
	}
	return err
}

func githubWorkflowFilePermissionError(owner, repo, workflowPath string, err error) error {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusForbidden && strings.Contains(ghErr.Message, "Resource not accessible by integration") {
		return fmt.Errorf("GitHub App installation for %s/%s cannot create, update, or delete %s. Update the GitHub App permissions to include Repository permissions: Contents read/write, Workflows read/write, Actions read/write, then reinstall or approve the app installation", owner, repo, workflowPath)
	}
	if strings.Contains(err.Error(), "Resource not accessible by integration") {
		return fmt.Errorf("GitHub App installation for %s/%s cannot create, update, or delete %s. Update the GitHub App permissions to include Repository permissions: Contents read/write, Workflows read/write, Actions read/write, then reinstall or approve the app installation", owner, repo, workflowPath)
	}
	return err
}

func githubWorkflowDispatchNotFoundError(owner, repo, branch, workflowPath, workflowFile string, err error) error {
	if err == nil {
		return fmt.Errorf("GitHub Actions workflow %s for %s/%s is not available yet. Retry in a minute, or make sure branch %q contains %s", workflowFile, owner, repo, branch, workflowPath)
	}
	return fmt.Errorf("GitHub Actions workflow %s for %s/%s is not available yet. GitHub may still be indexing the new workflow, or branch %q does not contain %s. Retry in a minute, or make sure the selected branch exists and contains the workflow file: %w", workflowFile, owner, repo, branch, workflowPath, err)
}

func truncateFailure(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 1000 {
		return value[:1000]
	}
	return value
}

func beancsBuildWorkflow(project *model.Project, callbackEnabled bool) string {
	projectName := strings.TrimSpace(project.Name)
	workflowSlug := beancsWorkflowSlug(project)
	imageBase := strings.TrimSpace(project.RegistryImageReference)
	if imageBase == "" {
		imageBase = strings.TrimSpace(project.ImageReference)
	}
	if imageBase == "" {
		imageBase = "ghcr.io/" + strings.ToLower(project.GitHubRepo)
	}
	ghcrBase := ghcrImageBase(project)
	if ghcrBase == "" {
		ghcrBase = "ghcr.io/" + strings.ToLower(project.GitHubRepo)
	}
	trigger := `  workflow_dispatch:
    inputs:
      image:
        description: Target BeanCS registry image reference
        required: true
      ghcr_image:
        description: Target GHCR image reference
        required: false
      dockerfile_path:
        description: Dockerfile path
        required: false
        default: Dockerfile
      context:
        description: Docker build context
        required: false
        default: .
`
	if project.AutoDeploy {
		trigger += fmt.Sprintf(`  push:
    tags:
      - '%s-v*'
      - 'all-v*'
`, workflowSlug)
	}
	callback := ""
	if callbackEnabled {
		callback = fmt.Sprintf(`      - name: Notify BeanCS
        if: always()
        env:
          BEANCS_PROJECT: %s
          BEANCS_WEBHOOK_URL: ${{ secrets.BEANCS_WEBHOOK_URL }}
          BEANCS_WEBHOOK_SECRET: ${{ secrets.BEANCS_WEBHOOK_SECRET }}
          BEANCS_STATUS: ${{ job.status }}
          BEANCS_IMAGE: ${{ steps.meta.outputs.image }}
        run: |
          STATUS="failure"
          if [ "$BEANCS_STATUS" = "success" ]; then STATUS="success"; fi
          BODY=$(jq -nc --arg project "$BEANCS_PROJECT" --arg tag "$BEANCS_IMAGE" --arg commit "$GITHUB_SHA" --arg status "$STATUS" '{project:$project, tag:$tag, commit:$commit, status:$status}')
          SIG="sha256=$(printf '%%s' "$BODY" | openssl dgst -sha256 -hmac "$BEANCS_WEBHOOK_SECRET" -binary | xxd -p -c 256)"
          curl -fsS -X POST "$BEANCS_WEBHOOK_URL" -H "Content-Type: application/json" -H "X-Hub-Signature-256: $SIG" --data "$BODY" || echo "BeanCS webhook notification failed; build result remains valid."
`, projectName)
	}
	return fmt.Sprintf(`name: BeanCS Build %s

on:
%s

permissions:
  contents: read
  packages: write

jobs:
  build:
    runs-on: ubuntu-latest
    env:
      BEANCS_PROJECT: %s
      BEANCS_IMAGE_BASE: %s
      BEANCS_GHCR_IMAGE_BASE: %s
    steps:
      - uses: actions/checkout@v4
      - name: Resolve image
        id: meta
        shell: bash
        run: |
          if [ "${{ github.event_name }}" = "workflow_dispatch" ]; then
            IMAGE="${{ inputs.image }}"
            GHCR_IMAGE="${{ inputs.ghcr_image }}"
          elif [[ "$GITHUB_REF" == refs/tags/* ]]; then
            TAG="${GITHUB_REF#refs/tags/}"
            VERSION_TAG="$TAG"
            VERSION_TAG="${VERSION_TAG#${BEANCS_PROJECT}-}"
            VERSION_TAG="${VERSION_TAG#all-}"
            IMAGE="${BEANCS_IMAGE_BASE}:${VERSION_TAG}"
            GHCR_IMAGE="${BEANCS_GHCR_IMAGE_BASE}:${VERSION_TAG}"
          else
            IMAGE="${BEANCS_IMAGE_BASE}:beancs-${GITHUB_SHA::12}"
            GHCR_IMAGE="${BEANCS_GHCR_IMAGE_BASE}:beancs-${GITHUB_SHA::12}"
          fi
          if [ -z "$GHCR_IMAGE" ]; then
            TAG="${IMAGE##*:}"
            GHCR_IMAGE="${BEANCS_GHCR_IMAGE_BASE}:${TAG}"
          fi
          echo "image=$IMAGE" >> "$GITHUB_OUTPUT"
          echo "ghcr_image=$GHCR_IMAGE" >> "$GITHUB_OUTPUT"
      - uses: docker/setup-qemu-action@v3
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/login-action@v3
        with:
          registry: ${{ secrets.BEANCS_REGISTRY_HOST }}
          username: ${{ secrets.BEANCS_REGISTRY_USERNAME }}
          password: ${{ secrets.BEANCS_REGISTRY_TOKEN }}
      - uses: docker/build-push-action@v6
        with:
          context: ${{ github.event_name == 'workflow_dispatch' && inputs.context || '.' }}
          file: ${{ github.event_name == 'workflow_dispatch' && inputs.dockerfile_path || '%s' }}
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ${{ steps.meta.outputs.image }}
            ${{ steps.meta.outputs.ghcr_image }}
%s`, projectName, trigger, projectName, imageBase, ghcrBase, coalesce(project.DockerfilePath, "Dockerfile"), callback)
}

func beancsWorkflowSlug(project *model.Project) string {
	if project == nil {
		return "project"
	}
	if slug := harborName(project.Name); slug != "" {
		return slug
	}
	return "project"
}

func beancsBuildWorkflowFile(project *model.Project) string {
	return "beancs-build-" + beancsWorkflowSlug(project) + ".yml"
}

func beancsBuildWorkflowPath(project *model.Project) string {
	return ".github/workflows/" + beancsBuildWorkflowFile(project)
}
