package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RuntimeOverview struct {
	Namespaces  []NamespaceSummary  `json:"namespaces"`
	Nodes       []NodeSummary       `json:"nodes"`
	Pods        []PodSummary        `json:"pods"`
	Deployments []DeploymentSummary `json:"deployments"`
	Services    []ServiceSummary    `json:"services"`
	Ingresses   []IngressSummary    `json:"ingresses"`
}

type NodeSummary struct {
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Roles      []string          `json:"roles"`
	Version    string            `json:"version"`
	InternalIP string            `json:"internal_ip,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	AgeSeconds int64             `json:"age_seconds"`
	CreatedAt  time.Time         `json:"created_at"`
}

type PodSummary struct {
	Namespace       string    `json:"namespace"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	ReadyContainers int       `json:"ready_containers"`
	TotalContainers int       `json:"total_containers"`
	Restarts        int32     `json:"restarts"`
	NodeName        string    `json:"node_name,omitempty"`
	PodIP           string    `json:"pod_ip,omitempty"`
	AgeSeconds      int64     `json:"age_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}

type DeploymentSummary struct {
	Namespace         string    `json:"namespace"`
	Name              string    `json:"name"`
	ReadyReplicas     int32     `json:"ready_replicas"`
	AvailableReplicas int32     `json:"available_replicas"`
	Replicas          int32     `json:"replicas"`
	UpdatedReplicas   int32     `json:"updated_replicas"`
	AgeSeconds        int64     `json:"age_seconds"`
	CreatedAt         time.Time `json:"created_at"`
}

type ServiceSummary struct {
	Namespace  string    `json:"namespace"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	ClusterIP  string    `json:"cluster_ip,omitempty"`
	ExternalIP string    `json:"external_ip,omitempty"`
	Ports      []string  `json:"ports"`
	AgeSeconds int64     `json:"age_seconds"`
	CreatedAt  time.Time `json:"created_at"`
}

type IngressSummary struct {
	Namespace  string    `json:"namespace"`
	Name       string    `json:"name"`
	Class      string    `json:"class,omitempty"`
	Hosts      []string  `json:"hosts"`
	Address    string    `json:"address,omitempty"`
	TLS        bool      `json:"tls"`
	AgeSeconds int64     `json:"age_seconds"`
	CreatedAt  time.Time `json:"created_at"`
}

type ProjectRuntimeStatus struct {
	Deployment *ProjectDeploymentStatus `json:"deployment,omitempty"`
	Pods       []ProjectPodStatus       `json:"pods"`
	Services   []ServiceSummary         `json:"services"`
	Ingresses  []IngressSummary         `json:"ingresses"`
	Events     []EventSummary           `json:"events"`
}

type ProjectDeploymentStatus struct {
	Name              string    `json:"name"`
	ReadyReplicas     int32     `json:"ready_replicas"`
	AvailableReplicas int32     `json:"available_replicas"`
	Replicas          int32     `json:"replicas"`
	UpdatedReplicas   int32     `json:"updated_replicas"`
	Conditions        []string  `json:"conditions"`
	CreatedAt         time.Time `json:"created_at"`
}

type ProjectPodStatus struct {
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	ReadyContainers int       `json:"ready_containers"`
	TotalContainers int       `json:"total_containers"`
	Restarts        int32     `json:"restarts"`
	NodeName        string    `json:"node_name,omitempty"`
	PodIP           string    `json:"pod_ip,omitempty"`
	Reason          string    `json:"reason,omitempty"`
	Message         string    `json:"message,omitempty"`
	Containers      []string  `json:"containers"`
	CreatedAt       time.Time `json:"created_at"`
}

type EventSummary struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Object    string    `json:"object"`
	Count     int32     `json:"count"`
	LastSeen  time.Time `json:"last_seen"`
	FirstSeen time.Time `json:"first_seen"`
}

