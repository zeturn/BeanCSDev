package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
	return m.ApplyDeploymentPortsWithPullSecret(ctx, namespace, projectName, image, ports, replicas, cpuReq, cpuLimit, memReq, memLimit, "")
}

func (m *Manager) ApplyDeploymentPortsWithPullSecret(ctx context.Context, namespace, projectName, image string, ports model.ProjectPorts, replicas int32, cpuReq, cpuLimit, memReq, memLimit, pullSecret string) error {
	project := model.Project{
		Name:                   projectName,
		Namespace:              namespace,
		Ports:                  ports,
		Replicas:               int(replicas),
		RegistryPullSecretName: pullSecret,
	}
	return m.ApplyProjectDeployment(ctx, project, image, model.ResourceSpec{
		CPURequest: cpuReq,
		CPULimit:   cpuLimit,
		MemRequest: memReq,
		MemLimit:   memLimit,
	})
}

func (m *Manager) ApplyProjectDeployment(ctx context.Context, project model.Project, image string, resources model.ResourceSpec) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if err := imageRequired(image); err != nil {
		return err
	}
	if project.Replicas <= 0 {
		project.Replicas = 1
	}
	if resources.CPURequest == "" {
		resources = model.ResourcePresets["small"]
	}
	ports := project.Ports
	if len(ports) == 0 && project.Port > 0 && project.ExposureMode != model.ExposureInternalOnly {
		ports = model.ProjectPorts{{Name: "http", Port: project.Port, Exposure: project.ExposureMode, Domain: project.Domain}}
	}
	containerPorts := make([]corev1.ContainerPort, 0, len(ports))
	for _, p := range ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{Name: p.Name, ContainerPort: int32(p.Port)})
	}
	labels := Labels(project.Name)
	container := corev1.Container{
		Name:    "app",
		Image:   image,
		Ports:   containerPorts,
		EnvFrom: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: project.EnvSecretName()}}}},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(resources.CPURequest), corev1.ResourceMemory: resource.MustParse(resources.MemRequest)},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(resources.CPULimit), corev1.ResourceMemory: resource.MustParse(resources.MemLimit)},
		},
	}
	configureContainerProbes(&container, project.HealthCheckConfig(), ports)
	volumes, mounts, err := m.reconcileProjectVolumes(ctx, project.Namespace, project.Name, project.VolumeConfig())
	if err != nil {
		return err
	}
	if len(mounts) > 0 {
		container.VolumeMounts = mounts
	}
	podSpec := corev1.PodSpec{
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
		Containers: []corev1.Container{container},
	}
	if len(volumes) > 0 {
		podSpec.Volumes = volumes
	}
	if project.RegistryPullSecretName != "" {
		podSpec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: project.RegistryPullSecretName}}
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: project.Name, Namespace: project.Namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(int32(project.Replicas)),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       podSpec,
			},
		},
	}
	_, err = m.Clientset.AppsV1().Deployments(project.Namespace).Create(ctx, dep, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.AppsV1().Deployments(project.Namespace).Get(ctx, project.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.Spec = dep.Spec
		current.Labels = dep.Labels
		_, err = m.Clientset.AppsV1().Deployments(project.Namespace).Update(ctx, current, metav1.UpdateOptions{})
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

func (m *Manager) WaitForDeploymentRollout(ctx context.Context, namespace, name string, timeout time.Duration) error {
	return m.waitForDeploymentRollout(ctx, namespace, name, "", timeout)
}

func (m *Manager) WaitForDeploymentImageRollout(ctx context.Context, namespace, name, image string, timeout time.Duration) error {
	return m.waitForDeploymentRollout(ctx, namespace, name, image, timeout)
}

func (m *Manager) waitForDeploymentRollout(ctx context.Context, namespace, name, image string, timeout time.Duration) error {
	if err := m.ensure(); err != nil {
		return err
	}
	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(name) == "" {
		return fmt.Errorf("deployment namespace and name are required")
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	deadline := time.Now().Add(timeout)
	for {
		dep, err := m.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if dep.Spec.Replicas != nil {
			expected := *dep.Spec.Replicas
			status := dep.Status
			imageMatches := true
			if strings.TrimSpace(image) != "" {
				imageMatches = false
				for _, container := range dep.Spec.Template.Spec.Containers {
					if container.Image == image {
						imageMatches = true
						break
					}
				}
			}
			if imageMatches &&
				status.ObservedGeneration >= dep.Generation &&
				status.UpdatedReplicas >= expected &&
				status.AvailableReplicas >= expected &&
				status.ReadyReplicas >= expected &&
				status.UnavailableReplicas == 0 {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("deployment rollout timeout for %s/%s: expected_image=%s observed_generation=%d generation=%d updated=%d ready=%d available=%d unavailable=%d", namespace, name, image, dep.Status.ObservedGeneration, dep.Generation, dep.Status.UpdatedReplicas, dep.Status.ReadyReplicas, dep.Status.AvailableReplicas, dep.Status.UnavailableReplicas)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

func int64Ptr(v int64) *int64 { return &v }

func int32Ptr(v int32) *int32 { return &v }

func httpProbe(port int32, initialDelay, period int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler:        corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstrFromInt32(port)}},
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
	}
}

func configureContainerProbes(container *corev1.Container, health *model.ProjectHealthCheck, ports model.ProjectPorts) {
	if health == nil {
		if len(ports) == 0 {
			return
		}
		probePort := int32(ports[0].Port)
		container.LivenessProbe = httpProbe(probePort, 10, 10)
		container.ReadinessProbe = httpProbe(probePort, 5, 5)
		return
	}
	if strings.EqualFold(health.Type, "disabled") {
		return
	}
	container.LivenessProbe = projectProbe(health, ports, 10, 10)
	container.ReadinessProbe = projectProbe(health, ports, 5, 5)
}

func projectProbe(health *model.ProjectHealthCheck, ports model.ProjectPorts, initialDelay, period int32) *corev1.Probe {
	if health.InitialDelaySeconds != nil {
		initialDelay = int32(*health.InitialDelaySeconds)
	}
	if health.PeriodSeconds != nil {
		period = int32(*health.PeriodSeconds)
	}
	probe := &corev1.Probe{
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
	}
	if health.TimeoutSeconds != nil {
		probe.TimeoutSeconds = int32(*health.TimeoutSeconds)
	}
	port := healthProbePort(health.Port, ports)
	switch strings.ToLower(strings.TrimSpace(health.Type)) {
	case "tcp":
		probe.ProbeHandler = corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: port}}
	default:
		path := strings.TrimSpace(health.Path)
		if path == "" {
			path = "/health"
		}
		probe.ProbeHandler = corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: path, Port: port}}
	}
	return probe
}

