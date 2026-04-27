package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/zeturn/beancs-controller/internal/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (m *Manager) ApplyDeployment(ctx context.Context, namespace, projectName, image string, port int32, replicas int32, cpuReq, cpuLimit, memReq, memLimit string) error {
	return m.ApplyDeploymentPorts(ctx, namespace, projectName, image, model.ProjectPorts{{Name: "http", Port: int(port), Exposure: model.ExposurePrivate}}, replicas, cpuReq, cpuLimit, memReq, memLimit)
}

func (m *Manager) ApplyDeploymentPorts(ctx context.Context, namespace, projectName, image string, ports model.ProjectPorts, replicas int32, cpuReq, cpuLimit, memReq, memLimit string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	containerPorts := make([]corev1.ContainerPort, 0, len(ports))
	for _, p := range ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{Name: p.Name, ContainerPort: int32(p.Port)})
	}
	probePort := int32(ports[0].Port)
	labels := Labels(projectName)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: projectName, Namespace: namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Tolerations: []corev1.Toleration{
						{Key: "node.kubernetes.io/not-ready", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute, TolerationSeconds: int64Ptr(30)},
						{Key: "node.kubernetes.io/unreachable", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute, TolerationSeconds: int64Ptr(30)},
					},
					Affinity: &corev1.Affinity{PodAntiAffinity: &corev1.PodAntiAffinity{PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
						Weight: 100,
						PodAffinityTerm: corev1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{MatchLabels: labels},
							TopologyKey:   "kubernetes.io/hostname",
						},
					}}}},
					Containers: []corev1.Container{{
						Name:    "app",
						Image:   image,
						Ports:   containerPorts,
						EnvFrom: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-env-vars"}}}},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpuReq), corev1.ResourceMemory: resource.MustParse(memReq)},
							Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpuLimit), corev1.ResourceMemory: resource.MustParse(memLimit)},
						},
						LivenessProbe:  httpProbe(probePort, 10, 10),
						ReadinessProbe: httpProbe(probePort, 5, 5),
					}},
				},
			},
		},
	}
	_, err := m.Clientset.AppsV1().Deployments(namespace).Create(ctx, dep, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.AppsV1().Deployments(namespace).Get(ctx, projectName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.Spec = dep.Spec
		current.Labels = dep.Labels
		_, err = m.Clientset.AppsV1().Deployments(namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}

func (m *Manager) RestartDeployment(ctx context.Context, namespace, name string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	dep, err := m.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if dep.Spec.Template.Annotations == nil {
		dep.Spec.Template.Annotations = map[string]string{}
	}
	dep.Spec.Template.Annotations["beancs/restarted-at"] = time.Now().UTC().Format(time.RFC3339)
	_, err = m.Clientset.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
	return err
}

func (m *Manager) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	if err := m.ensure(); err != nil {
		return err
	}
	dep, err := m.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	dep.Spec.Replicas = &replicas
	_, err = m.Clientset.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
	return err
}

func int64Ptr(v int64) *int64 { return &v }

func httpProbe(port int32, initialDelay, period int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler:        corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstrFromInt32(port)}},
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
	}
}

func intstrFromInt32(v int32) intstr.IntOrString {
	return intstr.FromInt32(v)
}

func imageRequired(image string) error {
	if image == "" {
		return fmt.Errorf("image is required")
	}
	return nil
}
