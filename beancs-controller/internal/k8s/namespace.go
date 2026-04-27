package k8s

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespaceSummary struct {
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Labels     map[string]string `json:"labels,omitempty"`
	AgeSeconds int64             `json:"age_seconds"`
	CreatedAt  time.Time         `json:"created_at"`
}

func (m *Manager) CreateNamespace(ctx context.Context, name, projectName string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	_, err := m.Clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: Labels(projectName)},
	}, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (m *Manager) DeleteNamespace(ctx context.Context, name string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	err := m.Clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (m *Manager) ListNamespaces(ctx context.Context) ([]NamespaceSummary, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	list, err := m.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]NamespaceSummary, 0, len(list.Items))
	now := time.Now()
	for _, ns := range list.Items {
		created := ns.CreationTimestamp.Time
		out = append(out, NamespaceSummary{
			Name:       ns.Name,
			Status:     string(ns.Status.Phase),
			Labels:     ns.Labels,
			AgeSeconds: int64(now.Sub(created).Seconds()),
			CreatedAt:  created,
		})
	}
	return out, nil
}
