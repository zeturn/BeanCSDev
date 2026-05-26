package service

import "testing"

func TestComposeComponentsInferBuildsAndPorts(t *testing.T) {
	meta := repositoryProjectMeta{ServicePorts: []repositoryProjectPort{{Service: "backend_api", Port: 8105}}}
	components, err := composeComponents([]byte(`
services:
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "8105:8105"
  frontend:
    build: ./frontend
    ports:
      - "5108:80"
  postgres:
    image: postgres:16
`), meta)
	if err != nil {
		t.Fatal(err)
	}
	if len(components) != 2 {
		t.Fatalf("components = %d", len(components))
	}
	backend := components[0]
	if backend.Name != "backend" || backend.BuildContext != "backend" || backend.DockerfilePath != "backend/Dockerfile" || backend.SuggestedPort != 8105 {
		t.Fatalf("backend component = %#v", backend)
	}
	frontend := components[1]
	if frontend.Name != "frontend" || frontend.BuildContext != "frontend" || frontend.DockerfilePath != "frontend/Dockerfile" || frontend.SuggestedPort != 80 || frontend.Kind != "frontend" {
		t.Fatalf("frontend component = %#v", frontend)
	}
}

func TestDockerfileVariantsInferRootComponents(t *testing.T) {
	if !isDockerfileName("backend.Dockerfile") {
		t.Fatal("expected backend.Dockerfile to be recognized")
	}
	if got := dockerfileComponentPath("backend.Dockerfile"); got != "backend" {
		t.Fatalf("component path = %q", got)
	}
	if got := componentNameFromDockerfile("frontend.Dockerfile"); got != "frontend" {
		t.Fatalf("component name = %q", got)
	}
}
