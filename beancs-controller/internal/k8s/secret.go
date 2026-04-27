package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Manager) UpsertSecret(ctx context.Context, namespace, name, projectName string, data map[string]string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	stringData := map[string]string{}
	for k, v := range data {
		stringData[k] = v
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: Labels(projectName)},
		Type:       corev1.SecretTypeOpaque,
		StringData: stringData,
	}
	_, err := m.Clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.StringData = stringData
		current.Labels = Labels(projectName)
		_, err = m.Clientset.CoreV1().Secrets(namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}

func (m *Manager) SecretData(ctx context.Context, namespace, name string) (map[string]string, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	secret, err := m.Clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for k, v := range secret.Data {
		if len(v) == 0 {
			out[k] = ""
		} else {
			out[k] = "********"
		}
	}
	return out, nil
}

func (m *Manager) SecretPlainData(ctx context.Context, namespace, name string) (map[string]string, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	secret, err := m.Clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for k, v := range secret.Data {
		out[k] = string(v)
	}
	return out, nil
}
