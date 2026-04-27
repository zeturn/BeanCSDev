package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (m *Manager) ApplyArgoCDApplication(ctx context.Context, projectName, repoURL, path, namespace string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	gvr := schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}
	app := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]any{
			"name":      projectName,
			"namespace": "argocd",
			"labels":    Labels(projectName),
		},
		"spec": map[string]any{
			"project":     "default",
			"source":      map[string]any{"repoURL": repoURL, "targetRevision": "HEAD", "path": path},
			"destination": map[string]any{"server": "https://kubernetes.default.svc", "namespace": namespace},
			"syncPolicy":  map[string]any{"automated": map[string]any{"prune": true, "selfHeal": true}},
		},
	}}
	_, err := m.Dynamic.Resource(gvr).Namespace("argocd").Create(ctx, app, metav1.CreateOptions{})
	return err
}
