package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const traefikNativeLBArg = "--providers.kubernetesingress.nativeLBByDefault=true"

type TraefikReconcileResult struct {
	Namespace string
	Name      string
	Updated   bool
	Message   string
}

func (m *Manager) EnsureTraefikPodNetwork(ctx context.Context) (*TraefikReconcileResult, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	dep, err := m.Clientset.AppsV1().Deployments("traefik").Get(ctx, "traefik", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		list, listErr := m.Clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=traefik",
		})
		if listErr != nil {
			return nil, listErr
		}
		if len(list.Items) == 0 {
			return nil, fmt.Errorf("traefik deployment was not found")
		}
		dep = &list.Items[0]
	} else if err != nil {
		return nil, err
	}

	changed := false
	if dep.Spec.Template.Spec.HostNetwork {
		dep.Spec.Template.Spec.HostNetwork = false
		changed = true
	}
	if dep.Spec.Template.Spec.DNSPolicy != corev1.DNSClusterFirst {
		dep.Spec.Template.Spec.DNSPolicy = corev1.DNSClusterFirst
		changed = true
	}
	if len(dep.Spec.Template.Spec.Containers) > 0 && !hasArg(dep.Spec.Template.Spec.Containers[0].Args, traefikNativeLBArg) {
		dep.Spec.Template.Spec.Containers[0].Args = append(dep.Spec.Template.Spec.Containers[0].Args, traefikNativeLBArg)
		changed = true
	}

	result := &TraefikReconcileResult{
		Namespace: dep.Namespace,
		Name:      dep.Name,
		Updated:   changed,
		Message:   "traefik already uses pod networking",
	}
	if !changed {
		return result, nil
	}
	if _, err := m.Clientset.AppsV1().Deployments(dep.Namespace).Update(ctx, dep, metav1.UpdateOptions{}); err != nil {
		return nil, err
	}
	result.Message = "traefik deployment updated to use pod networking"
	return result, nil
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
