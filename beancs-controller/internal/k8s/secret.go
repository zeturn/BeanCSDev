package k8s

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

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

func (m *Manager) UpsertRegistryPullSecret(ctx context.Context, namespace, projectName, secretName string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if secretName == "" {
		secretName = m.RegistryPullSecret
	}
	if secretName == "" {
		return fmt.Errorf("registry pull secret name is required")
	}
	if m.RegistryHost == "" || m.RegistryPullUsername == "" || m.RegistryPullToken == "" {
		return fmt.Errorf("registry pull credentials are not configured")
	}
	return m.UpsertRegistryPullSecretWithCredentials(ctx, namespace, projectName, secretName, m.RegistryHost, m.RegistryPullUsername, m.RegistryPullToken)
}

func (m *Manager) UpsertRegistryPullSecretWithCredentials(ctx context.Context, namespace, projectName, secretName, registryHost, username, token string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if secretName == "" {
		secretName = m.RegistryPullSecret
	}
	if secretName == "" {
		return fmt.Errorf("registry pull secret name is required")
	}
	if registryHost == "" || username == "" || token == "" {
		return fmt.Errorf("registry pull credentials are not configured")
	}
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + token))
	payload, err := json.Marshal(map[string]any{
		"auths": map[string]any{
			registryHost: map[string]string{
				"username": username,
				"password": token,
				"auth":     auth,
			},
		},
	})
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace, Labels: Labels(projectName)},
		Type:       corev1.SecretTypeDockerConfigJson,
		StringData: map[string]string{corev1.DockerConfigJsonKey: string(payload)},
	}
	_, err = m.Clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.Type = corev1.SecretTypeDockerConfigJson
		current.StringData = secret.StringData
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
