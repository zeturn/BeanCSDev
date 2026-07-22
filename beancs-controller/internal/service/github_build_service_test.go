package service

import (
	"strings"
	"testing"

	"github.com/zeturn/beancs-controller/internal/model"
)

func TestBuildWorkflowUsesWatchPathsForAutoDeploy(t *testing.T) {
	project := &model.Project{
		Name:                   "araneae-front",
		GitHubRepo:             "zeturn/AraneaeDev",
		GitHubBranch:           "main",
		BuildContext:           "Frontend",
		DockerfilePath:         "Frontend/Dockerfile",
		AutoDeploy:             true,
		WatchPaths:             model.StringList{"Frontend/**", "package.json", "pnpm-lock.yaml"},
		RegistryImageReference: "registry.beancs.hollowdata.com/hollowdata/araneae-front",
	}

	workflow := beancsBuildWorkflow(project, false)
	assertWorkflowContains(t, workflow, "branches:")
	assertWorkflowContains(t, workflow, "- 'main'")
	assertWorkflowContains(t, workflow, "paths:")
	assertWorkflowContains(t, workflow, "- 'Frontend/**'")
	assertWorkflowContains(t, workflow, "- 'package.json'")
	assertWorkflowContains(t, workflow, "- '.beancs/app.yaml'")
}

func TestBuildWorkflowFallsBackToBuildContextWatchPath(t *testing.T) {
	project := &model.Project{
		Name:                   "araneae-control",
		GitHubRepo:             "zeturn/AraneaeDev",
		GitHubBranch:           "main",
		BuildContext:           "Backend",
		DockerfilePath:         "Backend/Dockerfile",
		AutoDeploy:             true,
		RegistryImageReference: "registry.beancs.hollowdata.com/hollowdata/araneae-control",
	}

	workflow := beancsBuildWorkflow(project, false)
	assertWorkflowContains(t, workflow, "- 'Backend/**'")
	if strings.Contains(workflow, "- '**'") {
		t.Fatalf("expected build context path fallback, got repo-wide fallback:\n%s", workflow)
	}
}

func TestBuildWorkflowUsesBeanCSRegistryOnly(t *testing.T) {
	project := &model.Project{
		Name:                   "araneae-control",
		GitHubRepo:             "zeturn/AraneaeDev",
		BuildContext:           "Backend",
		DockerfilePath:         "Backend/Dockerfile",
		RegistryImageReference: "registry.beancs.hollowdata.com/hollowdata/araneae-control",
		BuildArgs: model.JSONMap{
			"BP_CLIENT_ID":     "araneae-control",
			"BP_CALLBACK_PATH": "/api/auth/basaltpass/callback/",
		},
	}

	workflow := beancsBuildWorkflow(project, true)
	assertWorkflowContains(t, workflow, "BEANCS_IMAGE_BASE: registry.beancs.hollowdata.com/hollowdata/araneae-control")
	assertWorkflowContains(t, workflow, "registry: ${{ secrets.BEANCS_REGISTRY_HOST }}")
	assertWorkflowContains(t, workflow, "BEANCS_IMAGE: ${{ steps.meta.outputs.image }}")
	assertWorkflowContains(t, workflow, "if: always() && github.event_name != 'workflow_dispatch'")
	assertWorkflowContains(t, workflow, "          BP_CALLBACK_PATH=/api/auth/basaltpass/callback/")
	assertWorkflowContains(t, workflow, "          BP_CLIENT_ID=araneae-control")
	assertWorkflowNotContains(t, workflow, "ghcr.io/")
	assertWorkflowNotContains(t, workflow, "ghcr_image")
	assertWorkflowNotContains(t, workflow, "BEANCS_GHCR_IMAGE_BASE")
	assertWorkflowNotContains(t, workflow, "packages: write")
}

func TestBuildImageReferenceRequiresConfiguredImageBase(t *testing.T) {
	if got := buildImageReference(&model.Project{Name: "araneae-control", GitHubRepo: "zeturn/AraneaeDev"}); got != "" {
		t.Fatalf("expected no implicit GHCR fallback, got %q", got)
	}

	got := buildImageReference(&model.Project{RegistryImageReference: "registry.beancs.hollowdata.com/hollowdata/araneae-control:latest"})
	if !strings.HasPrefix(got, "registry.beancs.hollowdata.com/hollowdata/araneae-control:beancs-") {
		t.Fatalf("expected BeanCS registry timestamp tag, got %q", got)
	}
}

func assertWorkflowContains(t *testing.T, value, needle string) {
	t.Helper()
	if !strings.Contains(value, needle) {
		t.Fatalf("expected %q to contain %q", value, needle)
	}
}

func assertWorkflowNotContains(t *testing.T, value, needle string) {
	t.Helper()
	if strings.Contains(value, needle) {
		t.Fatalf("expected %q not to contain %q", value, needle)
	}
}
