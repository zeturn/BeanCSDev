package service

import (
	"strings"

	"github.com/zeturn/beancs-controller/internal/model"
)

func UsesGitOps(project *model.Project, credential model.GitHubCredential) bool {
	if project == nil {
		return false
	}
	return project.BuildSource == model.BuildSourceGitHub &&
		strings.TrimSpace(credential.GitOpsRepo) != ""
}
