package k8s

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type BasaltPassRuntime struct {
	Name          string
	Namespace     string
	BackendImage  string
	FrontendImage string
	PullSecret    string
	Host          string
	Exposure      string
	Env           map[string]string
}

func (m *Manager) ApplyBasaltPass(ctx context.Context, in BasaltPassRuntime) error {
	if err := m.ensure(); err != nil {
		return err
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Namespace = strings.TrimSpace(in.Namespace)
	in.BackendImage = strings.TrimSpace(in.BackendImage)
	in.FrontendImage = strings.TrimSpace(in.FrontendImage)
	if in.Name == "" || in.Namespace == "" {
		return fmt.Errorf("BasaltPass name and namespace are required")
	}
	if in.BackendImage == "" || in.FrontendImage == "" {
		return fmt.Errorf("BasaltPass backend_image and frontend_image are required")
	}
	if err := m.CreateNamespace(ctx, in.Namespace, in.Name); err != nil {
		return err
	}
	secretName := in.Name + "-env"
	if err := m.UpsertSecret(ctx, in.Namespace, secretName, in.Name, in.Env); err != nil {
		return err
	}
	backendName := in.Name + "-backend"
	frontendName := in.Name + "-frontend"
	if err := m.applyBasaltPassDeployment(ctx, in.Namespace, backendName, in.BackendImage, 8101, secretName, in.PullSecret, true); err != nil {
		return err
	}
	if err := m.applyBasaltPassService(ctx, in.Namespace, "backend", backendName, 8101); err != nil {
		return err
	}
	if err := m.applyBasaltPassDeployment(ctx, in.Namespace, frontendName, in.FrontendImage, 80, secretName, in.PullSecret, false); err != nil {
		return err
	}
	if err := m.applyBasaltPassService(ctx, in.Namespace, frontendName, frontendName, 80); err != nil {
		return err
	}
	if strings.TrimSpace(in.Host) != "" {
		if err := m.applyBasaltPassIngress(ctx, in.Namespace, in.Name, frontendName, strings.TrimSpace(in.Host), strings.TrimSpace(in.Exposure)); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) applyBasaltPassDeployment(ctx context.Context, namespace, name, image string, port int32, secretName, pullSecret string, envFromSecret bool) error {
	labels := Labels(name)
	container := corev1.Container{
		Name:  "app",
		Image: image,
		Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: port}},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("128Mi")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("512Mi")},
		},
	}
	if envFromSecret {
		container.EnvFrom = []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: secretName}}}}
	}
	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{container},
	}
	if strings.TrimSpace(pullSecret) != "" {
		podSpec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: strings.TrimSpace(pullSecret)}}
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       podSpec,
			},
		},
	}
	_, err := m.Clientset.AppsV1().Deployments(namespace).Create(ctx, dep, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.Labels = dep.Labels
		current.Spec = dep.Spec
		_, err = m.Clientset.AppsV1().Deployments(namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}

func (m *Manager) applyBasaltPassService(ctx context.Context, namespace, serviceName, workloadName string, port int32) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: namespace, Labels: Labels(workloadName)},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: Labels(workloadName),
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       port,
				TargetPort: intstr.FromInt32(port),
			}},
		},
	}
	_, err := m.Clientset.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.Labels = svc.Labels
		current.Spec.Ports = svc.Spec.Ports
		current.Spec.Selector = svc.Spec.Selector
		_, err = m.Clientset.CoreV1().Services(namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}

func (m *Manager) applyBasaltPassIngress(ctx context.Context, namespace, name, serviceName, host, exposure string) error {
	className := "traefik"
	annotations := map[string]string{}
	if exposure == "private" {
		className = "tailscale"
	} else {
		annotations["kubernetes.io/ingress.class"] = "traefik"
		annotations["traefik.ingress.kubernetes.io/router.entrypoints"] = "websecure"
		annotations["traefik.ingress.kubernetes.io/router.tls"] = "true"
		if m.CertManager.IssuerName != "" {
			annotations["cert-manager.io/issuer"] = m.CertManager.IssuerName
		}
	}
	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: name + "-public", Namespace: namespace, Labels: Labels(name), Annotations: annotations},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &className,
			Rules: []networkingv1.IngressRule{{
				Host: ingressRuleHost(className, host),
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
					Path:     "/",
					PathType: &pathType,
					Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
						Name: serviceName,
						Port: networkingv1.ServiceBackendPort{Number: 80},
					}},
				}}}},
			}},
		},
	}
	if exposure == "private" {
		ing.Spec.TLS = []networkingv1.IngressTLS{{Hosts: []string{host}}}
	} else {
		ing.Spec.TLS = []networkingv1.IngressTLS{{Hosts: []string{host}, SecretName: name + "-tls"}}
	}
	_, err := m.Clientset.NetworkingV1().Ingresses(namespace).Create(ctx, ing, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.NetworkingV1().Ingresses(namespace).Get(ctx, ing.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.Labels = ing.Labels
		current.Annotations = ing.Annotations
		current.Spec = ing.Spec
		_, err = m.Clientset.NetworkingV1().Ingresses(namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}
