package service

import (
	"testing"

	"github.com/zeturn/beancs-controller/internal/config"
	"github.com/zeturn/beancs-controller/internal/model"
)

func TestConfigureBeanCSRegistryUsesProjectNameRepository(t *testing.T) {
	cfg := &config.Config{
		RegistryHost:       "https://registry.example.com/",
		RegistryPullSecret: "pull-secret",
	}
	project := &model.Project{
		Name:       "araneae-control",
		GitHubRepo: "zeturn/AraneaeDev",
	}

	if err := configureBeanCSRegistry(project, cfg, "Hollow_Data"); err != nil {
		t.Fatal(err)
	}

	if project.RegistryRepository != "araneae-control" {
		t.Fatalf("RegistryRepository = %q, want araneae-control", project.RegistryRepository)
	}
	if project.RegistryImageReference != "registry.example.com/hollow-data/araneae-control" {
		t.Fatalf("RegistryImageReference = %q", project.RegistryImageReference)
	}
	if project.ImageReference != project.RegistryImageReference {
		t.Fatalf("ImageReference = %q, want registry image reference", project.ImageReference)
	}
}
