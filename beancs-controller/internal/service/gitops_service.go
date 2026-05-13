package service

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/google/go-github/v62/github"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

var newTagPattern = regexp.MustCompile(`(?m)^(\s*newTag:\s*)(.+)$`)

type GitOpsService struct {
	db          *gorm.DB
	credentials *CredentialService
}

func NewGitOpsService(db *gorm.DB, credentials *CredentialService) *GitOpsService {
	return &GitOpsService{db: db, credentials: credentials}
}

// resolveGitOpsToken returns the correct token for accessing the GitOps repo.
// If the GitOps repo owner differs from the provided credential's account,
// it looks up a credential whose account_login matches the repo owner.
func (s *GitOpsService) resolveGitOpsToken(ctx context.Context, fallbackToken string, cred model.GitHubCredential) (string, error) {
	owner, _, ok := resolveGitOpsRepo(cred)
	if !ok {
		return fallbackToken, nil
	}
	// If the credential already belongs to the GitOps repo owner, use it directly
	if strings.EqualFold(cred.AccountLogin, owner) || strings.EqualFold(cred.Org, owner) {
		return fallbackToken, nil
	}
	// Look up a credential that matches the GitOps repo owner
	if s.db != nil && s.credentials != nil {
		var match model.GitHubCredential
		err := s.db.WithContext(ctx).
			Where("(account_login = ? OR org = ?) AND is_active = true", owner, owner).
			First(&match).Error
		if err == nil {
			if token, err := s.credentials.GitHubToken(ctx, match); err == nil {
				return token, nil
			}
		}
	}
	// Fall back to the provided token (will likely 404, but keeps existing behavior)
	return fallbackToken, nil
}

func (s *GitOpsService) CommitProjectManifests(ctx context.Context, token string, cred model.GitHubCredential, project *model.Project) error {
	if token == "" || cred.GitOpsRepo == "" {
		return nil
	}
	owner, repo, ok := resolveGitOpsRepo(cred)
	if !ok {
		return fmt.Errorf("gitops repo must be owner/repo when org is empty")
	}
	token, err := s.resolveGitOpsToken(ctx, token, cred)
	if err != nil {
		return err
	}
	client := github.NewClient(nil).WithAuthToken(token)
	files := s.RenderManifests(project)
	msg := fmt.Sprintf("beancs: add %s manifests", project.Name)
	for p, content := range files {
		if err := putFile(ctx, client, owner, repo, p, content, msg); err != nil {
			return err
		}
	}
	return nil
}

// UpdateImageTag updates only the newTag field in the overlay kustomization.yaml.
// This is much lighter than CommitProjectManifests — it modifies a single line in one file.
func (s *GitOpsService) UpdateImageTag(ctx context.Context, token string, cred model.GitHubCredential, project *model.Project, newImageRef string) error {
	if token == "" || cred.GitOpsRepo == "" {
		return nil
	}
	owner, repo, ok := resolveGitOpsRepo(cred)
	if !ok {
		return fmt.Errorf("gitops repo must be owner/repo when org is empty")
	}
	token, err := s.resolveGitOpsToken(ctx, token, cred)
	if err != nil {
		return err
	}
	newTag := extractImageTag(newImageRef)
	if newTag == "" {
		newTag = "latest"
	}
	filePath := path.Join("apps", project.Name, "overlays", "dev", "kustomization.yaml")
	client := github.NewClient(nil).WithAuthToken(token)

	// Read current kustomization.yaml
	current, _, resp, err := client.Repositories.GetContents(ctx, owner, repo, filePath, nil)
	if err != nil {
		if resp != nil && resp.Response != nil && resp.Response.StatusCode == 404 {
			// Overlay doesn't exist yet — fall back to full manifest commit
			return s.CommitProjectManifests(ctx, token, cred, project)
		}
		return fmt.Errorf("read gitops overlay: %w", err)
	}
	content, err := current.GetContent()
	if err != nil {
		return fmt.Errorf("decode gitops overlay: %w", err)
	}

	// Also update newName if the image base changed
	newName := extractImageName(newImageRef)
	updated := content
	if newName != "" {
		newNamePattern := regexp.MustCompile(`(?m)^(\s*newName:\s*)(.+)$`)
		updated = newNamePattern.ReplaceAllString(updated, "${1}"+newName)
	}

	// Replace newTag value
	if !newTagPattern.MatchString(updated) {
		// No newTag line found — fall back to full manifest commit
		return s.CommitProjectManifests(ctx, token, cred, project)
	}
	updated = newTagPattern.ReplaceAllString(updated, "${1}"+newTag)

	if updated == content {
		return nil // no change needed
	}

	msg := fmt.Sprintf("beancs(%s): update image to %s", project.Name, newTag)
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(msg),
		Content: []byte(updated),
		SHA:     current.SHA,
	}
	_, _, err = client.Repositories.UpdateFile(ctx, owner, repo, filePath, opts)
	return err
}

