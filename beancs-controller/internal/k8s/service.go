package k8s

import (
	"context"

	"github.com/zeturn/beancs-controller/internal/model"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (m *Manager) ApplyService(ctx context.Context, namespace, projectName string, port int32) error {
	return m.ApplyServicePorts(ctx, namespace, projectName, model.ProjectPorts{{Name: "http", Port: int(port), Exposure: model.ExposurePrivate}})
}

func (m *Manager) ApplyServicePorts(ctx context.Context, namespace, projectName string, ports model.ProjectPorts) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if len(ports) == 0 {
		return nil
	}
	servicePorts := make([]corev1.ServicePort, 0, len(ports))
	for _, p := range ports {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:       p.Name,
			Port:       int32(p.Port),
			TargetPort: intstr.FromInt(p.Port),
		})
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectName,
			Namespace: namespace,
			Labels:    Labels(projectName),
			Annotations: map[string]string{
				"traefik.ingress.kubernetes.io/service.nativelb": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: Labels(projectName),
			Ports:    servicePorts,
		},
	}
	_, err := m.Clientset.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.CoreV1().Services(namespace).Get(ctx, svc.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.Spec.Ports = svc.Spec.Ports
		current.Spec.Selector = svc.Spec.Selector
		current.Labels = svc.Labels
		current.Annotations = svc.Annotations
		_, err = m.Clientset.CoreV1().Services(namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}
