package service

import (
	"testing"

	"github.com/zeturn/beancs-controller/internal/model"
)

func TestVersionFromImageRef(t *testing.T) {
	tests := map[string]string{
		"":       "",
		"v0.3.0": "v0.3.0",
		"registry.beancs.hollowdata.com/hollowdata/californiabeans:v0.3.0": "v0.3.0",
		"registry.beancs.hollowdata.com:5000/hollowdata/app:beancs-123456": "beancs-123456",
		"ghcr.io/zeturn/californiabeans@sha256:abc123":                     "sha256:abc123",
	}
	for input, want := range tests {
		if got := versionFromImageRef(input); got != want {
			t.Fatalf("versionFromImageRef(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDeploymentVersionPrefersImageRef(t *testing.T) {
	dep := model.Deployment{
		Tag:      "registry.example.com/app:v0.2.0",
		ImageRef: "registry.example.com/app:v0.3.0",
	}
	if got := deploymentVersion(dep); got != "v0.3.0" {
		t.Fatalf("deploymentVersion() = %q, want v0.3.0", got)
	}
}
