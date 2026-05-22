package service

import (
	"testing"

	appspec "github.com/zeturn/beancs-controller/internal/application/spec"
	"github.com/zeturn/beancs-controller/internal/dto"
)

func TestSpecToMonorepoRequestResolvesGeneratedComponentSecrets(t *testing.T) {
	length := 40
	doc := &appspec.ApplicationSpecDocument{
		Metadata: appspec.ApplicationMetadata{Name: "araneae"},
		Spec: appspec.ApplicationSpec{
			Type: "monorepo",
			Repo: appspec.RepoSpec{Name: "zeturn/AraneaeDev", Branch: "main"},
			Components: []appspec.ComponentSpec{
				{
					Name:        "control",
					Kind:        "service",
					ProjectName: "araneae-control",
					Env:         map[string]any{"CONTROL_HTTP_ADDR": ":8180"},
					Secrets: []appspec.SecretSpec{{
						Name:     "EXECUTION_CALLBACK_KEY",
						Generate: &appspec.GenerateSpec{Length: length},
					}},
				},
				{
					Name:        "executor",
					Kind:        "worker",
					ProjectName: "araneae-executor",
					Secrets: []appspec.SecretSpec{{
						Name:          "EXECUTION_CALLBACK_KEY",
						FromComponent: "control",
					}},
				},
			},
		},
	}

	req := (&ApplicationSpecService{}).specToMonorepoRequest(doc, dto.ApplicationSpecRepoRequest{})
	if len(req.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(req.Components))
	}
	controlKey := req.Components[0].Env["EXECUTION_CALLBACK_KEY"]
	executorKey := req.Components[1].Env["EXECUTION_CALLBACK_KEY"]
	if len(controlKey) != length {
		t.Fatalf("expected generated key length %d, got %d", length, len(controlKey))
	}
	if executorKey != controlKey {
		t.Fatalf("expected executor to reuse control key")
	}
}
