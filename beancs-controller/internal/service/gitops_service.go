package service

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v62/github"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

var newTagPattern = regexp.MustCompile(`(?m)^(\s*newTag:\s*)(.+)$`)

type GitOpsService struct {
	db                 *gorm.DB
	credentials        *CredentialService
	dependencyRegistry *DependencyDefinitionRegistry
}

func NewGitOpsService(db *gorm.DB, credentials *CredentialService) *GitOpsService {
	return &GitOpsService{db: db, credentials: credentials}
}

func (s *GitOpsService) SetDependencyRegistry(registry *DependencyDefinitionRegistry) {
	s.dependencyRegistry = registry
}

// resolveGitOpsToken returns the correct token for accessing the GitOps repo.
// If the GitOps repo owner differs from the provided credential's account,
// it looks up a credential whose account_login matches the repo owner.
func (s *GitOpsService) resolveGitOpsToken(ctx context.Context, fallbackToken string, cred model.GitHubCredential) (string, error) {
	match := s.resolveGitOpsCredential(ctx, cred)
	if match.ID == cred.ID {
		return fallbackToken, nil
	}
	if s.credentials != nil {
		if token, err := s.credentials.GitHubToken(ctx, match); err == nil {
			return token, nil
		}
	}
	return fallbackToken, nil
}

