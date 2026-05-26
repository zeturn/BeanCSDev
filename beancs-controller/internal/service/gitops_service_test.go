package service

import (
	"path"
	"strings"
	"testing"

	"github.com/zeturn/beancs-controller/internal/model"
)

func TestRenderManifestsIncludesHealthCheckAndPVC(t *testing.T) {
	service := NewGitOpsService(nil, nil)
	project := &model.Project{
		Name:           "araneae-control",
		Namespace:      "app-araneae",
		GitHubRepo:     "zeturn/AraneaeDev",
		ImageReference: "registry.local/araneae-control:abc123",
		Replicas:       2,
		Ports: model.ProjectPorts{
			{Name: "http", Port: 8180, Exposure: model.ExposurePublic},
			{Name: "grpc", Port: 9190, Exposure: model.ExposurePrivate},
		},
		HealthCheck: model.JSONMap{
			"type": "http",
			"path": "/healthz",
			"port": "http",
		},
		Volumes: model.JSONMap{
			"items": []map[string]any{
				{"name": "data", "type": "pvc", "mountPath": "/data", "size": "10Gi"},
			},
		},
	}

	files := service.RenderManifests(project)
	deployment := files[path.Join("apps", project.Name, "base", "deployment.yaml")]
	assertContains(t, deployment, "path: /healthz")
	assertContains(t, deployment, "port: http")
	assertContains(t, deployment, "volumeMounts:")
	assertContains(t, deployment, "mountPath: /data")
	assertContains(t, deployment, "claimName: araneae-control-data")

	kustomization := files[path.Join("apps", project.Name, "base", "kustomization.yaml")]
	assertContains(t, kustomization, "- pvc-data.yaml")

	pvc := files[path.Join("apps", project.Name, "base", "pvc-data.yaml")]
	assertContains(t, pvc, "kind: PersistentVolumeClaim")
	assertContains(t, pvc, "storage: 10Gi")
}

func TestRenderManifestsIncludesTCPHealthCheck(t *testing.T) {
	service := NewGitOpsService(nil, nil)
	project := &model.Project{
		Name:           "araneae-front",
		Namespace:      "app-araneae",
		GitHubRepo:     "zeturn/AraneaeDev",
		ImageReference: "registry.local/araneae-front:abc123",
		Replicas:       1,
		Ports:          model.ProjectPorts{{Name: "http", Port: 80, Exposure: model.ExposurePublic}},
		HealthCheck: model.JSONMap{
			"type": "tcp",
			"port": "http",
		},
	}

	files := service.RenderManifests(project)
	deployment := files[path.Join("apps", project.Name, "base", "deployment.yaml")]
	assertContains(t, deployment, "tcpSocket:")
	assertContains(t, deployment, "port: http")
	if strings.Contains(deployment, "httpGet:") {
		t.Fatalf("expected tcp health check, got httpGet in deployment:\n%s", deployment)
	}
}

func TestRenderDependencyManifestsUsesExistingSecret(t *testing.T) {
	service := NewGitOpsService(nil, nil)
	registry, err := NewDependencyDefinitionRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, ok := registry.Get("rabbitmq")
	if !ok {
		t.Fatal("rabbitmq definition not found")
	}
	dep := model.ManagedDependency{
		Name:         "rabbitmq",
		Type:         "rabbitmq",
		Namespace:    "app-araneae",
		ServiceName:  "rabbitmq",
		SecretName:   "araneae-rabbitmq-credentials",
		DeployMethod: model.DependencyDeployMethodHelm,
		Config: model.JSONMap{
			"persistence": map[string]any{"enabled": true, "size": "8Gi"},
		},
		Outputs: model.JSONMap{
			"username": map[string]any{"value": "araneae", "secret": true},
			"password": map[string]any{"value": "secret-password", "secret": true},
			"url":      map[string]any{"value": "amqp://araneae:secret-password@rabbitmq:5672/", "secret": true},
		},
	}

	files := service.RenderDependencyManifests(model.Application{Name: "araneae"}, dep, def)
	values := files[path.Join("apps", "araneae", "dependencies", "rabbitmq", "values.yaml")]
	assertContains(t, values, "allowInsecureImages: true")
	assertContains(t, values, "repository: bitnamilegacy/rabbitmq")
	assertContains(t, values, "existingPasswordSecret: araneae-rabbitmq-credentials")
	assertContains(t, values, "size: \"8Gi\"")
	if strings.Contains(values, "secret-password") {
		t.Fatalf("dependency values must not include secret plaintext:\n%s", values)
	}

	app := files[path.Join("apps", "araneae", "dependencies", "rabbitmq", "application.yaml")]
	assertContains(t, app, "repoURL: https://charts.bitnami.com/bitnami")
	assertContains(t, app, "targetRevision: 16.0.14")
	assertContains(t, app, "chart: rabbitmq")
	assertContains(t, app, "namespace: app-araneae")
	assertContains(t, app, "releaseName: rabbitmq")
}

func assertContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected %q in:\n%s", want, text)
	}
}
