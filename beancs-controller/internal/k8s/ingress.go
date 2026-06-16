package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/zeturn/beancs-controller/internal/model"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Manager) ApplyIngress(ctx context.Context, namespace, projectName, host, exposureMode string, servicePort int32) error {
	if host == "" {
		return nil
	}
	return m.ApplyIngressPorts(ctx, namespace, projectName, model.ProjectPorts{{
		Name:     "http",
		Port:     int(servicePort),
		Exposure: exposureMode,
		Domain:   host,
	}})
}

func (m *Manager) ApplyIngressPorts(ctx context.Context, namespace, projectName string, ports model.ProjectPorts) error {
	if err := m.ensure(); err != nil {
		return err
	}
	for _, port := range ports {
		if port.Exposure == model.ExposureInternalOnly || port.Domain == "" {
			continue
		}
		if err := m.applyIngressPort(ctx, namespace, projectName, port); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) applyIngressPort(ctx context.Context, namespace, projectName string, port model.ProjectPort) error {
	className := "traefik"
	annotations := map[string]string{}
	if port.Exposure == model.ExposurePublic {
		annotations["cert-manager.io/cluster-issuer"] = "letsencrypt-prod"
		annotations["kubernetes.io/ingress.class"] = "traefik"
		annotations["traefik.ingress.kubernetes.io/router.entrypoints"] = "websecure"
		annotations["traefik.ingress.kubernetes.io/router.tls"] = "true"
	}
	if port.Exposure == model.ExposurePrivate {
		className = "tailscale"
		annotations = map[string]string{}
	}
	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", projectName, port.Name),
			Namespace:   namespace,
			Labels:      Labels(projectName),
			Annotations: annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &className,
			Rules: []networkingv1.IngressRule{{
				Host: ingressRuleHost(className, port.Domain),
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
					Path:     "/",
					PathType: &pathType,
					Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
						Name: projectName,
						Port: networkingv1.ServiceBackendPort{Number: int32(port.Port)},
					}},
				}}}},
			}},
		},
	}
	if port.Exposure == model.ExposurePrivate {
		ing.Spec.TLS = []networkingv1.IngressTLS{{Hosts: []string{port.Domain}}}
	} else if port.Exposure == model.ExposurePublic {
		ing.Spec.TLS = []networkingv1.IngressTLS{{Hosts: []string{port.Domain}, SecretName: fmt.Sprintf("%s-%s-tls", projectName, port.Name)}}
	}
	_, err := m.Clientset.NetworkingV1().Ingresses(namespace).Create(ctx, ing, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.NetworkingV1().Ingresses(namespace).Get(ctx, ing.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.Spec = ing.Spec
		current.Labels = ing.Labels
		current.Annotations = ing.Annotations
		_, err = m.Clientset.NetworkingV1().Ingresses(namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}

func ingressRuleHost(className, host string) string {
	if isTailscaleIngress(className) {
		return ""
	}
	return strings.TrimSpace(host)
}

func isTailscaleIngress(className string) bool {
	return strings.EqualFold(strings.TrimSpace(className), "tailscale")
}