func (s *GitOpsService) resolveGitOpsCredential(ctx context.Context, cred model.GitHubCredential) model.GitHubCredential {
	owner, _, ok := resolveGitOpsRepo(cred)
	if !ok {
		return cred
	}
	// If the credential already belongs to the GitOps repo owner, use it directly
	if strings.EqualFold(cred.AccountLogin, owner) || strings.EqualFold(cred.Org, owner) {
		return cred
	}
	// Look up a credential that matches the GitOps repo owner
	if s.db != nil && s.credentials != nil {
		var match model.GitHubCredential
		err := s.db.WithContext(ctx).
			Where("(account_login = ? OR org = ?) AND is_active = true", owner, owner).
			First(&match).Error
		if err == nil && match.ID != 0 {
			return match
		}
	}
	return cred
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

func (s *GitOpsService) CommitDependencyManifests(ctx context.Context, token string, cred model.GitHubCredential, app model.Application, dep model.ManagedDependency) error {
	if token == "" || cred.GitOpsRepo == "" || s.dependencyRegistry == nil {
		return nil
	}
	def, ok := s.dependencyRegistry.Get(dep.DefinitionName)
	if !ok {
		def, ok = s.dependencyRegistry.Get(dep.Type)
	}
	if !ok {
		return fmt.Errorf("dependency definition %q not found", dep.DefinitionName)
	}
	if dep.DeployMethod != model.DependencyDeployMethodHelm {
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
	files := s.RenderDependencyManifests(app, dep, def)
	msg := fmt.Sprintf("beancs: add %s dependency %s", app.Name, dep.Name)
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
		imageNamePattern := regexp.MustCompile(`(?m)^(\s*-\s*name:\s*)(.+)$`)
		updated = imageNamePattern.ReplaceAllString(updated, "${1}"+newName)
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
		image = strings.TrimSpace(project.RegistryImageReference)
	}
	if image == "" {
		image = project.Name + ":latest"
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
	if len(ports) == 0 && project.Port > 0 && project.ExposureMode != model.ExposureInternalOnly {
		ports = model.ProjectPorts{{Name: "http", Port: project.Port, Exposure: project.ExposureMode, Domain: project.Domain}}
	}
	volumes := project.VolumeConfig()
	files := map[string]string{
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
%s
      containers:
        - name: app
          image: %s
%s
          envFrom:
            - secretRef:
                name: %s
%s%s
`, project.Name, project.Namespace, project.Name, project.Replicas, project.Name, project.Name, pullSecrets, renderPodVolumesBlock(volumes, project.Name), image, renderContainerPortsBlock(ports), project.EnvSecretName(), renderProbeBlock(project.HealthCheckConfig(), ports), renderContainerVolumeMountsBlock(volumes)),
		path.Join(base, "base", "kustomization.yaml"): renderBaseKustomization(ports, volumes),
		path.Join(base, "overlays", "dev", "kustomization.yaml"): fmt.Sprintf(`resources:
  - ../../base
images:
  - name: %s
    newName: %s
    newTag: %s
`, imageName, imageName, imageTag),
	}
	if len(ports) > 0 {
		files[path.Join(base, "base", "service.yaml")] = fmt.Sprintf(`apiVersion: v1
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
`, project.Name, project.Namespace, project.Name, project.Name, renderServicePorts(ports))
	}
	for _, volume := range volumes {
		if strings.EqualFold(volume.Type, "pvc") {
			files[path.Join(base, "base", "pvc-"+volume.Name+".yaml")] = renderPVCManifest(project, volume)
		}
	}
	return files
}

func (s *GitOpsService) RenderDependencyManifests(app model.Application, dep model.ManagedDependency, def DependencyDefinition) map[string]string {
	base := path.Join("apps", app.Name, "dependencies", dep.Name)
	values := renderDependencyHelmValues(dep)
	return map[string]string{
		path.Join(base, "application.yaml"): renderDependencyArgoApplication(app, dep, def, values),
		path.Join(base, "values.yaml"):      values,
		path.Join(base, "kustomization.yaml"): `resources:
  - application.yaml
`,
	}
}

func renderContainerPortsBlock(ports model.ProjectPorts) string {
	if len(ports) == 0 {
		return ""
	}
	return "          ports:\n" + renderContainerPorts(ports)
}

func renderDependencyArgoApplication(app model.Application, dep model.ManagedDependency, def DependencyDefinition, values string) string {
	appName := dependencyArgoApplicationName(app.Name, dep.Name)
	chartRepo := strings.TrimSpace(def.Spec.Helm.Chart.Repo)
	chartName := strings.TrimSpace(def.Spec.Helm.Chart.Name)
	chartVersion := strings.TrimSpace(def.Spec.Helm.Chart.Version)
	if chartVersion == "" {
		chartVersion = "*"
	}
	targetRevision := chartVersion
	if strings.Contains(chartVersion, "*") {
		targetRevision = yamlScalar(chartVersion)
	}
	return fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: argocd
  labels:
    app: %s
    managed-by: beancs
    beancs.io/application: %s
    beancs.io/dependency: %s
spec:
  project: default
  source:
    repoURL: %s
    chart: %s
    targetRevision: %s
    helm:
      releaseName: %s
      values: |
%s
  destination:
    server: https://kubernetes.default.svc
    namespace: %s
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
`, appName, dep.Name, app.Name, dep.Name, chartRepo, chartName, targetRevision, dep.ServiceName, indentYAMLBlock(values, 8), dep.Namespace)
}

func renderDependencyHelmValues(dep model.ManagedDependency) string {
	switch dep.Type {
	case "rabbitmq":
		return renderRabbitMQValues(dep)
	case "postgresql":
		return renderPostgreSQLValues(dep)
	case "timescaledb":
		return renderTimescaleDBValues(dep)
	case "mysql":
		return renderMySQLValues(dep)
	case "redis":
		return renderRedisValues(dep)
	default:
		return fmt.Sprintf("fullnameOverride: %s\n", dep.ServiceName)
	}
}

func renderRabbitMQValues(dep model.ManagedDependency) string {
	username := dependencyOutputValue(dep, "username")
	if username == "" {
		username = fmt.Sprint(dep.Config["username"])
	}
	if username == "" {
		username = "user"
	}
	return fmt.Sprintf(`fullnameOverride: %s
global:
  security:
    allowInsecureImages: true
image:
  registry: docker.io
  repository: bitnamilegacy/rabbitmq
  tag: 4.1.3-debian-12-r1
auth:
  username: %s
  existingPasswordSecret: %s
  existingSecretPasswordKey: rabbitmq-password
persistence:
  enabled: %s
%s
  size: %s
`, dep.ServiceName, yamlScalar(username), dep.SecretName, yamlBool(configBool(dep.Config, "persistence.enabled", true)), renderDependencyStorageClass(dep, 2), yamlScalar(configString(dep.Config, "persistence.size", "8Gi")))
}

func renderPostgreSQLValues(dep model.ManagedDependency) string {
	username := dependencyOutputValue(dep, "username")
	if username == "" {
		username = fmt.Sprint(dep.Config["username"])
	}
	database := dependencyOutputValue(dep, "database")
	if database == "" {
		database = fmt.Sprint(dep.Config["database"])
	}
	return fmt.Sprintf(`fullnameOverride: %s
global:
  security:
    allowInsecureImages: true
image:
  registry: docker.io
  repository: bitnamilegacy/postgresql
  tag: latest
auth:
  username: %s
  database: %s
  existingSecret: %s
  customPasswordFiles:
    user: password
primary:
  persistence:
    enabled: %s
%s
    size: %s
`, dep.ServiceName, yamlScalar(coalesce(username, "app")), yamlScalar(coalesce(database, "app")), dep.SecretName, yamlBool(configBool(dep.Config, "persistence.enabled", true)), renderDependencyStorageClass(dep, 4), yamlScalar(configString(dep.Config, "persistence.size", "20Gi")))
}

func renderTimescaleDBValues(dep model.ManagedDependency) string {
	return fmt.Sprintf(`fullnameOverride: %s
replicaCount: %d
secrets:
  credentialsSecretName: %s
persistentVolumes:
  data:
    enabled: %s
%s
    size: %s
  wal:
    enabled: %s
%s
    size: %s
networkPolicy:
  enabled: false
`, dep.ServiceName, configInt(dep.Config, "replica_count", 1), yamlScalar(dep.SecretName), yamlBool(configBool(dep.Config, "persistence.enabled", true)), renderDependencyStorageClass(dep, 4), yamlScalar(configString(dep.Config, "persistence.size", "20Gi")), yamlBool(configBool(dep.Config, "persistence.enabled", true)), renderDependencyStorageClass(dep, 4), yamlScalar(configString(dep.Config, "persistence.walSize", "5Gi")))
}

func renderMySQLValues(dep model.ManagedDependency) string {
	username := dependencyOutputValue(dep, "username")
	if username == "" {
		username = fmt.Sprint(dep.Config["username"])
	}
	database := dependencyOutputValue(dep, "database")
	if database == "" {
		database = fmt.Sprint(dep.Config["database"])
	}
	return fmt.Sprintf(`fullnameOverride: %s
global:
  security:
    allowInsecureImages: true
image:
  registry: docker.io
  repository: bitnamilegacy/mysql
  tag: 9.4.0-debian-12-r1
auth:
  username: %s
  database: %s
  existingSecret: %s
primary:
  persistence:
    enabled: %s
%s
    size: %s
`, dep.ServiceName, yamlScalar(coalesce(username, "app")), yamlScalar(coalesce(database, "app")), dep.SecretName, yamlBool(configBool(dep.Config, "persistence.enabled", true)), renderDependencyStorageClass(dep, 4), yamlScalar(configString(dep.Config, "persistence.size", "20Gi")))
}

func renderRedisValues(dep model.ManagedDependency) string {
	architecture := fmt.Sprint(dep.Config["architecture"])
	if architecture == "" {
		architecture = "standalone"
	}
	return fmt.Sprintf(`fullnameOverride: %s
architecture: %s
auth:
  enabled: true
  existingSecret: %s
  existingSecretPasswordKey: password
master:
  persistence:
    enabled: %s
%s
    size: %s
`, dep.ServiceName, yamlScalar(architecture), dep.SecretName, yamlBool(configBool(dep.Config, "persistence.enabled", true)), renderDependencyStorageClass(dep, 4), yamlScalar(configString(dep.Config, "persistence.size", "8Gi")))
}

func renderDependencyStorageClass(dep model.ManagedDependency, spaces int) string {
	storageClass := strings.TrimSpace(configString(dep.Config, "persistence.storageClass", ""))
	if storageClass == "" {
		storageClass = strings.TrimSpace(configString(dep.Config, "persistence.storageClassName", ""))
	}
	if storageClass == "" {
		return ""
	}
	return fmt.Sprintf("%sstorageClass: %s\n", strings.Repeat(" ", spaces), yamlScalar(storageClass))
}

func dependencySecretRuntimeData(dep model.ManagedDependency) map[string]string {
	data := map[string]string{}
	for key, raw := range dep.Outputs {
		if m, ok := raw.(map[string]any); ok {
			data[key] = fmt.Sprint(m["value"])
			continue
		}
		data[key] = fmt.Sprint(raw)
	}
	if password := dependencyOutputValue(dep, "password"); password != "" {
		data["password"] = password
		switch dep.Type {
		case "mysql":
			data["mysql-password"] = password
			data["mysql-root-password"] = password
			data["mysql-replication-password"] = password
		case "rabbitmq":
			data["rabbitmq-password"] = password
		case "postgresql":
			data["postgres-password"] = password
			data["replication-password"] = password
		case "timescaledb":
			data["PATRONI_SUPERUSER_PASSWORD"] = password
			data["PATRONI_REPLICATION_PASSWORD"] = password
			data["PATRONI_admin_PASSWORD"] = password
		}
	}
	if username := dependencyOutputValue(dep, "username"); username != "" {
		data["username"] = username
	}
	if database := dependencyOutputValue(dep, "database"); database != "" {
		data["database"] = database
	}
	return data
}

func dependencyOutputValue(dep model.ManagedDependency, key string) string {
	raw, ok := dep.Outputs[key]
	if !ok {
		return ""
	}
	if m, ok := raw.(map[string]any); ok {
		return fmt.Sprint(m["value"])
	}
	return fmt.Sprint(raw)
}

func configString(config model.JSONMap, path, fallback string) string {
	value, ok := nestedConfigValue(config, path)
	if !ok || fmt.Sprint(value) == "" {
		return fallback
	}
	return fmt.Sprint(value)
}

func configBool(config model.JSONMap, path string, fallback bool) bool {
	value, ok := nestedConfigValue(config, path)
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	default:
		return fallback
	}
}

func configInt(config model.JSONMap, path string, fallback int) int {
	value, ok := nestedConfigValue(config, path)
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func nestedConfigValue(config model.JSONMap, path string) (any, bool) {
	var current any = config
	for _, part := range strings.Split(path, ".") {
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[part]
			if !ok {
				return nil, false
			}
			current = next
		case model.JSONMap:
			next, ok := typed[part]
			if !ok {
				return nil, false
			}
			current = next
		default:
			return nil, false
		}
	}
	return current, true
}

func yamlBool(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func yamlScalar(v string) string {
	if v == "" {
		return `""`
	}
	escaped := strings.ReplaceAll(v, `"`, `\"`)
	return `"` + escaped + `"`
}

func indentYAMLBlock(value string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.TrimRight(value, "\n"), "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func dependencyArgoApplicationName(appName, depName string) string {
	name := appName + "-" + depName
	if len(name) <= 63 {
		return name
	}
	return name[:63]
}

func dependencyRootArgoApplicationName(appName, depName string) string {
	name := dependencyArgoApplicationName(appName, depName) + "-root"
	if len(name) <= 63 {
		return name
	}
	return name[:63]
}

func renderBaseKustomization(ports model.ProjectPorts, volumes []model.ProjectVolume) string {
	var b strings.Builder
	b.WriteString("resources:\n  - deployment.yaml\n")
	if len(ports) > 0 {
		b.WriteString("  - service.yaml\n")
	}
	for _, volume := range volumes {
		if strings.EqualFold(volume.Type, "pvc") {
			fmt.Fprintf(&b, "  - pvc-%s.yaml\n", volume.Name)
		}
	}
	return b.String()
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

func renderProbeBlock(health *model.ProjectHealthCheck, ports model.ProjectPorts) string {
	if health == nil {
		if len(ports) == 0 {
			return ""
		}
		health = &model.ProjectHealthCheck{Type: "http", Path: "/health", Port: ports[0].Port}
	}
	if strings.EqualFold(health.Type, "disabled") {
		return ""
	}
	return renderOneProbe("readinessProbe", health, ports, 5, 5) + renderOneProbe("livenessProbe", health, ports, 10, 10)
}

func renderOneProbe(name string, health *model.ProjectHealthCheck, ports model.ProjectPorts, initialDelay, period int) string {
	if health.InitialDelaySeconds != nil {
		initialDelay = *health.InitialDelaySeconds
	}
	if health.PeriodSeconds != nil {
		period = *health.PeriodSeconds
	}
	timeout := 0
	if health.TimeoutSeconds != nil {
		timeout = *health.TimeoutSeconds
	}
	port := renderProbePort(health.Port, ports)
	var b strings.Builder
	fmt.Fprintf(&b, "          %s:\n", name)
	if strings.EqualFold(health.Type, "tcp") {
		fmt.Fprintf(&b, "            tcpSocket:\n              port: %s\n", port)
	} else {
		probePath := strings.TrimSpace(health.Path)
		if probePath == "" {
			probePath = "/health"
		}
		fmt.Fprintf(&b, "            httpGet:\n              path: %s\n              port: %s\n", probePath, port)
	}
	fmt.Fprintf(&b, "            initialDelaySeconds: %d\n            periodSeconds: %d\n", initialDelay, period)
	if timeout > 0 {
		fmt.Fprintf(&b, "            timeoutSeconds: %d\n", timeout)
	}
	return b.String()
}

func renderProbePort(port any, ports model.ProjectPorts) string {
	switch v := port.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	case float64:
		return strconv.Itoa(int(v))
	case int:
		return strconv.Itoa(v)
	case int32:
		return strconv.Itoa(int(v))
	case int64:
		return strconv.Itoa(int(v))
	}
	if len(ports) > 0 {
		if strings.TrimSpace(ports[0].Name) != "" {
			return ports[0].Name
		}
		return strconv.Itoa(ports[0].Port)
	}
	return "8080"
}

func renderPodVolumesBlock(volumes []model.ProjectVolume, projectName string) string {
	if len(volumes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("      volumes:\n")
	for _, volume := range volumes {
		fmt.Fprintf(&b, "        - name: %s\n", volume.Name)
		switch {
		case strings.EqualFold(volume.Type, "emptyDir"):
			b.WriteString("          emptyDir: {}\n")
		case strings.EqualFold(volume.Type, "pvc"):
			fmt.Fprintf(&b, "          persistentVolumeClaim:\n            claimName: %s\n", projectVolumeClaimName(projectName, volume.Name))
		}
	}
	return b.String()
}

func renderContainerVolumeMountsBlock(volumes []model.ProjectVolume) string {
	if len(volumes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("          volumeMounts:\n")
	for _, volume := range volumes {
		fmt.Fprintf(&b, "            - name: %s\n              mountPath: %s\n", volume.Name, volume.MountPath)
	}
	return b.String()
}

func renderPVCManifest(project *model.Project, volume model.ProjectVolume) string {
	accessModes := volume.AccessModes
	if len(accessModes) == 0 {
		accessModes = []string{"ReadWriteOnce"}
	}
	var modes strings.Builder
	for _, mode := range accessModes {
		fmt.Fprintf(&modes, "    - %s\n", mode)
	}
	return fmt.Sprintf(`apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    managed-by: beancs
spec:
  accessModes:
%s
  resources:
    requests:
      storage: %s
%s`, projectVolumeClaimName(project.Name, volume.Name), project.Namespace, project.Name, strings.TrimRight(modes.String(), "\n"), volume.Size, renderPVCStorageClassName(volume))
}

func renderPVCStorageClassName(volume model.ProjectVolume) string {
	storageClass := strings.TrimSpace(volume.StorageClassName)
	if storageClass == "" {
		return ""
	}
	return fmt.Sprintf("  storageClassName: %s\n", yamlScalar(storageClass))
}

func projectVolumeClaimName(projectName, volumeName string) string {
	return projectName + "-" + volumeName
}

func putFile(ctx context.Context, client *github.Client, owner, repo, p, content, msg string) error {
	const maxRetries = 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		current, _, resp, err := client.Repositories.GetContents(ctx, owner, repo, p, nil)
		opts := &github.RepositoryContentFileOptions{
			Message: github.String(msg),
			Content: []byte(content),
		}
		if err == nil && current != nil {
			opts.SHA = current.SHA
			_, _, updateErr := client.Repositories.UpdateFile(ctx, owner, repo, p, opts)
			if updateErr != nil && attempt < maxRetries && isConflict(updateErr) {
				continue // re-read SHA and retry
			}
			return updateErr
		}
		if resp != nil && resp.Response != nil && resp.Response.StatusCode != 404 {
			return err
		}
		_, _, createErr := client.Repositories.CreateFile(ctx, owner, repo, p, opts)
		if createErr != nil && attempt < maxRetries && isConflict(createErr) {
			continue // file may have been created concurrently
		}
		return createErr
	}
	return fmt.Errorf("putFile %s: max retries exceeded", p)
}

func deleteFileIfExists(ctx context.Context, client *github.Client, owner, repo, p, msg string) error {
	current, _, resp, err := client.Repositories.GetContents(ctx, owner, repo, p, nil)
	if err != nil {
		if resp != nil && resp.Response != nil && resp.Response.StatusCode == 404 {
			return nil
		}
		return err
	}
	if current == nil {
		return nil
	}
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(msg),
		SHA:     current.SHA,
	}
	_, _, err = client.Repositories.DeleteFile(ctx, owner, repo, p, opts)
	return err
}

func isConflict(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "409")
}

func splitRepo(repo string) (string, string, bool) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
