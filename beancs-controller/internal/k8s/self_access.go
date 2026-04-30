package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ControllerAccessOptions struct {
	Namespace     string
	Name          string
	ServicePort   int
	PublicHost    string
	TailscaleHost string
	WebhookHost   string
}

func (m *Manager) ApplyControllerAccess(ctx context.Context, opts ControllerAccessOptions) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if opts.Namespace == "" {
		return fmt.Errorf("controller namespace is required")
	}
	if opts.Name == "" {
		return fmt.Errorf("controller name is required")
	}
	if opts.ServicePort == 0 {
		opts.ServicePort = 8080
	}
	if err := m.applyControllerService(ctx, opts); err != nil {
		return err
	}
	if opts.TailscaleHost != "" {
		if err := m.applyControllerIngress(ctx, controllerIngressOptions{
			Namespace: opts.Namespace,
			Name:      opts.Name + "-tailscale",
			Service:   opts.Name,
			Host:      opts.TailscaleHost,
			ClassName: "tailscale",
			Path:      "/",
			Port:      opts.ServicePort,
		}); err != nil {
			return err
		}
	}
	if opts.PublicHost != "" {
		if err := m.applyControllerIngress(ctx, controllerIngressOptions{
			Namespace:     opts.Namespace,
			Name:          opts.Name + "-public",
			Service:       opts.Name,
			Host:          opts.PublicHost,
			ClassName:     "traefik",
			Path:          "/",
			Port:          opts.ServicePort,
			TLSSecretName: opts.Name + "-public-tls",
			Annotations:   map[string]string{"cert-manager.io/cluster-issuer": "letsencrypt-prod"},
		}); err != nil {
			return err
		}
	}
	if opts.WebhookHost != "" {
		webhookTLSSecret := opts.Name + "-webhooks-public-tls"
		if opts.WebhookHost == opts.PublicHost {
			webhookTLSSecret = opts.Name + "-public-tls"
		}
		if err := m.applyControllerIngress(ctx, controllerIngressOptions{
			Namespace:     opts.Namespace,
			Name:          opts.Name + "-webhooks-public",
			Service:       opts.Name,
			Host:          opts.WebhookHost,
			ClassName:     "traefik",
			Path:          "/api/v1/webhooks",
			Port:          opts.ServicePort,
			TLSSecretName: webhookTLSSecret,
			Annotations:   map[string]string{"cert-manager.io/cluster-issuer": "letsencrypt-prod"},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) applyControllerService(ctx context.Context, opts ControllerAccessOptions) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: opts.Name, Namespace: opts.Namespace, Labels: Labels(opts.Name)},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": opts.Name},
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       int32(opts.ServicePort),
				TargetPort: intstr.FromString("http"),
			}},
		},
	}
	_, err := m.Clientset.CoreV1().Services(opts.Namespace).Create(ctx, svc, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.CoreV1().Services(opts.Namespace).Get(ctx, opts.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.Labels = svc.Labels
		current.Spec.Selector = svc.Spec.Selector
		current.Spec.Ports = svc.Spec.Ports
		_, err = m.Clientset.CoreV1().Services(opts.Namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}

type controllerIngressOptions struct {
	Namespace     string
	Name          string
	Service       string
	Host          string
	ClassName     string
	Path          string
	Port          int
	TLSSecretName string
	Annotations   map[string]string
}

func (m *Manager) applyControllerIngress(ctx context.Context, opts controllerIngressOptions) error {
	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        opts.Name,
			Namespace:   opts.Namespace,
			Labels:      Labels(opts.Service),
			Annotations: opts.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &opts.ClassName,
			Rules: []networkingv1.IngressRule{{
				Host: opts.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
					Path:     opts.Path,
					PathType: &pathType,
					Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
						Name: opts.Service,
						Port: networkingv1.ServiceBackendPort{Number: int32(opts.Port)},
					}},
				}}}},
			}},
		},
	}
	if opts.TLSSecretName != "" {
		ing.Spec.TLS = []networkingv1.IngressTLS{{Hosts: []string{opts.Host}, SecretName: opts.TLSSecretName}}
	}
	_, err := m.Clientset.NetworkingV1().Ingresses(opts.Namespace).Create(ctx, ing, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.NetworkingV1().Ingresses(opts.Namespace).Get(ctx, opts.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.Labels = ing.Labels
		current.Annotations = ing.Annotations
		current.Spec = ing.Spec
		_, err = m.Clientset.NetworkingV1().Ingresses(opts.Namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}
