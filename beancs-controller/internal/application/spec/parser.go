package spec

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	APIVersionV1Alpha1 = "beancs.io/v1alpha1"
	KindApplication    = "Application"
)

func Parse(content []byte) (*ApplicationSpecDocument, error) {
	var doc ApplicationSpecDocument
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("parse application spec yaml: %w", err)
	}
	ApplyDefaults(&doc)
	return &doc, nil
}

func ApplyDefaults(doc *ApplicationSpecDocument) {
	if doc == nil {
		return
	}
	doc.APIVersion = strings.TrimSpace(doc.APIVersion)
	doc.Kind = strings.TrimSpace(doc.Kind)
	doc.Metadata.Name = strings.TrimSpace(doc.Metadata.Name)
	doc.Spec.Type = defaultString(strings.TrimSpace(doc.Spec.Type), "monorepo")
	doc.Spec.Repo.Provider = defaultString(strings.TrimSpace(doc.Spec.Repo.Provider), "github")
	doc.Spec.Repo.Name = strings.TrimSpace(doc.Spec.Repo.Name)
	doc.Spec.Repo.Branch = defaultString(strings.TrimSpace(doc.Spec.Repo.Branch), "main")
	if doc.Spec.Namespace == nil {
		doc.Spec.Namespace = &NamespaceSpec{Strategy: "per-project"}
	}
	doc.Spec.Namespace.Strategy = defaultString(strings.TrimSpace(doc.Spec.Namespace.Strategy), "per-project")
	doc.Spec.Namespace.Name = strings.TrimSpace(doc.Spec.Namespace.Name)
	if doc.Spec.AutoDeploy == nil {
		doc.Spec.AutoDeploy = &AutoDeploySpec{Enabled: false, Mode: "disabled"}
	}
	if !doc.Spec.AutoDeploy.Enabled {
		doc.Spec.AutoDeploy.Mode = "disabled"
	} else {
		doc.Spec.AutoDeploy.Mode = defaultString(strings.TrimSpace(doc.Spec.AutoDeploy.Mode), "all")
	}
	for i := range doc.Spec.Dependencies {
		dep := &doc.Spec.Dependencies[i]
		dep.Name = strings.TrimSpace(dep.Name)
		dep.Type = strings.TrimSpace(dep.Type)
		dep.DeployMethod = strings.TrimSpace(dep.DeployMethod)
	}
	for i := range doc.Spec.Components {
		component := &doc.Spec.Components[i]
		component.Name = strings.TrimSpace(component.Name)
		component.Kind = defaultString(strings.TrimSpace(component.Kind), "service")
		component.ProjectName = strings.TrimSpace(component.ProjectName)
		if component.Build != nil {
			component.Build.Context = defaultString(strings.TrimSpace(component.Build.Context), ".")
			component.Build.Dockerfile = strings.TrimSpace(component.Build.Dockerfile)
		}
		if component.Image != nil {
			component.Image.Repository = strings.TrimSpace(component.Image.Repository)
			component.Image.TagPolicy = defaultString(strings.TrimSpace(component.Image.TagPolicy), "git-sha")
		}
		for j := range component.Ports {
			component.Ports[j].Name = strings.TrimSpace(component.Ports[j].Name)
			component.Ports[j].Protocol = defaultString(strings.TrimSpace(component.Ports[j].Protocol), "tcp")
			component.Ports[j].Exposure = defaultString(strings.TrimSpace(component.Ports[j].Exposure), "private")
		}
		for j := range component.Volumes {
			if len(component.Volumes[j].AccessModes) == 0 {
				component.Volumes[j].AccessModes = []string{"ReadWriteOnce"}
			}
		}
	}
}

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
