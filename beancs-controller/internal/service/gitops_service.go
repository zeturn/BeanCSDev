package service

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/google/go-github/v62/github"
	"github.com/zeturn/beancs-controller/internal/model"
)

type GitOpsService struct{}

func NewGitOpsService() *GitOpsService { return &GitOpsService{} }

func (s *GitOpsService) CommitProjectManifests(ctx context.Context, token string, cred model.GitHubCredential, project *model.Project) error {
	if token == "" || cred.GitOpsRepo == "" {
		return nil
	}
	owner, repo, ok := splitRepo(cred.GitOpsRepo)
	if !ok {
		if cred.Org == "" {
			return fmt.Errorf("gitops repo must be owner/repo when org is empty")
		}
		owner, repo = cred.Org, cred.GitOpsRepo
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

func (s *GitOpsService) RenderManifests(project *model.Project) map[string]string {
	base := "apps/" + project.Name
	image := "ghcr.io/" + strings.ToLower(project.GitHubRepo) + ":latest"
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
`, project.Name, project.Namespace, project.Name, project.Replicas, project.Name, project.Name, image, renderContainerPorts(ports)),
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
		path.Join(base, "overlays", "dev", "kustomization.yaml"): `resources:
  - ../../base
images:
  - name: app
    newName: ghcr.io/placeholder/app
    newTag: latest
`,
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