func healthProbePort(port any, ports model.ProjectPorts) intstr.IntOrString {
	switch v := port.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			break
		}
		if n, err := strconv.Atoi(v); err == nil {
			return intstr.FromInt(n)
		}
		return intstr.FromString(v)
	case float64:
		return intstr.FromInt(int(v))
	case int:
		return intstr.FromInt(v)
	case int32:
		return intstr.FromInt32(v)
	case int64:
		return intstr.FromInt(int(v))
	}
	if len(ports) > 0 {
		if strings.TrimSpace(ports[0].Name) != "" {
			return intstr.FromString(ports[0].Name)
		}
		return intstr.FromInt(ports[0].Port)
	}
	return intstr.FromInt(8080)
}

func (m *Manager) reconcileProjectVolumes(ctx context.Context, namespace, projectName string, specs []model.ProjectVolume) ([]corev1.Volume, []corev1.VolumeMount, error) {
	volumes := make([]corev1.Volume, 0, len(specs))
	mounts := make([]corev1.VolumeMount, 0, len(specs))
	for _, spec := range specs {
		name := strings.TrimSpace(spec.Name)
		mountPath := strings.TrimSpace(spec.MountPath)
		if name == "" || mountPath == "" {
			return nil, nil, fmt.Errorf("volume name and mountPath are required")
		}
		mounts = append(mounts, corev1.VolumeMount{Name: name, MountPath: mountPath})
		switch strings.ToLower(strings.TrimSpace(spec.Type)) {
		case "emptydir", "emptyDir":
			volumes = append(volumes, corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
		case "pvc":
			claimName := projectVolumeClaimName(projectName, name)
			if err := m.reconcilePVC(ctx, namespace, projectName, claimName, spec); err != nil {
				return nil, nil, err
			}
			volumes = append(volumes, corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName}}})
		case "existingpvc":
			claimName := strings.TrimSpace(spec.ClaimName)
			if claimName == "" {
				return nil, nil, fmt.Errorf("existing PVC volume %s requires claimName", name)
			}
			volumes = append(volumes, corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName}}})
		default:
			return nil, nil, fmt.Errorf("unsupported volume type %q for volume %s", spec.Type, name)
		}
	}
	return volumes, mounts, nil
}

func (m *Manager) reconcilePVC(ctx context.Context, namespace, projectName, claimName string, spec model.ProjectVolume) error {
	if strings.TrimSpace(spec.Size) == "" {
		return fmt.Errorf("pvc volume %s requires size", spec.Name)
	}
	labels := Labels(projectName)
	accessModes := make([]corev1.PersistentVolumeAccessMode, 0, len(spec.AccessModes))
	for _, mode := range spec.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode))
	}
	if len(accessModes) == 0 {
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: claimName, Namespace: namespace, Labels: labels},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(spec.Size)},
			},
		},
	}
	if storageClass := strings.TrimSpace(spec.StorageClassName); storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}
	_, err := m.Clientset.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, claimName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		current.Labels = pvc.Labels
		current.Spec.AccessModes = pvc.Spec.AccessModes
		current.Spec.Resources = pvc.Spec.Resources
		_, err = m.Clientset.CoreV1().PersistentVolumeClaims(namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}

func projectVolumeClaimName(projectName, volumeName string) string {
	return projectName + "-" + volumeName
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
