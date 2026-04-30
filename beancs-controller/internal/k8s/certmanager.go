package k8s

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var certManagerIssuerGVR = schema.GroupVersionResource{
	Group:    "cert-manager.io",
	Version:  "v1",
	Resource: "issuers",
}

func (m *Manager) ApplyProjectCertificateIssuer(ctx context.Context, namespace, projectName, cloudflareToken string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if m.Dynamic == nil {
		return fmt.Errorf("kubernetes dynamic client not configured")
	}
	if m.CertManager.IssuerName == "" {
		return fmt.Errorf("cert-manager issuer name is required")
	}
	if m.CertManager.ACMEServer == "" {
		return fmt.Errorf("cert-manager ACME server is required")
	}
	if m.CertManager.CloudflareSecretName == "" || m.CertManager.CloudflareSecretKey == "" {
		return fmt.Errorf("cert-manager Cloudflare secret name and key are required")
	}
	if cloudflareToken == "" {
		return fmt.Errorf("Cloudflare API token is required for cert-manager DNS-01")
	}
	if err := m.UpsertSecret(ctx, namespace, m.CertManager.CloudflareSecretName, projectName, map[string]string{
		m.CertManager.CloudflareSecretKey: cloudflareToken,
	}); err != nil {
		return err
	}
	return m.upsertCertManagerIssuer(ctx, namespace, projectName)
}

func (m *Manager) upsertCertManagerIssuer(ctx context.Context, namespace, projectName string) error {
	issuer := m.certManagerIssuer(namespace, projectName)
	_, err := m.Dynamic.Resource(certManagerIssuerGVR).Namespace(namespace).Create(ctx, issuer, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Dynamic.Resource(certManagerIssuerGVR).Namespace(namespace).Get(ctx, m.CertManager.IssuerName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.SetLabels(issuer.GetLabels())
		if setErr := unstructured.SetNestedMap(current.Object, issuer.Object["spec"].(map[string]any), "spec"); setErr != nil {
			return setErr
		}
		_, err = m.Dynamic.Resource(certManagerIssuerGVR).Namespace(namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}

func (m *Manager) certManagerIssuer(namespace, projectName string) *unstructured.Unstructured {
	privateKeySecretName := m.CertManager.IssuerName + "-" + m.CertManager.PrivateKeySecretSuffix
	acme := map[string]any{
		"server": m.CertManager.ACMEServer,
		"privateKeySecretRef": map[string]any{
			"name": privateKeySecretName,
		},
		"solvers": []any{
			map[string]any{
				"dns01": map[string]any{
					"cloudflare": map[string]any{
						"apiTokenSecretRef": map[string]any{
							"name": m.CertManager.CloudflareSecretName,
							"key":  m.CertManager.CloudflareSecretKey,
						},
					},
				},
			},
		},
	}
	if m.CertManager.Email != "" {
		acme["email"] = m.CertManager.Email
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "cert-manager.io/v1",
		"kind":       "Issuer",
		"metadata": map[string]any{
			"name":      m.CertManager.IssuerName,
			"namespace": namespace,
			"labels":    Labels(projectName),
		},
		"spec": map[string]any{
			"acme": acme,
		},
	}}
}