func (m *Manager) PodStatus(ctx context.Context, namespace, projectName string) ([]corev1.Pod, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	list, err := m.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app=" + projectName + ",managed-by=beancs"})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (m *Manager) ProjectRuntimeStatus(ctx context.Context, namespace, projectName string) (*ProjectRuntimeStatus, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	out := &ProjectRuntimeStatus{Pods: []ProjectPodStatus{}, Services: []ServiceSummary{}, Ingresses: []IngressSummary{}, Events: []EventSummary{}}

	if deploy, err := m.Clientset.AppsV1().Deployments(namespace).Get(ctx, projectName, metav1.GetOptions{}); err == nil {
		out.Deployment = projectDeploymentStatus(*deploy)
	}

	pods, err := m.PodStatus(ctx, namespace, projectName)
	if err != nil {
		return nil, err
	}
	for _, pod := range pods {
		out.Pods = append(out.Pods, projectPodStatus(pod))
	}

	services, err := m.Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app=" + projectName + ",managed-by=beancs"})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for _, svc := range services.Items {
		created := svc.CreationTimestamp.Time
		out.Services = append(out.Services, ServiceSummary{
			Namespace:  svc.Namespace,
			Name:       svc.Name,
			Type:       string(svc.Spec.Type),
			ClusterIP:  serviceClusterIP(svc),
			ExternalIP: serviceExternalIP(svc),
			Ports:      servicePorts(svc),
			AgeSeconds: int64(now.Sub(created).Seconds()),
			CreatedAt:  created,
		})
	}

	ingresses, err := m.Clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app=" + projectName + ",managed-by=beancs"})
	if err != nil {
		return nil, err
	}
	for _, ing := range ingresses.Items {
		created := ing.CreationTimestamp.Time
		out.Ingresses = append(out.Ingresses, IngressSummary{
			Namespace:  ing.Namespace,
			Name:       ing.Name,
			Class:      ingressClass(ing),
			Hosts:      ingressHosts(ing),
			Address:    ingressAddress(ing),
			TLS:        len(ing.Spec.TLS) > 0,
			AgeSeconds: int64(now.Sub(created).Seconds()),
			CreatedAt:  created,
		})
	}

	events, err := m.Clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, event := range events.Items {
		if event.InvolvedObject.Name != projectName && !strings.HasPrefix(event.InvolvedObject.Name, projectName+"-") {
			continue
		}
		out.Events = append(out.Events, EventSummary{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Object:    event.InvolvedObject.Kind + "/" + event.InvolvedObject.Name,
			Count:     event.Count,
			LastSeen:  event.LastTimestamp.Time,
			FirstSeen: event.FirstTimestamp.Time,
		})
	}
	sort.Slice(out.Events, func(i, j int) bool { return out.Events[i].LastSeen.After(out.Events[j].LastSeen) })
	if len(out.Events) > 20 {
		out.Events = out.Events[:20]
	}

	return out, nil
}

func (m *Manager) RuntimeOverview(ctx context.Context) (*RuntimeOverview, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}

	namespaces, err := m.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	nodes, err := m.listNodeSummaries(ctx)
	if err != nil {
		return nil, err
	}
	pods, err := m.listPodSummaries(ctx)
	if err != nil {
		return nil, err
	}
	deployments, err := m.listDeploymentSummaries(ctx)
	if err != nil {
		return nil, err
	}
	services, err := m.listServiceSummaries(ctx)
	if err != nil {
		return nil, err
	}
	ingresses, err := m.listIngressSummaries(ctx)
	if err != nil {
		return nil, err
	}

	return &RuntimeOverview{
		Namespaces:  namespaces,
		Nodes:       nodes,
		Pods:        pods,
		Deployments: deployments,
		Services:    services,
		Ingresses:   ingresses,
	}, nil
}

func (m *Manager) Logs(ctx context.Context, namespace, projectName string, tail int64) (string, error) {
	if err := m.ensure(); err != nil {
		return "", err
	}
	pods, err := m.PodStatus(ctx, namespace, projectName)
	if err != nil {
		return "", err
	}
	if len(pods) == 0 {
		return "", nil
	}
	var buf bytes.Buffer
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			buf.WriteString("==> ")
			buf.WriteString(pod.Name)
			buf.WriteString("/")
			buf.WriteString(container.Name)
			buf.WriteString(" <==\n")
			req := m.Clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: container.Name, TailLines: &tail, Timestamps: true})
			stream, err := req.Stream(ctx)
			if err != nil {
				buf.WriteString("log stream unavailable: ")
				buf.WriteString(err.Error())
				buf.WriteString("\n")
				continue
			}
			_, err = io.Copy(&buf, stream)
			_ = stream.Close()
			if err != nil {
				buf.WriteString("log read failed: ")
				buf.WriteString(err.Error())
				buf.WriteString("\n")
			}
			buf.WriteString("\n")
		}
	}
	return buf.String(), nil
}

