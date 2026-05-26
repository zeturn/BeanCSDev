package spec

import "testing"

const araneaeSpecYAML = `
apiVersion: beancs.io/v1alpha1
kind: Application
metadata:
  name: araneae
  displayName: Araneae
spec:
  type: monorepo
  repo:
    provider: github
    name: zeturn/AraneaeDev
    branch: main
  namespace:
    strategy: shared
    name: app-araneae
  autoDeploy:
    enabled: true
    mode: affected-components
  dependencies:
    - name: rabbitmq
      type: rabbitmq
      deployMethod: helm
      config:
        username: araneae
  components:
    - name: control
      kind: service
      projectName: araneae-control
      build:
        context: Backend
        dockerfile: Backend/Dockerfile
        args:
          TARGET: control
      ports:
        - name: http
          port: 8180
          protocol: http
          exposure: public
      healthCheck:
        type: http
        path: /healthz
        port: http
      envFromDependencies:
        - dependency: rabbitmq
          preset: rabbitmq_default
      volumes:
        - name: data
          type: pvc
          mountPath: /data
          size: 10Gi
      watchPaths:
        - Backend/**
    - name: executor
      kind: worker
      projectName: araneae-executor
      build:
        context: Backend
        dockerfile: Backend/Dockerfile
        args:
          TARGET: executor
      ports:
        - name: http
          port: 4280
          protocol: http
          exposure: private
      healthCheck:
        type: http
        path: /healthz
        port: http
      envFromDependencies:
        - dependency: rabbitmq
          preset: rabbitmq_default
      watchPaths:
        - Backend/**
    - name: front
      kind: frontend
      projectName: araneae-front
      build:
        context: Frontend
        dockerfile: Frontend/Dockerfile
        args:
          VITE_API_FLAVOR: go
      ports:
        - name: http
          port: 80
          protocol: http
          exposure: public
      healthCheck:
        type: tcp
        port: http
      watchPaths:
        - Frontend/**
`

func TestParseApplicationSpec(t *testing.T) {
	doc, err := Parse([]byte(araneaeSpecYAML))
	if err != nil {
		t.Fatal(err)
	}
	if doc.APIVersion != APIVersionV1Alpha1 {
		t.Fatalf("apiVersion = %q", doc.APIVersion)
	}
	if doc.Metadata.Name != "araneae" {
		t.Fatalf("metadata.name = %q", doc.Metadata.Name)
	}
	if len(doc.Spec.Components) != 3 {
		t.Fatalf("components = %d", len(doc.Spec.Components))
	}
}

func TestParseApplicationSpecInvalidKind(t *testing.T) {
	doc, err := Parse([]byte(`apiVersion: beancs.io/v1alpha1
kind: Widget
metadata:
  name: bad
spec:
  type: monorepo
  repo:
    name: owner/repo
  components: []
`))
	if err != nil {
		t.Fatal(err)
	}
	result := Validate(doc, ValidateOptions{})
	if result.Valid {
		t.Fatal("expected invalid result")
	}
	if result.Errors[0].Field != "kind" {
		t.Fatalf("first error field = %q", result.Errors[0].Field)
	}
}

func TestValidateAllowsForwardComponentReferences(t *testing.T) {
	doc, err := Parse([]byte(`apiVersion: beancs.io/v1alpha1
kind: Application
metadata:
  name: ordered
spec:
  type: monorepo
  repo:
    name: owner/repo
  components:
    - name: backend
      kind: service
      projectName: ordered-backend
      dependsOn:
        - datafs
      build:
        context: backend
        dockerfile: backend/Dockerfile
    - name: datafs
      kind: service
      projectName: ordered-datafs
      build:
        context: datafs
        dockerfile: datafs/Dockerfile
`))
	if err != nil {
		t.Fatal(err)
	}
	result := Validate(doc, ValidateOptions{RepoFiles: map[string]bool{
		"backend/Dockerfile": true,
		"datafs/Dockerfile":  true,
	}})
	for _, issue := range result.Errors {
		if issue.Field == "spec.components[0].dependsOn[0]" {
			t.Fatalf("unexpected forward reference error: %#v", issue)
		}
	}
	if !result.Valid {
		t.Fatalf("expected valid result, got %#v", result.Errors)
	}
}
