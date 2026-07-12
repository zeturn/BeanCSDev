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
				{"name": "data", "type": "pvc", "mountPath": "/data", "size": "10Gi", "storageClassName": "longhorn"},
			},
		},
	}

	files := service.RenderManifests(project)
	deployment := files[path.Join("apps", project.Name, "base", "deployment.yaml")]
	assertContains(t, deployment, "path: /healthz")
	assertContains(t, deployment, "port: http")
	assertContains(t, deployment, "name: app-env-vars-araneae-control")
	assertContains(t, deployment, "volumeMounts:")
	assertContains(t, deployment, "mountPath: /data")
	assertContains(t, deployment, "claimName: araneae-control-data")

	kustomization := files[path.Join("apps", project.Name, "base", "kustomization.yaml")]
	assertContains(t, kustomization, "- pvc-data.yaml")

	pvc := files[path.Join("apps", project.Name, "base", "pvc-data.yaml")]
	assertContains(t, pvc, "kind: PersistentVolumeClaim")
	assertContains(t, pvc, "storage: 10Gi")
	assertContains(t, pvc, `storageClassName: "longhorn"`)
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

func TestUpdateKustomizeImageEntryUpdatesOnlyMatchingImage(t *testing.T) {
	content := `resources:
  - ../../base
images:
  - name: registry.beancs.hollowdata.com/hollowdata/apicred-backend
    newName: registry.beancs.hollowdata.com/hollowdata/apicred-backend
    newTag: beancs-73fe1fa
  - name: registry.beancs.hollowdata.com/hollowdata/apicred-frontend
    newName: registry.beancs.hollowdata.com/hollowdata/apicred-frontend
    newTag: beancs-73fe1fa
`
	project := &model.Project{
		Name:                   "apicred-backend",
		RegistryImageReference: "registry.beancs.hollowdata.com/hollowdata/apicred-backend",
	}

	updated, matched, changed := updateKustomizeImageEntry(content, project, "registry.beancs.hollowdata.com/hollowdata/apicred-backend:v2026.07.12-5")

	if !matched {
		t.Fatal("expected backend image entry to match")
	}
	if !changed {
		t.Fatal("expected kustomization to change")
	}
	assertContains(t, updated, "registry.beancs.hollowdata.com/hollowdata/apicred-backend\n    newName: registry.beancs.hollowdata.com/hollowdata/apicred-backend\n    newTag: v2026.07.12-5")
	assertContains(t, updated, "registry.beancs.hollowdata.com/hollowdata/apicred-frontend\n    newName: registry.beancs.hollowdata.com/hollowdata/apicred-frontend\n    newTag: beancs-73fe1fa")
}