// DeleteProjectManifests removes the entire apps/<project>/ directory from the gitops repo.
func (s *GitOpsService) DeleteProjectManifests(ctx context.Context, token string, cred model.GitHubCredential, projectName string) error {
	if token == "" || cred.GitOpsRepo == "" {
		return nil
	}
	owner, repo, ok := resolveGitOpsRepo(cred)
	if !ok {
		return fmt.Errorf("gitops repo must be owner/repo when org is empty")
	}
	token, _ = s.resolveGitOpsToken(ctx, token, cred)
	client := github.NewClient(nil).WithAuthToken(token)
	dirPath := "apps/" + projectName

	// List all files in the project directory recursively
	files, err := listGitOpsFiles(ctx, client, owner, repo, dirPath)
	if err != nil {
		return nil // directory doesn't exist or error listing — skip silently
	}
	if len(files) == 0 {
		return nil
	}

	// Delete each file (GitHub API requires per-file deletion for contents API)
	msg := fmt.Sprintf("beancs: remove %s manifests", projectName)
	for _, f := range files {
		opts := &github.RepositoryContentFileOptions{
			Message: github.String(msg),
			SHA:     github.String(f.sha),
		}
		_, _, err := client.Repositories.DeleteFile(ctx, owner, repo, f.path, opts)
		if err != nil {
			return fmt.Errorf("delete gitops file %s: %w", f.path, err)
		}
	}
	return nil
}

type gitOpsFile struct {
	path string
	sha  string
}

// listGitOpsFiles recursively lists all files under a directory in the gitops repo.
func listGitOpsFiles(ctx context.Context, client *github.Client, owner, repo, dirPath string) ([]gitOpsFile, error) {
	_, dirContents, resp, err := client.Repositories.GetContents(ctx, owner, repo, dirPath, nil)
	if err != nil {
		if resp != nil && resp.Response != nil && resp.Response.StatusCode == 404 {
			return nil, nil
		}
		return nil, err
	}
	var files []gitOpsFile
	for _, entry := range dirContents {
		if entry.GetType() == "file" {
			files = append(files, gitOpsFile{path: entry.GetPath(), sha: entry.GetSHA()})
		} else if entry.GetType() == "dir" {
			subFiles, err := listGitOpsFiles(ctx, client, owner, repo, entry.GetPath())
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		}
	}
	return files, nil
}

// resolveGitOpsRepo extracts owner/repo from the GitHubCredential's GitOpsRepo field.
func resolveGitOpsRepo(cred model.GitHubCredential) (string, string, bool) {
	owner, repo, ok := splitRepo(cred.GitOpsRepo)
	if !ok {
		if cred.Org == "" {
			return "", "", false
		}
		owner, repo = cred.Org, cred.GitOpsRepo
		ok = true
	}
	return owner, repo, ok
}

// extractImageTag extracts the tag from a full image reference like "harbor.host/proj/app:v1.2.0".
func extractImageTag(imageRef string) string {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return ""
	}
	// Handle digest references (image@sha256:...)
	if atIdx := strings.Index(imageRef, "@"); atIdx >= 0 {
		imageRef = imageRef[:atIdx]
	}
	lastSlash := strings.LastIndex(imageRef, "/")
	lastColon := strings.LastIndex(imageRef, ":")
	if lastColon > lastSlash {
		return imageRef[lastColon+1:]
	}
	return ""
}

