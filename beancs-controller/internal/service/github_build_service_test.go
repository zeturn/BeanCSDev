package service

import (
	"strings"
	"testing"

	"github.com/zeturn/beancs-controller/internal/model"
)

func TestBuildWorkflowUsesWatchPathsForAutoDeploy(t *testing.T) {
	project := &model.Project{
		Name:           "araneae-front",
		GitHubRepo:     "zeturn/AraneaeDev",
		GitHubBranch:   "main",
		BuildContext:   "Frontend",
		DockerfilePath: "Frontend/Dockerfile",
		AutoDeploy:     true,
		WatchPaths:     model.StringList{"Frontend/**", "package.json", "pnpm-lock.yaml"},
	}

	workflow := beancsBuildWorkflow(project, false)
	assertContains(t, workflow, "branches:")
	assertContains(t, workflow, "- 'main'")
	assertContains(t, workflow, "paths:")
	assertContains(t, workflow, "- 'Frontend/**'")
	assertContains(t, workflow, "- 'package.json'")
	assertContains(t, workflow, "- '.beancs/app.yaml'")
}

func TestBuildWorkflowFallsBackToBuildContextWatchPath(t *testing.T) {
	project := &model.Project{
		Name:           "araneae-control",
		GitHubRepo:     "zeturn/AraneaeDev",
		GitHubBranch:   "main",
		BuildContext:   "Backend",
		DockerfilePath: "Backend/Dockerfile",
		AutoDeploy:     true,
	}

	workflow := beancsBuildWorkflow(project, false)
	assertContains(t, workflow, "- 'Backend/**'")
	if strings.Contains(workflow, "- '**'") {
		t.Fatalf("expected build context path fallback, got repo-wide fallback:\n%s", workflow)
	}
}
