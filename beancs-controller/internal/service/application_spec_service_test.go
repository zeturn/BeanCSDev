package service

import (
	"testing"

	appspec "github.com/zeturn/beancs-controller/internal/application/spec"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/model"
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

	req := (&ApplicationSpecService{}).specToMonorepoRequest(nil, "", doc, dto.ApplicationSpecRepoRequest{GitHubCredentialID: 42})
	if req.GitHubCredentialID != 42 {
		t.Fatalf("github credential id = %d", req.GitHubCredentialID)
	}
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

func TestSpecToMonorepoRequestPreservesBasaltPassConfig(t *testing.T) {
	doc := &appspec.ApplicationSpecDocument{
		Metadata: appspec.ApplicationMetadata{Name: "araneae"},
		Spec: appspec.ApplicationSpec{
			Type: "monorepo",
			Repo: appspec.RepoSpec{Name: "zeturn/AraneaeDev", Branch: "main"},
			Components: []appspec.ComponentSpec{{
				Name:        "control",
				Kind:        "service",
				ProjectName: "araneae-control",
				BasaltPass: &appspec.BasaltPassSpec{
					CallbackPath: "/api/auth/basaltpass/callback/",
					Scopes:       []string{"openid", "profile"},
				},
			}},
		},
	}

	req := (&ApplicationSpecService{}).specToMonorepoRequest(nil, "", doc, dto.ApplicationSpecRepoRequest{GitHubCredentialID: 42})
	if req.Components[0].BasaltPass == nil {
		t.Fatal("expected basaltPass config")
	}
	if got := req.Components[0].BasaltPass.CallbackPath; got != "/api/auth/basaltpass/callback/" {
		t.Fatalf("callbackPath = %q", got)
	}
	if got := req.Components[0].BasaltPass.Scopes[1]; got != "profile" {
		t.Fatalf("scope[1] = %q", got)
	}
}

func TestBasaltPassRedirectURIsUsesCallbackPath(t *testing.T) {
	project := &model.Project{Name: "araneae-control", Namespace: "app-araneae", Domain: "araneae-control.hollowdata.com"}
	uris := basaltPassRedirectURIs(project, &dto.BasaltPassComponentConfig{CallbackPath: "/api/auth/basaltpass/callback/"})
	if len(uris) != 1 || uris[0] != "https://araneae-control.hollowdata.com/api/auth/basaltpass/callback/" {
		t.Fatalf("redirect uris = %#v", uris)
	}
}

func TestApplyComponentDomainsFillsRoutableHosts(t *testing.T) {
	component := dto.MonorepoComponentRequest{
		ProjectName: "araneae-control",
		Ports: model.ProjectPorts{
			{Name: "http", Port: 8180, Protocol: "http", Exposure: model.ExposurePublic},
			{Name: "grpc", Port: 9190, Protocol: "grpc", Exposure: model.ExposurePrivate},
		},
	}

	applyComponentDomains(&component, "app-araneae", "hollowdata.com", nil)

	if got := component.Ports[0].Domain; got != "araneae-control.hollowdata.com" {
		t.Fatalf("public domain = %q", got)
	}
	if got := component.Ports[1].Domain; got != "araneae-control.app-araneae.ts.net" {
		t.Fatalf("private domain = %q", got)
	}
	if got := component.Ports[1].Protocol; got != "" {
		t.Fatalf("expected unsupported project protocol to be cleared, got %q", got)
	}
}

func TestApplyComponentDomainsUsesOverrides(t *testing.T) {
	component := dto.MonorepoComponentRequest{
		Name:        "front",
		ProjectName: "araneae-front",
		Ports: model.ProjectPorts{
			{Name: "http", Port: 80, Protocol: "http", Exposure: model.ExposurePublic},
		},
	}

	applyComponentDomains(&component, "app-araneae", "hollowdata.com", map[string]string{
		"araneae-front": "spider.hollowdata.com",
	})

	if got := component.Ports[0].Domain; got != "spider.hollowdata.com" {
		t.Fatalf("override domain = %q", got)
	}
}

func TestSpecToMonorepoRequestPreservesPortDomain(t *testing.T) {
	doc := &appspec.ApplicationSpecDocument{
		Metadata: appspec.ApplicationMetadata{Name: "issuetick"},
		Spec: appspec.ApplicationSpec{
			Type: "monorepo",
			Repo: appspec.RepoSpec{Name: "zeturn/IssueTick", Branch: "master"},
			Components: []appspec.ComponentSpec{{
				Name:        "frontend",
				Kind:        "frontend",
				ProjectName: "issuetick-frontend",
				Ports: []appspec.PortSpec{{
					Name:     "http",
					Port:     80,
					Protocol: "http",
					Exposure: "public",
					Domain:   "IssueTick.BeanCS.com.",
				}},
			}},
		},
	}

	req := (&ApplicationSpecService{}).specToMonorepoRequest(nil, "", doc, dto.ApplicationSpecRepoRequest{GitHubCredentialID: 42})
	if got := req.Components[0].Ports[0].Domain; got != "issuetick.beancs.com" {
		t.Fatalf("port domain = %q", got)
	}
}

func TestApplyComponentDomainsDoesNotReusePublicOverrideForPrivatePort(t *testing.T) {
	component := dto.MonorepoComponentRequest{
		Name:        "control",
		ProjectName: "araneae-control",
		Ports: model.ProjectPorts{
			{Name: "http", Port: 8180, Protocol: "http", Exposure: model.ExposurePublic},
			{Name: "grpc", Port: 9190, Protocol: "grpc", Exposure: model.ExposurePrivate},
		},
	}

	applyComponentDomains(&component, "app-araneae", "hollowdata.com", map[string]string{
		"araneae-control": "araneae-control.hollowdata.com",
	})

	if got := component.Ports[0].Domain; got != "araneae-control.hollowdata.com" {
		t.Fatalf("public override domain = %q", got)
	}
	if got := component.Ports[1].Domain; got != "araneae-control.app-araneae.ts.net" {
		t.Fatalf("private fallback domain = %q", got)
	}
}
