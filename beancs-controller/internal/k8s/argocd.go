package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ArgoCDApplicationStatus struct {
	SyncStatus string
	Health     string
	Revision   string
	Phase      string
	Message    string
}

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
	resource := m.Dynamic.Resource(gvr).Namespace("argocd")
	current, err := resource.Get(ctx, projectName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = resource.Create(ctx, app, metav1.CreateOptions{})
		if apierrors.IsNotFound(err) && strings.Contains(strings.ToLower(err.Error()), "requested resource") {
			return fmt.Errorf("Argo CD Application CRD is not installed or not reachable in this cluster; install Argo CD first or choose passive update mode")
		}
		return err
	}
	if err != nil {
		return err
	}
	app.SetResourceVersion(current.GetResourceVersion())
	_, err = resource.Update(ctx, app, metav1.UpdateOptions{})
	return err
}

func (m *Manager) GetArgoCDApplicationStatus(ctx context.Context, namespace, name string) (ArgoCDApplicationStatus, error) {
	if err := m.ensure(); err != nil {
		return ArgoCDApplicationStatus{}, err
	}
	if strings.TrimSpace(namespace) == "" {
		namespace = "argocd"
	}
	gvr := schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}
	resource := m.Dynamic.Resource(gvr).Namespace(namespace)
	current, err := resource.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return ArgoCDApplicationStatus{}, err
	}
	syncStatus, _, _ := unstructured.NestedString(current.Object, "status", "sync", "status")
	revision, _, _ := unstructured.NestedString(current.Object, "status", "sync", "revision")
	health, _, _ := unstructured.NestedString(current.Object, "status", "health", "status")
	phase, _, _ := unstructured.NestedString(current.Object, "status", "operationState", "phase")
	message, _, _ := unstructured.NestedString(current.Object, "status", "operationState", "message")
	return ArgoCDApplicationStatus{
		SyncStatus: strings.TrimSpace(syncStatus),
		Health:     strings.TrimSpace(health),
		Revision:   strings.TrimSpace(revision),
		Phase:      strings.TrimSpace(phase),
		Message:    strings.TrimSpace(message),
	}, nil
}

func (m *Manager) WaitForArgoCDApplication(ctx context.Context, name string, expectedRevision string, timeout time.Duration) error {
	return m.waitForArgoCDApplication(ctx, name, expectedRevision, timeout, true)
}

func (m *Manager) WaitForArgoCDApplicationSync(ctx context.Context, name string, expectedRevision string, timeout time.Duration) error {
	return m.waitForArgoCDApplication(ctx, name, expectedRevision, timeout, false)
}

func (m *Manager) waitForArgoCDApplication(ctx context.Context, name string, expectedRevision string, timeout time.Duration, requireHealthy bool) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("argocd application name is required")
	}
	if strings.TrimSpace(expectedRevision) == "" {
		return fmt.Errorf("expected argocd revision is required")
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	deadline := time.Now().Add(timeout)
	for {
		status, err := m.GetArgoCDApplicationStatus(ctx, "argocd", name)
		if err != nil {
			return err
		}
		if strings.EqualFold(status.Phase, "Failed") || strings.EqualFold(status.Phase, "Error") {
			return fmt.Errorf("argocd application %s failed: phase=%s message=%s", name, status.Phase, status.Message)
		}
		if strings.EqualFold(status.SyncStatus, "Synced") &&
			(!requireHealthy || strings.EqualFold(status.Health, "Healthy")) &&
			revisionMatches(status.Revision, expectedRevision) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("argocd application %s sync timeout: expected_revision=%s actual_revision=%s sync_status=%s health=%s phase=%s message=%s", name, expectedRevision, status.Revision, status.SyncStatus, status.Health, status.Phase, status.Message)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

func (m *Manager) ApplyArgoCDRepository(ctx context.Context, name, repoURL string, data map[string]string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if strings.TrimSpace(repoURL) == "" {
		return fmt.Errorf("argocd repository url is required")
	}
	secretName := "beancs-repo-" + strings.Trim(strings.ToLower(name), "-")
	if len(secretName) > 63 {
		secretName = secretName[:63]
	}
	stringData := map[string]string{
		"type": "git",
		"url":  repoURL,
	}
	for k, v := range data {
		if strings.TrimSpace(v) != "" {
			stringData[k] = v
		}
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "argocd",
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type": "repository",
				"managed-by":                     "beancs",
			},
		},
		StringData: stringData,
		Type:       corev1.SecretTypeOpaque,
	}
	current, err := m.Clientset.CoreV1().Secrets("argocd").Get(ctx, secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = m.Clientset.CoreV1().Secrets("argocd").Create(ctx, secret, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	secret.ResourceVersion = current.ResourceVersion
	_, err = m.Clientset.CoreV1().Secrets("argocd").Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

// DeleteArgoCDApplication removes the Argo CD Application CR for a project.
func (m *Manager) DeleteArgoCDApplication(ctx context.Context, projectName string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	gvr := schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}
	resource := m.Dynamic.Resource(gvr).Namespace("argocd")
	err := resource.Delete(ctx, projectName, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil // already gone
	}
	return err
}

func revisionMatches(actual, expected string) bool {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)
	return actual == expected ||
		strings.HasPrefix(actual, expected) ||
		strings.HasPrefix(expected, actual)
}
