package k8s

import (
	"context"
	"strings"

	"github.com/zeturn/beancs-controller/internal/dto"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (m *Manager) CreateNamespaceWithLabels(ctx context.Context, name string, labels map[string]string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if labels == nil {
		labels = map[string]string{}
	}
	_, err := m.Clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
	}, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (m *Manager) PatchNamespaceLabels(ctx context.Context, name string, labels map[string]string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	ns, err := m.Clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	for key, value := range labels {
		if strings.TrimSpace(value) == "" {
			delete(ns.Labels, key)
		} else {
			ns.Labels[key] = value
		}
	}
	_, err = m.Clientset.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	return err
}

func (m *Manager) DeletePod(ctx context.Context, namespace, name string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	err := m.Clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (m *Manager) PodLogs(ctx context.Context, namespace, name string, tail int64) (string, error) {
	if err := m.ensure(); err != nil {
		return "", err
	}
	pod, err := m.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return m.logsForPods(ctx, []corev1.Pod{*pod}, tail)
}

func (m *Manager) UpsertService(ctx context.Context, req dto.CreateServiceRequest) error {
	if err := m.ensure(); err != nil {
		return err
	}
	serviceType := corev1.ServiceType(req.Type)
	if serviceType == "" {
		serviceType = corev1.ServiceTypeClusterIP
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace, Labels: req.Labels},
		Spec: corev1.ServiceSpec{
			Type:     serviceType,
			Selector: req.Selector,
			Ports:    runtimeServicePorts(req.Ports),
		},
	}
	current, err := m.Clientset.CoreV1().Services(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = m.Clientset.CoreV1().Services(req.Namespace).Create(ctx, svc, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	current.Labels = req.Labels
	current.Spec.Type = serviceType
	current.Spec.Selector = req.Selector
	current.Spec.Ports = svc.Spec.Ports
	_, err = m.Clientset.CoreV1().Services(req.Namespace).Update(ctx, current, metav1.UpdateOptions{})
	return err
}

func (m *Manager) DeleteService(ctx context.Context, namespace, name string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	err := m.Clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func runtimeServicePorts(ports []dto.ServicePortSpec) []corev1.ServicePort {
	out := make([]corev1.ServicePort, 0, len(ports))
	for _, p := range ports {
		target := p.TargetPort
		if target == 0 {
			target = p.Port
		}
		protocol := corev1.Protocol(strings.ToUpper(strings.TrimSpace(p.Protocol)))
		if protocol == "" {
			protocol = corev1.ProtocolTCP
		}
		out = append(out, corev1.ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: intstr.FromInt32(target),
			Protocol:   protocol,
		})
	}
	return out
}