func (m *Manager) Nodes(ctx context.Context) ([]corev1.Node, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	list, err := m.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (m *Manager) listNodeSummaries(ctx context.Context) ([]NodeSummary, error) {
	list, err := m.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]NodeSummary, 0, len(list.Items))
	for _, node := range list.Items {
		created := node.CreationTimestamp.Time
		out = append(out, NodeSummary{
			Name:       node.Name,
			Status:     nodeReadyStatus(node),
			Roles:      nodeRoles(node.Labels),
			Version:    node.Status.NodeInfo.KubeletVersion,
			InternalIP: nodeInternalIP(node),
			Labels:     node.Labels,
			AgeSeconds: int64(now.Sub(created).Seconds()),
			CreatedAt:  created,
		})
	}
	return out, nil
}

func (m *Manager) listPodSummaries(ctx context.Context) ([]PodSummary, error) {
	list, err := m.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]PodSummary, 0, len(list.Items))
	for _, pod := range list.Items {
		created := pod.CreationTimestamp.Time
		ready, total, restarts := podContainerTotals(pod)
		out = append(out, PodSummary{
			Namespace:       pod.Namespace,
			Name:            pod.Name,
			Status:          string(pod.Status.Phase),
			ReadyContainers: ready,
			TotalContainers: total,
			Restarts:        restarts,
			NodeName:        pod.Spec.NodeName,
			PodIP:           pod.Status.PodIP,
			AgeSeconds:      int64(now.Sub(created).Seconds()),
			CreatedAt:       created,
		})
	}
	return out, nil
}

func (m *Manager) listDeploymentSummaries(ctx context.Context) ([]DeploymentSummary, error) {
	list, err := m.Clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]DeploymentSummary, 0, len(list.Items))
	for _, deploy := range list.Items {
		created := deploy.CreationTimestamp.Time
		out = append(out, DeploymentSummary{
			Namespace:         deploy.Namespace,
			Name:              deploy.Name,
			ReadyReplicas:     deploy.Status.ReadyReplicas,
			AvailableReplicas: deploy.Status.AvailableReplicas,
			Replicas:          deploymentReplicas(deploy),
			UpdatedReplicas:   deploy.Status.UpdatedReplicas,
			AgeSeconds:        int64(now.Sub(created).Seconds()),
			CreatedAt:         created,
		})
	}
	return out, nil
}

func (m *Manager) listServiceSummaries(ctx context.Context) ([]ServiceSummary, error) {
	list, err := m.Clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]ServiceSummary, 0, len(list.Items))
	for _, svc := range list.Items {
		created := svc.CreationTimestamp.Time
		out = append(out, ServiceSummary{
			Namespace:  svc.Namespace,
			Name:       svc.Name,
			Type:       string(svc.Spec.Type),
			ClusterIP:  serviceClusterIP(svc),
			ExternalIP: serviceExternalIP(svc),
			Ports:      servicePorts(svc),
			AgeSeconds: int64(now.Sub(created).Seconds()),
			CreatedAt:  created,
		})
	}
	return out, nil
}

func (m *Manager) listIngressSummaries(ctx context.Context) ([]IngressSummary, error) {
	list, err := m.Clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]IngressSummary, 0, len(list.Items))
	for _, ing := range list.Items {
		created := ing.CreationTimestamp.Time
		out = append(out, IngressSummary{
			Namespace:  ing.Namespace,
			Name:       ing.Name,
			Class:      ingressClass(ing),
			Hosts:      ingressHosts(ing),
			Address:    ingressAddress(ing),
			TLS:        len(ing.Spec.TLS) > 0,
			AgeSeconds: int64(now.Sub(created).Seconds()),
			CreatedAt:  created,
		})
	}
	return out, nil
}

func nodeReadyStatus(node corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func nodeRoles(labels map[string]string) []string {
	roles := []string{}
	for key := range labels {
		const prefix = "node-role.kubernetes.io/"
		if strings.HasPrefix(key, prefix) {
			role := strings.TrimPrefix(key, prefix)
			if role == "" {
				role = "node"
			}
			roles = append(roles, role)
		}
	}
	sort.Strings(roles)
	if len(roles) == 0 {
		return []string{"worker"}
	}
	return roles
}

func nodeInternalIP(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address
		}
	}
	return ""
}

func podContainerTotals(pod corev1.Pod) (int, int, int32) {
	ready := 0
	var restarts int32
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			ready++
		}
		restarts += status.RestartCount
	}
	return ready, len(pod.Spec.Containers), restarts
}