// extractImageName extracts the image name (without tag) from a full image reference.
func extractImageName(imageRef string) string {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return ""
	}
	if atIdx := strings.Index(imageRef, "@"); atIdx >= 0 {
		imageRef = imageRef[:atIdx]
	}
	lastSlash := strings.LastIndex(imageRef, "/")
	lastColon := strings.LastIndex(imageRef, ":")
	if lastColon > lastSlash {
		return imageRef[:lastColon]
	}
	return imageRef
}

func (s *GitOpsService) RenderManifests(project *model.Project) map[string]string {
	base := "apps/" + project.Name
	image := strings.TrimSpace(project.ImageReference)
	if image == "" {
		image = "ghcr.io/" + strings.ToLower(project.GitHubRepo) + ":latest"
	}
	imageName := extractImageName(image)
	imageTag := extractImageTag(image)
	if imageTag == "" {
		imageTag = "latest"
	}
	pullSecrets := ""
	if project.RegistryPullSecretName != "" {
		pullSecrets = fmt.Sprintf(`      imagePullSecrets:
        - name: %s
`, project.RegistryPullSecretName)
	}
	ports := project.Ports
	if len(ports) == 0 {
		ports = model.ProjectPorts{{Name: "http", Port: project.Port, Exposure: project.ExposureMode, Domain: project.Domain}}
	}
	return map[string]string{
		path.Join(base, "base", "deployment.yaml"): fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    managed-by: beancs
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
      managed-by: beancs
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1
  template:
    metadata:
      labels:
        app: %s
        managed-by: beancs
    spec:
%s
      tolerations:
        - key: node.kubernetes.io/not-ready
          operator: Exists
          effect: NoExecute
          tolerationSeconds: 30
        - key: node.kubernetes.io/unreachable
          operator: Exists
          effect: NoExecute
          tolerationSeconds: 30
      containers:
        - name: app
          image: %s
          ports:
%s
          envFrom:
            - secretRef:
                name: app-env-vars
`, project.Name, project.Namespace, project.Name, project.Replicas, project.Name, project.Name, pullSecrets, image, renderContainerPorts(ports)),
		path.Join(base, "base", "service.yaml"): fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    managed-by: beancs
spec:
  selector:
    app: %s
    managed-by: beancs
  ports:
%s
`, project.Name, project.Namespace, project.Name, project.Name, renderServicePorts(ports)),
		path.Join(base, "base", "kustomization.yaml"): `resources:
  - deployment.yaml
  - service.yaml
`,
		path.Join(base, "overlays", "dev", "kustomization.yaml"): fmt.Sprintf(`resources:
  - ../../base
images:
  - name: app
    newName: %s
    newTag: %s
`, imageName, imageTag),
	}
}

func renderContainerPorts(ports model.ProjectPorts) string {
	var b strings.Builder
	for _, p := range ports {
		fmt.Fprintf(&b, "            - name: %s\n              containerPort: %d\n", p.Name, p.Port)
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderServicePorts(ports model.ProjectPorts) string {
	var b strings.Builder
	for _, p := range ports {
		fmt.Fprintf(&b, "    - name: %s\n      port: %d\n      targetPort: %d\n", p.Name, p.Port, p.Port)
	}
	return strings.TrimRight(b.String(), "\n")
}

func putFile(ctx context.Context, client *github.Client, owner, repo, p, content, msg string) error {
	current, _, resp, err := client.Repositories.GetContents(ctx, owner, repo, p, nil)
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(msg),
		Content: []byte(content),
	}
	if err == nil && current != nil {
		opts.SHA = current.SHA
		_, _, err = client.Repositories.UpdateFile(ctx, owner, repo, p, opts)
		return err
	}
	if resp != nil && resp.Response != nil && resp.Response.StatusCode != 404 {
		return err
	}
	_, _, err = client.Repositories.CreateFile(ctx, owner, repo, p, opts)
	return err
}

func splitRepo(repo string) (string, string, bool) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