func TestUpdateKustomizeImageEntryReportsMatchWithoutChange(t *testing.T) {
	content := `images:
  - name: registry.beancs.hollowdata.com/hollowdata/apicred-frontend
    newName: registry.beancs.hollowdata.com/hollowdata/apicred-frontend
    newTag: v2026.07.12-5
`
	project := &model.Project{
		Name:                   "apicred-frontend",
		RegistryImageReference: "registry.beancs.hollowdata.com/hollowdata/apicred-frontend",
	}

	updated, matched, changed := updateKustomizeImageEntry(content, project, "registry.beancs.hollowdata.com/hollowdata/apicred-frontend:v2026.07.12-5")

	if !matched {
		t.Fatal("expected frontend image entry to match")
	}
	if changed {
		t.Fatal("expected matching image entry to require no change")
	}
	if updated != content {
		t.Fatal("expected content to stay unchanged")
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
			"persistence": map[string]any{"enabled": true, "size": "8Gi", "storageClass": "longhorn"},
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
	assertContains(t, values, `storageClass: "longhorn"`)
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

func TestRenderMySQLDependencyManifestsUseLegacyImage(t *testing.T) {
	service := NewGitOpsService(nil, nil)
	registry, err := NewDependencyDefinitionRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, ok := registry.Get("mysql")
	if !ok {
		t.Fatal("mysql definition not found")
	}
	dep := model.ManagedDependency{
		Name:         "mysql",
		Type:         "mysql",
		Namespace:    "app-araneae",
		ServiceName:  "mysql",
		SecretName:   "araneae-mysql-credentials",
		DeployMethod: model.DependencyDeployMethodHelm,
		Config: model.JSONMap{
			"database":    "app",
			"username":    "app",
			"persistence": map[string]any{"enabled": false, "size": "1Gi"},
		},
		Outputs: model.JSONMap{
			"username": map[string]any{"value": "app", "secret": true},
			"database": map[string]any{"value": "app"},
			"password": map[string]any{"value": "secret-password", "secret": true},
		},
	}

	files := service.RenderDependencyManifests(model.Application{Name: "araneae"}, dep, def)
	values := files[path.Join("apps", "araneae", "dependencies", "mysql", "values.yaml")]
	assertContains(t, values, "allowInsecureImages: true")
	assertContains(t, values, "repository: bitnamilegacy/mysql")
	assertContains(t, values, "existingSecret: araneae-mysql-credentials")
	assertContains(t, values, "enabled: false")
	if strings.Contains(values, "secret-password") {
		t.Fatalf("dependency values must not include secret plaintext:\n%s", values)
	}

	app := files[path.Join("apps", "araneae", "dependencies", "mysql", "application.yaml")]
	assertContains(t, app, "targetRevision: 14.0.3")
}

func TestRenderPostgreSQLDependencyManifestsUsePostgreSQLSecretKeys(t *testing.T) {
	service := NewGitOpsService(nil, nil)
	registry, err := NewDependencyDefinitionRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, ok := registry.Get("postgresql")
	if !ok {
		t.Fatal("postgresql definition not found")
	}
	dep := model.ManagedDependency{
		Name:         "pgsql",
		Type:         "postgresql",
		Namespace:    "app-araneae",
		ServiceName:  "pgsql",
		SecretName:   "araneae-pgsql-credentials",
		DeployMethod: model.DependencyDeployMethodHelm,
		Config: model.JSONMap{
			"database":    "app",
			"username":    "app",
			"persistence": map[string]any{"enabled": false, "size": "1Gi"},
		},
		Outputs: model.JSONMap{
			"username": map[string]any{"value": "app", "secret": true},
			"database": map[string]any{"value": "app"},
			"password": map[string]any{"value": "secret-password", "secret": true},
		},
	}

	files := service.RenderDependencyManifests(model.Application{Name: "araneae"}, dep, def)
	values := files[path.Join("apps", "araneae", "dependencies", "pgsql", "values.yaml")]
	assertContains(t, values, "allowInsecureImages: true")
	assertContains(t, values, "repository: bitnamilegacy/postgresql")
	assertContains(t, values, "existingSecret: araneae-pgsql-credentials")
	assertContains(t, values, "user: password")
	assertContains(t, values, "enabled: false")
	if strings.Contains(values, "bitnamilegacy/mysql") {
		t.Fatalf("postgresql dependency values must not include mysql settings:\n%s", values)
	}
	if strings.Contains(values, "secret-password") {
		t.Fatalf("dependency values must not include secret plaintext:\n%s", values)
	}

	app := files[path.Join("apps", "araneae", "dependencies", "pgsql", "application.yaml")]
	assertContains(t, app, "chart: postgresql")
	assertContains(t, app, "targetRevision: 18.6.7")
}

func TestRenderTimescaleDBDependencyManifests(t *testing.T) {
	service := NewGitOpsService(nil, nil)
	registry, err := NewDependencyDefinitionRegistry()
	if err != nil {
		t.Fatal(err)
	}
	def, ok := registry.Get("timescale")
	if !ok {
		t.Fatal("timescaledb definition not found")
	}
	dep := model.ManagedDependency{
		Name:         "timescale",
		Type:         "timescaledb",
		Namespace:    "app-araneae",
		ServiceName:  "timescale",
		SecretName:   "araneae-timescale-credentials",
		DeployMethod: model.DependencyDeployMethodHelm,
		Config: model.JSONMap{
			"database":      "metrics",
			"username":      "admin",
			"replica_count": "1",
			"persistence":   map[string]any{"enabled": true, "size": "20Gi", "storageClass": "longhorn"},
		},
		Outputs: model.JSONMap{
			"username": map[string]any{"value": "admin", "secret": true},
			"database": map[string]any{"value": "metrics"},
			"password": map[string]any{"value": "secret-password", "secret": true},
		},
	}

	files := service.RenderDependencyManifests(model.Application{Name: "araneae"}, dep, def)
	values := files[path.Join("apps", "araneae", "dependencies", "timescale", "values.yaml")]
	assertContains(t, values, "replicaCount: 1")
	assertContains(t, values, `credentialsSecretName: "araneae-timescale-credentials"`)
	assertContains(t, values, `storageClass: "longhorn"`)
	if strings.Contains(values, `replicaCount: "1"`) {
		t.Fatalf("timescaledb replicaCount must be rendered as an integer:\n%s", values)
	}
	if strings.Contains(values, "secret-password") {
		t.Fatalf("dependency values must not include secret plaintext:\n%s", values)
	}
	secretData := dependencySecretRuntimeData(dep)
	if secretData["PATRONI_SUPERUSER_PASSWORD"] != "secret-password" {
		t.Fatalf("expected TimescaleDB superuser password in runtime secret data")
	}
	if secretData["PATRONI_REPLICATION_PASSWORD"] != "secret-password" {
		t.Fatalf("expected TimescaleDB replication password in runtime secret data")
	}
	if secretData["PATRONI_admin_PASSWORD"] != "secret-password" {
		t.Fatalf("expected TimescaleDB admin password in runtime secret data")
	}

	app := files[path.Join("apps", "araneae", "dependencies", "timescale", "application.yaml")]
	assertContains(t, app, "repoURL: https://charts.timescale.com")
	assertContains(t, app, "chart: timescaledb-single")
	assertContains(t, app, "targetRevision: 0.33.1")
}

func assertContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("expected %q in:\n%s", want, text)
	}
}
