package k8s

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestRevisionMatches(t *testing.T) {
	if !revisionMatches("abcdef123456", "abcdef1") {
		t.Fatalf("expected short revision prefix to match")
	}
	if revisionMatches("abcdef123456", "fffffff") {
		t.Fatalf("unexpected revision match")
	}
}

func TestWaitForArgoCDApplicationRequiresExpectedRevision(t *testing.T) {
	manager := testArgoCDManager(t, "Synced", "Healthy", "oldsha123", "Succeeded", "")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := manager.WaitForArgoCDApplication(ctx, "demo", "newsha999", 10*time.Millisecond)
	if err == nil {
		t.Fatalf("expected wait to fail when revision does not match")
	}
}

func TestWaitForArgoCDApplicationSucceedsOnExpectedRevision(t *testing.T) {
	manager := testArgoCDManager(t, "Synced", "Healthy", "abcdef123456", "Succeeded", "")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.WaitForArgoCDApplication(ctx, "demo", "abcdef12", 100*time.Millisecond); err != nil {
		t.Fatalf("expected wait to succeed, got %v", err)
	}
}

func TestWaitForArgoCDApplicationFailsOnOperationPhase(t *testing.T) {
	manager := testArgoCDManager(t, "OutOfSync", "Progressing", "abcdef123456", "Failed", "sync failed")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := manager.WaitForArgoCDApplication(ctx, "demo", "abcdef12", time.Second)
	if err == nil {
		t.Fatalf("expected failed operation phase to return error")
	}
}

func testArgoCDManager(t *testing.T, syncStatus, health, revision, phase, message string) *Manager {
	t.Helper()
	app := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Application",
			"metadata": map[string]any{
				"name":      "demo",
				"namespace": "argocd",
			},
			"status": map[string]any{
				"sync": map[string]any{
					"status":   syncStatus,
					"revision": revision,
				},
				"health": map[string]any{
					"status": health,
				},
				"operationState": map[string]any{
					"phase":   phase,
					"message": message,
				},
			},
		},
	}
	app.SetGroupVersionKind(applicationGVK())
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), app)
	return &Manager{
		Clientset: k8sfake.NewSimpleClientset(),
		Dynamic:   dyn,
	}
}

func applicationGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	}
}
