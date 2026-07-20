package service

import (
	"testing"

	"github.com/zeturn/beancs-controller/internal/model"
)

func TestUsesGitOps(t *testing.T) {
	project := &model.Project{BuildSource: model.BuildSourceGitHub}
	cred := model.GitHubCredential{GitOpsRepo: "zeturn/beancsnotes"}
	if !UsesGitOps(project, cred) {
		t.Fatalf("expected github project with gitops repo to use gitops")
	}
}

func TestUsesGitOpsFalseForRegistrySource(t *testing.T) {
	project := &model.Project{BuildSource: model.BuildSourceRegistry}
	cred := model.GitHubCredential{GitOpsRepo: "zeturn/beancsnotes"}
	if UsesGitOps(project, cred) {
		t.Fatalf("expected registry source project not to use gitops")
	}
}