func projectDeploymentStatus(deploy appsv1.Deployment) *ProjectDeploymentStatus {
	conditions := make([]string, 0, len(deploy.Status.Conditions))
	for _, condition := range deploy.Status.Conditions {
		conditions = append(conditions, fmt.Sprintf("%s=%s:%s", condition.Type, condition.Status, condition.Reason))
	}
	return &ProjectDeploymentStatus{
		Name:              deploy.Name,
		ReadyReplicas:     deploy.Status.ReadyReplicas,
		AvailableReplicas: deploy.Status.AvailableReplicas,
		Replicas:          deploymentReplicas(deploy),
		UpdatedReplicas:   deploy.Status.UpdatedReplicas,
		Conditions:        conditions,
		CreatedAt:         deploy.CreationTimestamp.Time,
	}
}

func projectPodStatus(pod corev1.Pod) ProjectPodStatus {
	ready, total, restarts := podContainerTotals(pod)
	containers := make([]string, 0, len(pod.Status.ContainerStatuses))
	reason := ""
	message := ""
	for _, status := range pod.Status.ContainerStatuses {
		state := "waiting"
		if status.State.Running != nil {
			state = "running"
		} else if status.State.Terminated != nil {
			state = "terminated"
			if status.State.Terminated.Reason != "" {
				reason = status.State.Terminated.Reason
			}
			if status.State.Terminated.Message != "" {
				message = status.State.Terminated.Message
			}
		} else if status.State.Waiting != nil {
			if status.State.Waiting.Reason != "" {
				reason = status.State.Waiting.Reason
			}
			if status.State.Waiting.Message != "" {
				message = status.State.Waiting.Message
			}
		}
		containers = append(containers, fmt.Sprintf("%s:%s ready=%t restarts=%d", status.Name, state, status.Ready, status.RestartCount))
	}
	if reason == "" {
		reason = pod.Status.Reason
	}
	if message == "" {
		message = pod.Status.Message
	}
	return ProjectPodStatus{
		Name:            pod.Name,
		Status:          string(pod.Status.Phase),
		ReadyContainers: ready,
		TotalContainers: total,
		Restarts:        restarts,
		NodeName:        pod.Spec.NodeName,
		PodIP:           pod.Status.PodIP,
		Reason:          reason,
		Message:         message,
		Containers:      containers,
		CreatedAt:       pod.CreationTimestamp.Time,
	}
}

func deploymentReplicas(deploy appsv1.Deployment) int32 {
	if deploy.Spec.Replicas == nil {
		return 1
	}
	return *deploy.Spec.Replicas
}

func serviceClusterIP(svc corev1.Service) string {
	if svc.Spec.ClusterIP == corev1.ClusterIPNone {
		return "None"
	}
	return svc.Spec.ClusterIP
}

func serviceExternalIP(svc corev1.Service) string {
	ips := append([]string{}, svc.Spec.ExternalIPs...)
	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			ips = append(ips, ingress.IP)
		} else if ingress.Hostname != "" {
			ips = append(ips, ingress.Hostname)
		}
	}
	return strings.Join(ips, ", ")
}

func servicePorts(svc corev1.Service) []string {
	ports := make([]string, 0, len(svc.Spec.Ports))
	for _, port := range svc.Spec.Ports {
		value := strings.Builder{}
		if port.Name != "" {
			value.WriteString(port.Name)
			value.WriteString(":")
		}
		value.WriteString(fmt.Sprintf("%d", port.Port))
		if port.NodePort > 0 {
			value.WriteString(":")
			value.WriteString(fmt.Sprintf("%d", port.NodePort))
		}
		value.WriteString("/")
		value.WriteString(string(port.Protocol))
		ports = append(ports, value.String())
	}
	return ports
}

func ingressClass(ing networkingv1.Ingress) string {
	if ing.Spec.IngressClassName != nil {
		return *ing.Spec.IngressClassName
	}
	return ing.Annotations["kubernetes.io/ingress.class"]
}

func ingressHosts(ing networkingv1.Ingress) []string {
	hosts := []string{}
	for _, rule := range ing.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}
	sort.Strings(hosts)
	return hosts
}

func ingressAddress(ing networkingv1.Ingress) string {
	addresses := []string{}
	for _, item := range ing.Status.LoadBalancer.Ingress {
		if item.IP != "" {
			addresses = append(addresses, item.IP)
		} else if item.Hostname != "" {
			addresses = append(addresses, item.Hostname)
		}
	}
	return strings.Join(addresses, ", ")
}
