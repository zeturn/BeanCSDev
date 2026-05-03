package k8s

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type RuntimeOverview struct {
	Namespaces  []NamespaceSummary  `json:"namespaces"`
	Nodes       []NodeSummary       `json:"nodes"`
	Pods        []PodSummary        `json:"pods"`
	Deployments []DeploymentSummary `json:"deployments"`
	Services    []ServiceSummary    `json:"services"`
	Ingresses   []IngressSummary    `json:"ingresses"`
}

type NetworkOverview struct {
	Services        []ServiceSummary       `json:"services"`
	Ingresses       []IngressSummary       `json:"ingresses"`
	Endpoints       []EndpointSummary      `json:"endpoints"`
	NetworkPolicies []NetworkPolicySummary `json:"network_policies"`
	Access          []ServiceAccessSummary `json:"access"`
	Controllers     NetworkControllers     `json:"controllers"`
	CheckedAt       time.Time              `json:"checked_at"`
}

type EndpointSummary struct {
	Namespace  string    `json:"namespace"`
	Name       string    `json:"name"`
	Addresses  []string  `json:"addresses"`
	Ports      []string  `json:"ports"`
	AgeSeconds int64     `json:"age_seconds"`
	CreatedAt  time.Time `json:"created_at"`
}

type NetworkPolicySummary struct {
	Namespace    string            `json:"namespace"`
	Name         string            `json:"name"`
	PodSelector  map[string]string `json:"pod_selector,omitempty"`
	PolicyTypes  []string          `json:"policy_types"`
	IngressRules int               `json:"ingress_rules"`
	EgressRules  int               `json:"egress_rules"`
	AgeSeconds   int64             `json:"age_seconds"`
	CreatedAt    time.Time         `json:"created_at"`
}

type ServiceAccessSummary struct {
	Namespace    string   `json:"namespace"`
	Service      string   `json:"service"`
	Type         string   `json:"type"`
	Hosts        []string `json:"hosts,omitempty"`
	URLs         []string `json:"urls"`
	TLS          bool     `json:"tls"`
	Ingress      string   `json:"ingress,omitempty"`
	Class        string   `json:"class,omitempty"`
	LoadBalancer string   `json:"load_balancer,omitempty"`
	NodePorts    []string `json:"node_ports,omitempty"`
}

type NetworkControllers struct {
	TraefikNamespaces   []string `json:"traefik_namespaces"`
	TailscaleNamespaces []string `json:"tailscale_namespaces"`
	TraefikIngresses    int      `json:"traefik_ingresses"`
	TailscaleIngresses  int      `json:"tailscale_ingresses"`
	TLSIngresses        int      `json:"tls_ingresses"`
}

type ClusterDashboard struct {
	ClusterName       string                `json:"cluster_name"`
	Status            string                `json:"status"`
	Healthy           bool                  `json:"healthy"`
	Nodes             ClusterNodeTotals     `json:"nodes"`
	KubernetesVersion string                `json:"kubernetes_version"`
	K3sVersion        string                `json:"k3s_version,omitempty"`
	Resources         ClusterResourceTotals `json:"resources"`
	Pods              ClusterPodTotals      `json:"pods"`
	Alerts            []ClusterAlert        `json:"alerts"`
	Events            []EventSummary        `json:"events"`
	UptimeSeconds     int64                 `json:"uptime_seconds"`
	StartedAt         time.Time             `json:"started_at,omitempty"`
	MetricsAvailable  bool                  `json:"metrics_available"`
	MetricsError      string                `json:"metrics_error,omitempty"`
	CheckedAt         time.Time             `json:"checked_at"`
}

type ClusterNodeTotals struct {
	Total    int `json:"total"`
	Ready    int `json:"ready"`
	NotReady int `json:"not_ready"`
	Server   int `json:"server"`
	Agent    int `json:"agent"`
}

type ClusterResourceTotals struct {
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryPercent    float64 `json:"memory_percent"`
	DiskPercent      float64 `json:"disk_percent"`
	CPUUsedMillis    int64   `json:"cpu_used_millis"`
	CPUTotalMillis   int64   `json:"cpu_total_millis"`
	MemoryUsedBytes  int64   `json:"memory_used_bytes"`
	MemoryTotalBytes int64   `json:"memory_total_bytes"`
	DiskUsedBytes    int64   `json:"disk_used_bytes"`
	DiskTotalBytes   int64   `json:"disk_total_bytes"`
}

type ClusterPodTotals struct {
	Total    int `json:"total"`
	Running  int `json:"running"`
	Abnormal int `json:"abnormal"`
	Pending  int `json:"pending"`
	Failed   int `json:"failed"`
}

type ClusterAlert struct {
	Severity  string    `json:"severity"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Object    string    `json:"object"`
	LastSeen  time.Time `json:"last_seen"`
	Namespace string    `json:"namespace,omitempty"`
}

type NodeSummary struct {
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	Schedulable bool              `json:"schedulable"`
	Roles       []string          `json:"roles"`
	Version     string            `json:"version"`
	InternalIP  string            `json:"internal_ip,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	AgeSeconds  int64             `json:"age_seconds"`
	CreatedAt   time.Time         `json:"created_at"`
}

type NodeDetail struct {
	Summary          NodeSummary            `json:"summary"`
	Addresses        map[string]string      `json:"addresses"`
	Capacity         NodeResources          `json:"capacity"`
	Allocatable      NodeResources          `json:"allocatable"`
	Usage            *NodeResourceUsage     `json:"usage,omitempty"`
	Disk             *NodeDiskUsage         `json:"disk,omitempty"`
	Network          *NodeNetworkUsage      `json:"network,omitempty"`
	Conditions       []NodeConditionSummary `json:"conditions"`
	Taints           []string               `json:"taints"`
	Pods             []PodSummary           `json:"pods"`
	Labels           map[string]string      `json:"labels,omitempty"`
	Annotations      map[string]string      `json:"annotations,omitempty"`
	SystemInfo       map[string]string      `json:"system_info"`
	MetricsAvailable bool                   `json:"metrics_available"`
	MetricsError     string                 `json:"metrics_error,omitempty"`
	CheckedAt        time.Time              `json:"checked_at"`
}

type NodeResources struct {
	CPUMillis      int64  `json:"cpu_millis"`
	MemoryBytes    int64  `json:"memory_bytes"`
	Pods           int64  `json:"pods"`
	EphemeralBytes int64  `json:"ephemeral_bytes"`
	CPU            string `json:"cpu"`
	Memory         string `json:"memory"`
	Ephemeral      string `json:"ephemeral"`
}

type NodeResourceUsage struct {
	CPUMillis                int64   `json:"cpu_millis"`
	MemoryBytes              int64   `json:"memory_bytes"`
	CPU                      string  `json:"cpu"`
	Memory                   string  `json:"memory"`
	CPUAllocatablePercent    float64 `json:"cpu_allocatable_percent"`
	MemoryAllocatablePercent float64 `json:"memory_allocatable_percent"`
	CPUCapacityPercent       float64 `json:"cpu_capacity_percent"`
	MemoryCapacityPercent    float64 `json:"memory_capacity_percent"`
}

type NodeDiskUsage struct {
	UsedBytes     int64   `json:"used_bytes"`
	CapacityBytes int64   `json:"capacity_bytes"`
	UsedPercent   float64 `json:"used_percent"`
}

type NodeNetworkUsage struct {
	RxBytes int64 `json:"rx_bytes"`
	TxBytes int64 `json:"tx_bytes"`
}

type NodeHealthCheck struct {
	Name         string           `json:"name"`
	Healthy      bool             `json:"healthy"`
	Status       string           `json:"status"`
	Checks       []NodeHealthItem `json:"checks"`
	AbnormalPods []PodSummary     `json:"abnormal_pods"`
	MetricsError string           `json:"metrics_error,omitempty"`
	CheckedAt    time.Time        `json:"checked_at"`
}

type NodeHealthItem struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Severity string `json:"severity"`
	Message  string `json:"message,omitempty"`
}

type NodeJoinCommand struct {
	Role       string `json:"role"`
	Configured bool   `json:"configured"`
	Command    string `json:"command,omitempty"`
	Message    string `json:"message,omitempty"`
}

type DrainNodeResult struct {
	Node        string   `json:"node"`
	EvictedPods []string `json:"evicted_pods"`
	SkippedPods []string `json:"skipped_pods"`
}

type NodeConditionSummary struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastHeartbeatTime  time.Time `json:"last_heartbeat_time,omitempty"`
	LastTransitionTime time.Time `json:"last_transition_time,omitempty"`
}

type PodSummary struct {
	Namespace       string            `json:"namespace"`
	Name            string            `json:"name"`
	Status          string            `json:"status"`
	ReadyContainers int               `json:"ready_containers"`
	TotalContainers int               `json:"total_containers"`
	Restarts        int32             `json:"restarts"`
	NodeName        string            `json:"node_name,omitempty"`
	PodIP           string            `json:"pod_ip,omitempty"`
	Containers      []string          `json:"containers,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	AgeSeconds      int64             `json:"age_seconds"`
	CreatedAt       time.Time         `json:"created_at"`
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
	Namespace  string            `json:"namespace"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	ClusterIP  string            `json:"cluster_ip,omitempty"`
	ExternalIP string            `json:"external_ip,omitempty"`
	Ports      []string          `json:"ports"`
	Selector   map[string]string `json:"selector,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	AgeSeconds int64             `json:"age_seconds"`
	CreatedAt  time.Time         `json:"created_at"`
}

type IngressSummary struct {
	Namespace  string    `json:"namespace"`
	Name       string    `json:"name"`
	Class      string    `json:"class,omitempty"`
	Hosts      []string  `json:"hosts"`
	Address    string    `json:"address,omitempty"`
	Services   []string  `json:"services,omitempty"`
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

type LogTarget struct {
	Namespace string
	Pod       string
	Container string
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
			Selector:   svc.Spec.Selector,
			Labels:     svc.Labels,
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
			Services:   ingressBackendServices(ing),
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

func (m *Manager) NetworkOverview(ctx context.Context) (*NetworkOverview, error) {
	if err := m.ensure(); err != nil {
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
	endpoints, err := m.listEndpointSummaries(ctx)
	if err != nil {
		return nil, err
	}
	policies, err := m.listNetworkPolicySummaries(ctx)
	if err != nil {
		return nil, err
	}
	return &NetworkOverview{
		Services:        services,
		Ingresses:       ingresses,
		Endpoints:       endpoints,
		NetworkPolicies: policies,
		Access:          serviceAccessSummaries(services, ingresses),
		Controllers: NetworkControllers{
			TraefikNamespaces:   m.PublicIngressNamespaces,
			TailscaleNamespaces: m.PrivateIngressNamespaces,
			TraefikIngresses:    countIngressClass(ingresses, "traefik"),
			TailscaleIngresses:  countIngressClass(ingresses, "tailscale"),
			TLSIngresses:        countTLSIngresses(ingresses),
		},
		CheckedAt: time.Now().UTC(),
	}, nil
}

func (m *Manager) ClusterDashboard(ctx context.Context) (*ClusterDashboard, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := &ClusterDashboard{
		ClusterName:  m.ClusterName,
		Status:       "Healthy",
		Healthy:      true,
		CheckedAt:    now,
		Alerts:       []ClusterAlert{},
		Events:       []EventSummary{},
		MetricsError: "",
	}
	if out.ClusterName == "" {
		out.ClusterName = "production-k3s"
	}

	nodes, err := m.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	pods, err := m.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	events, err := m.Clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if version, err := m.Clientset.Discovery().ServerVersion(); err == nil && version != nil {
		out.KubernetesVersion = version.GitVersion
	}

	var startedAt time.Time
	for _, node := range nodes.Items {
		out.Nodes.Total++
		if nodeReadyStatus(node) == "Ready" {
			out.Nodes.Ready++
		} else {
			out.Nodes.NotReady++
			out.Healthy = false
		}
		if nodeIsServer(node) {
			out.Nodes.Server++
		} else {
			out.Nodes.Agent++
		}
		if out.K3sVersion == "" && strings.Contains(strings.ToLower(node.Status.NodeInfo.KubeletVersion), "k3s") {
			out.K3sVersion = node.Status.NodeInfo.KubeletVersion
		}
		out.Resources.CPUTotalMillis += nodeResources(node.Status.Allocatable).CPUMillis
		out.Resources.MemoryTotalBytes += nodeResources(node.Status.Allocatable).MemoryBytes
		if startedAt.IsZero() || node.CreationTimestamp.Time.Before(startedAt) {
			startedAt = node.CreationTimestamp.Time
		}
	}
	if !startedAt.IsZero() {
		out.StartedAt = startedAt
		out.UptimeSeconds = int64(now.Sub(startedAt).Seconds())
	}

	for _, pod := range pods.Items {
		out.Pods.Total++
		switch pod.Status.Phase {
		case corev1.PodRunning:
			out.Pods.Running++
		case corev1.PodPending:
			out.Pods.Pending++
		case corev1.PodFailed:
			out.Pods.Failed++
		}
		if podAbnormal(pod) {
			out.Pods.Abnormal++
			out.Healthy = false
			if len(out.Alerts) < 12 {
				out.Alerts = append(out.Alerts, podAlert(pod))
			}
		}
	}

	if err := m.applyDashboardMetrics(ctx, nodes.Items, out); err != nil {
		out.MetricsError = err.Error()
	}
	out.Resources.CPUPercent = percent(out.Resources.CPUUsedMillis, out.Resources.CPUTotalMillis)
	out.Resources.MemoryPercent = percent(out.Resources.MemoryUsedBytes, out.Resources.MemoryTotalBytes)
	out.Resources.DiskPercent = percent(out.Resources.DiskUsedBytes, out.Resources.DiskTotalBytes)
	if out.Resources.CPUUsedMillis > 0 || out.Resources.MemoryUsedBytes > 0 {
		out.MetricsAvailable = true
	}

	warnings := dashboardWarningEvents(events.Items)
	out.Events = warnings
	for _, event := range warnings {
		if len(out.Alerts) >= 12 {
			break
		}
		out.Alerts = append(out.Alerts, ClusterAlert{
			Severity: strings.ToLower(event.Type),
			Reason:   event.Reason,
			Message:  event.Message,
			Object:   event.Object,
			LastSeen: event.LastSeen,
		})
	}
	if out.Nodes.NotReady > 0 || out.Pods.Abnormal > 0 || len(out.Alerts) > 0 {
		out.Status = "Degraded"
	}
	if out.Nodes.Total == 0 || out.Nodes.Ready == 0 {
		out.Status = "NotReady"
		out.Healthy = false
	}
	return out, nil
}

func (m *Manager) Logs(ctx context.Context, namespace, projectName string, tail int64, container string) (string, error) {
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
	return m.logsForTargets(ctx, logTargetsForPods(pods, container), tail), nil
}

func (m *Manager) ProjectLogTargets(ctx context.Context, namespace, projectName, container string) ([]LogTarget, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	pods, err := m.PodStatus(ctx, namespace, projectName)
	if err != nil {
		return nil, err
	}
	return logTargetsForPods(pods, container), nil
}

func (m *Manager) StreamLogs(ctx context.Context, targets []LogTarget, tail int64, follow bool, writer *bufio.Writer) {
	if len(targets) == 0 {
		writeFlushed(writer, []byte("No matching containers found.\n"))
		return
	}
	streamWriter := &flushingLogWriter{writer: writer}
	var wg sync.WaitGroup
	for _, target := range targets {
		target := target
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.streamLogTarget(ctx, target, tail, follow, streamWriter)
		}()
	}
	wg.Wait()
}

func (m *Manager) streamLogTarget(ctx context.Context, target LogTarget, tail int64, follow bool, writer io.Writer) {
	_, _ = fmt.Fprintf(writer, "==> %s/%s <==\n", target.Pod, target.Container)
	req := m.Clientset.CoreV1().Pods(target.Namespace).GetLogs(target.Pod, &corev1.PodLogOptions{
		Container:  target.Container,
		TailLines:  &tail,
		Follow:     follow,
		Timestamps: true,
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(writer, "log stream unavailable: %s\n", err.Error())
		return
	}
	defer stream.Close()
	if _, err := io.Copy(writer, stream); err != nil {
		_, _ = fmt.Fprintf(writer, "\nlog read failed: %s\n", err.Error())
	}
	_, _ = writer.Write([]byte("\n"))
}

func (m *Manager) logsForPods(ctx context.Context, pods []corev1.Pod, tail int64) (string, error) {
	return m.logsForTargets(ctx, logTargetsForPods(pods, ""), tail), nil
}

func (m *Manager) logsForTargets(ctx context.Context, targets []LogTarget, tail int64) string {
	var buf bytes.Buffer
	if len(targets) == 0 {
		return "No matching containers found.\n"
	}
	for _, target := range targets {
		buf.WriteString("==> ")
		buf.WriteString(target.Pod)
		buf.WriteString("/")
		buf.WriteString(target.Container)
		buf.WriteString(" <==\n")
		req := m.Clientset.CoreV1().Pods(target.Namespace).GetLogs(target.Pod, &corev1.PodLogOptions{Container: target.Container, TailLines: &tail, Timestamps: true})
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
	return buf.String()
}

func logTargetsForPods(pods []corev1.Pod, containerFilter string) []LogTarget {
	containerFilter = strings.TrimSpace(containerFilter)
	targets := []LogTarget{}
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			if containerFilter != "" && container.Name != containerFilter {
				continue
			}
			targets = append(targets, LogTarget{
				Namespace: pod.Namespace,
				Pod:       pod.Name,
				Container: container.Name,
			})
		}
	}
	return targets
}

type flushingLogWriter struct {
	mu     sync.Mutex
	writer *bufio.Writer
}

func (w *flushingLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.writer.Write(p)
	if flushErr := w.writer.Flush(); err == nil {
		err = flushErr
	}
	return n, err
}

func writeFlushed(writer *bufio.Writer, p []byte) {
	_, _ = writer.Write(p)
	_ = writer.Flush()
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

func (m *Manager) NodeDetail(ctx context.Context, name string) (*NodeDetail, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	node, err := m.Clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	pods, err := m.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{FieldSelector: "spec.nodeName=" + name})
	if err != nil {
		return nil, err
	}
	out := &NodeDetail{
		Summary:     nodeSummary(*node),
		Addresses:   nodeAddresses(*node),
		Capacity:    nodeResources(node.Status.Capacity),
		Allocatable: nodeResources(node.Status.Allocatable),
		Conditions:  nodeConditions(*node),
		Taints:      nodeTaints(*node),
		Pods:        make([]PodSummary, 0, len(pods.Items)),
		Labels:      node.Labels,
		Annotations: node.Annotations,
		SystemInfo:  nodeSystemInfo(*node),
		CheckedAt:   time.Now().UTC(),
	}
	for _, pod := range pods.Items {
		out.Pods = append(out.Pods, podSummary(pod))
	}
	usage, metricsErr := m.nodeMetrics(ctx, name, out.Capacity, out.Allocatable)
	if metricsErr != nil {
		out.MetricsError = metricsErr.Error()
	} else {
		out.MetricsAvailable = true
		out.Usage = usage
	}
	if disk, network, err := m.nodeUsageStats(ctx, name); err == nil {
		out.Disk = disk
		out.Network = network
	} else if out.MetricsError == "" {
		out.MetricsError = err.Error()
	}
	return out, nil
}

func (m *Manager) listNodeSummaries(ctx context.Context) ([]NodeSummary, error) {
	list, err := m.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]NodeSummary, 0, len(list.Items))
	for _, node := range list.Items {
		out = append(out, nodeSummary(node))
	}
	return out, nil
}

func (m *Manager) listPodSummaries(ctx context.Context) ([]PodSummary, error) {
	list, err := m.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]PodSummary, 0, len(list.Items))
	for _, pod := range list.Items {
		out = append(out, podSummary(pod))
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
			Selector:   svc.Spec.Selector,
			Labels:     svc.Labels,
			AgeSeconds: int64(now.Sub(created).Seconds()),
			CreatedAt:  created,
		})
	}
	return out, nil
}

func podContainerNames(pod corev1.Pod) []string {
	out := make([]string, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		out = append(out, container.Name+":"+container.Image)
	}
	return out
}

func nodeSummary(node corev1.Node) NodeSummary {
	now := time.Now()
	created := node.CreationTimestamp.Time
	return NodeSummary{
		Name:        node.Name,
		Status:      nodeReadyStatus(node),
		Schedulable: !node.Spec.Unschedulable,
		Roles:       nodeRoles(node.Labels),
		Version:     node.Status.NodeInfo.KubeletVersion,
		InternalIP:  nodeInternalIP(node),
		Labels:      node.Labels,
		AgeSeconds:  int64(now.Sub(created).Seconds()),
		CreatedAt:   created,
	}
}

func podSummary(pod corev1.Pod) PodSummary {
	now := time.Now()
	created := pod.CreationTimestamp.Time
	ready, total, restarts := podContainerTotals(pod)
	return PodSummary{
		Namespace:       pod.Namespace,
		Name:            pod.Name,
		Status:          string(pod.Status.Phase),
		ReadyContainers: ready,
		TotalContainers: total,
		Restarts:        restarts,
		NodeName:        pod.Spec.NodeName,
		PodIP:           pod.Status.PodIP,
		Containers:      podContainerNames(pod),
		Labels:          pod.Labels,
		AgeSeconds:      int64(now.Sub(created).Seconds()),
		CreatedAt:       created,
	}
}

func nodeAddresses(node corev1.Node) map[string]string {
	out := map[string]string{}
	for _, address := range node.Status.Addresses {
		out[string(address.Type)] = address.Address
	}
	return out
}

func nodeResources(resources corev1.ResourceList) NodeResources {
	cpu := resources.Cpu()
	memory := resources.Memory()
	pods := resources.Pods()
	ephemeral := resources[corev1.ResourceEphemeralStorage]
	out := NodeResources{}
	if cpu != nil {
		out.CPUMillis = cpu.MilliValue()
		out.CPU = cpu.String()
	}
	if memory != nil {
		out.MemoryBytes = memory.Value()
		out.Memory = memory.String()
	}
	if pods != nil {
		out.Pods = pods.Value()
	}
	if !ephemeral.IsZero() {
		out.EphemeralBytes = ephemeral.Value()
		out.Ephemeral = ephemeral.String()
	}
	return out
}

func nodeConditions(node corev1.Node) []NodeConditionSummary {
	out := make([]NodeConditionSummary, 0, len(node.Status.Conditions))
	for _, condition := range node.Status.Conditions {
		out = append(out, NodeConditionSummary{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastHeartbeatTime:  condition.LastHeartbeatTime.Time,
			LastTransitionTime: condition.LastTransitionTime.Time,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type == "Ready" {
			return true
		}
		if out[j].Type == "Ready" {
			return false
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func nodeTaints(node corev1.Node) []string {
	out := make([]string, 0, len(node.Spec.Taints))
	for _, taint := range node.Spec.Taints {
		value := taint.Key
		if taint.Value != "" {
			value += "=" + taint.Value
		}
		value += ":" + string(taint.Effect)
		out = append(out, value)
	}
	return out
}

func nodeSystemInfo(node corev1.Node) map[string]string {
	info := node.Status.NodeInfo
	return map[string]string{
		"architecture":              info.Architecture,
		"boot_id":                   info.BootID,
		"container_runtime_version": info.ContainerRuntimeVersion,
		"kernel_version":            info.KernelVersion,
		"kube_proxy_version":        info.KubeProxyVersion,
		"kubelet_version":           info.KubeletVersion,
		"machine_id":                info.MachineID,
		"operating_system":          info.OperatingSystem,
		"os_image":                  info.OSImage,
		"system_uuid":               info.SystemUUID,
	}
}

func nodeIsServer(node corev1.Node) bool {
	for key := range node.Labels {
		if key == "node-role.kubernetes.io/control-plane" || key == "node-role.kubernetes.io/master" || key == "node-role.kubernetes.io/etcd" {
			return true
		}
	}
	return false
}

func podAbnormal(pod corev1.Pod) bool {
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodUnknown {
		return true
	}
	if pod.Status.Phase == corev1.PodPending {
		return true
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" && status.State.Waiting.Reason != "ContainerCreating" && status.State.Waiting.Reason != "PodInitializing" {
			return true
		}
		if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
			return true
		}
		if status.RestartCount > 5 {
			return true
		}
	}
	return false
}

func podAlert(pod corev1.Pod) ClusterAlert {
	reason := string(pod.Status.Phase)
	message := strings.TrimSpace(pod.Status.Message)
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
			reason = status.State.Waiting.Reason
			message = status.State.Waiting.Message
			break
		}
		if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
			reason = status.State.Terminated.Reason
			message = status.State.Terminated.Message
			break
		}
	}
	if message == "" {
		message = fmt.Sprintf("Pod phase is %s", pod.Status.Phase)
	}
	return ClusterAlert{
		Severity:  "warning",
		Reason:    reason,
		Message:   message,
		Object:    "Pod/" + pod.Name,
		Namespace: pod.Namespace,
		LastSeen:  time.Now().UTC(),
	}
}

func dashboardWarningEvents(events []corev1.Event) []EventSummary {
	out := []EventSummary{}
	for _, event := range events {
		if event.Type != corev1.EventTypeWarning {
			continue
		}
		lastSeen := event.LastTimestamp.Time
		if lastSeen.IsZero() {
			lastSeen = event.EventTime.Time
		}
		firstSeen := event.FirstTimestamp.Time
		out = append(out, EventSummary{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Object:    event.InvolvedObject.Kind + "/" + event.InvolvedObject.Namespace + "/" + event.InvolvedObject.Name,
			Count:     event.Count,
			LastSeen:  lastSeen,
			FirstSeen: firstSeen,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen.After(out[j].LastSeen) })
	if len(out) > 10 {
		return out[:10]
	}
	return out
}

func (m *Manager) applyDashboardMetrics(ctx context.Context, nodes []corev1.Node, out *ClusterDashboard) error {
	var metricErrs []string
	if err := m.applyNodeMetrics(ctx, out); err != nil {
		metricErrs = append(metricErrs, "metrics.k8s.io: "+err.Error())
	}
	for _, node := range nodes {
		used, total, err := m.nodeDiskUsage(ctx, node.Name)
		if err != nil {
			if len(metricErrs) < 3 {
				metricErrs = append(metricErrs, "disk "+node.Name+": "+err.Error())
			}
			continue
		}
		out.Resources.DiskUsedBytes += used
		out.Resources.DiskTotalBytes += total
	}
	if len(metricErrs) > 0 {
		return errors.New(strings.Join(metricErrs, "; "))
	}
	return nil
}

func (m *Manager) applyNodeMetrics(ctx context.Context, out *ClusterDashboard) error {
	if m.Dynamic == nil {
		return fmt.Errorf("metrics client not configured")
	}
	gvr := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}
	list, err := m.Dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		usage, ok, err := unstructured.NestedStringMap(item.Object, "usage")
		if err != nil || !ok {
			continue
		}
		cpu, err := resource.ParseQuantity(usage["cpu"])
		if err == nil {
			out.Resources.CPUUsedMillis += cpu.MilliValue()
		}
		memory, err := resource.ParseQuantity(usage["memory"])
		if err == nil {
			out.Resources.MemoryUsedBytes += memory.Value()
		}
	}
	return nil
}

type nodeStatsSummary struct {
	Node struct {
		FS struct {
			AvailableBytes uint64 `json:"availableBytes"`
			CapacityBytes  uint64 `json:"capacityBytes"`
		} `json:"fs"`
		Network struct {
			RxBytes uint64 `json:"rxBytes"`
			TxBytes uint64 `json:"txBytes"`
		} `json:"network"`
	} `json:"node"`
}

func (m *Manager) nodeDiskUsage(ctx context.Context, nodeName string) (int64, int64, error) {
	disk, _, err := m.nodeUsageStats(ctx, nodeName)
	if err != nil {
		return 0, 0, err
	}
	return disk.UsedBytes, disk.CapacityBytes, nil
}

func (m *Manager) nodeUsageStats(ctx context.Context, nodeName string) (*NodeDiskUsage, *NodeNetworkUsage, error) {
	raw, err := m.Clientset.CoreV1().RESTClient().
		Get().
		Resource("nodes").
		Name(nodeName).
		SubResource("proxy").
		Suffix("stats/summary").
		DoRaw(ctx)
	if err != nil {
		return nil, nil, err
	}
	var summary nodeStatsSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		return nil, nil, err
	}
	total := int64(summary.Node.FS.CapacityBytes)
	available := int64(summary.Node.FS.AvailableBytes)
	if total <= 0 || available < 0 {
		return nil, nil, fmt.Errorf("node fs metrics unavailable")
	}
	disk := &NodeDiskUsage{
		UsedBytes:     total - available,
		CapacityBytes: total,
		UsedPercent:   percent(total-available, total),
	}
	network := &NodeNetworkUsage{
		RxBytes: int64(summary.Node.Network.RxBytes),
		TxBytes: int64(summary.Node.Network.TxBytes),
	}
	return disk, network, nil
}

func (m *Manager) nodeMetrics(ctx context.Context, name string, capacity, allocatable NodeResources) (*NodeResourceUsage, error) {
	if m.Dynamic == nil {
		return nil, fmt.Errorf("metrics client not configured")
	}
	gvr := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}
	item, err := m.Dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	usage, ok, err := unstructured.NestedStringMap(item.Object, "usage")
	if err != nil || !ok {
		return nil, fmt.Errorf("node metrics usage was not returned")
	}
	cpu, err := resource.ParseQuantity(usage["cpu"])
	if err != nil {
		return nil, err
	}
	memory, err := resource.ParseQuantity(usage["memory"])
	if err != nil {
		return nil, err
	}
	cpuMillis := cpu.MilliValue()
	memoryBytes := memory.Value()
	return &NodeResourceUsage{
		CPUMillis:                cpuMillis,
		MemoryBytes:              memoryBytes,
		CPU:                      cpu.String(),
		Memory:                   memory.String(),
		CPUAllocatablePercent:    percent(cpuMillis, allocatable.CPUMillis),
		MemoryAllocatablePercent: percent(memoryBytes, allocatable.MemoryBytes),
		CPUCapacityPercent:       percent(cpuMillis, capacity.CPUMillis),
		MemoryCapacityPercent:    percent(memoryBytes, capacity.MemoryBytes),
	}, nil
}

func percent(used, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(used) * 100 / float64(total)
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
			Services:   ingressBackendServices(ing),
			TLS:        len(ing.Spec.TLS) > 0,
			AgeSeconds: int64(now.Sub(created).Seconds()),
			CreatedAt:  created,
		})
	}
	return out, nil
}

func (m *Manager) listEndpointSummaries(ctx context.Context) ([]EndpointSummary, error) {
	list, err := m.Clientset.CoreV1().Endpoints("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]EndpointSummary, 0, len(list.Items))
	for _, endpoint := range list.Items {
		created := endpoint.CreationTimestamp.Time
		out = append(out, EndpointSummary{
			Namespace:  endpoint.Namespace,
			Name:       endpoint.Name,
			Addresses:  endpointAddresses(endpoint),
			Ports:      endpointPorts(endpoint),
			AgeSeconds: int64(now.Sub(created).Seconds()),
			CreatedAt:  created,
		})
	}
	return out, nil
}

func (m *Manager) listNetworkPolicySummaries(ctx context.Context) ([]NetworkPolicySummary, error) {
	list, err := m.Clientset.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]NetworkPolicySummary, 0, len(list.Items))
	for _, policy := range list.Items {
		created := policy.CreationTimestamp.Time
		out = append(out, NetworkPolicySummary{
			Namespace:    policy.Namespace,
			Name:         policy.Name,
			PodSelector:  policy.Spec.PodSelector.MatchLabels,
			PolicyTypes:  policyTypes(policy.Spec.PolicyTypes),
			IngressRules: len(policy.Spec.Ingress),
			EgressRules:  len(policy.Spec.Egress),
			AgeSeconds:   int64(now.Sub(created).Seconds()),
			CreatedAt:    created,
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

func endpointAddresses(endpoint corev1.Endpoints) []string {
	out := []string{}
	for _, subset := range endpoint.Subsets {
		for _, address := range subset.Addresses {
			if address.TargetRef != nil {
				out = append(out, address.IP+" ("+address.TargetRef.Kind+"/"+address.TargetRef.Name+")")
			} else {
				out = append(out, address.IP)
			}
		}
	}
	sort.Strings(out)
	return out
}

func endpointPorts(endpoint corev1.Endpoints) []string {
	out := []string{}
	for _, subset := range endpoint.Subsets {
		for _, port := range subset.Ports {
			name := port.Name
			if name == "" {
				name = "port"
			}
			out = append(out, fmt.Sprintf("%s:%d/%s", name, port.Port, port.Protocol))
		}
	}
	sort.Strings(out)
	return out
}

func policyTypes(types []networkingv1.PolicyType) []string {
	out := make([]string, 0, len(types))
	for _, item := range types {
		out = append(out, string(item))
	}
	return out
}

func serviceAccessSummaries(services []ServiceSummary, ingresses []IngressSummary) []ServiceAccessSummary {
	out := []ServiceAccessSummary{}
	for _, svc := range services {
		item := ServiceAccessSummary{
			Namespace:    svc.Namespace,
			Service:      svc.Name,
			Type:         svc.Type,
			URLs:         []string{},
			LoadBalancer: svc.ExternalIP,
			NodePorts:    serviceNodePorts(svc.Ports),
		}
		if svc.ClusterIP != "" && svc.ClusterIP != "None" {
			item.URLs = append(item.URLs, fmt.Sprintf("http://%s.%s.svc.cluster.local", svc.Name, svc.Namespace))
		}
		if svc.ExternalIP != "" {
			item.URLs = append(item.URLs, svc.ExternalIP)
		}
		for _, ing := range ingresses {
			if ing.Namespace != svc.Namespace || !ingressMayRouteService(ing, svc.Name) {
				continue
			}
			item.Ingress = ing.Name
			item.Class = ing.Class
			item.Hosts = append(item.Hosts, ing.Hosts...)
			item.TLS = item.TLS || ing.TLS
			scheme := "http"
			if ing.TLS {
				scheme = "https"
			}
			for _, host := range ing.Hosts {
				item.URLs = append(item.URLs, scheme+"://"+host)
			}
		}
		if len(item.URLs) > 0 || len(item.NodePorts) > 0 {
			out = append(out, item)
		}
	}
	return out
}

func serviceNodePorts(ports []string) []string {
	out := []string{}
	for _, port := range ports {
		parts := strings.Split(port, "/")
		left := strings.Split(parts[0], ":")
		if len(left) >= 3 {
			out = append(out, left[len(left)-1]+"/"+parts[len(parts)-1])
		}
	}
	return out
}

func ingressMayRouteService(ing IngressSummary, serviceName string) bool {
	for _, svc := range ing.Services {
		if svc == serviceName {
			return true
		}
	}
	return strings.HasPrefix(ing.Name, serviceName+"-") || ing.Name == serviceName
}

func countIngressClass(ingresses []IngressSummary, className string) int {
	count := 0
	for _, ing := range ingresses {
		if strings.EqualFold(ing.Class, className) {
			count++
		}
	}
	return count
}

func countTLSIngresses(ingresses []IngressSummary) int {
	count := 0
	for _, ing := range ingresses {
		if ing.TLS {
			count++
		}
	}
	return count
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

func ingressBackendServices(ing networkingv1.Ingress) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil || path.Backend.Service.Name == "" || seen[path.Backend.Service.Name] {
				continue
			}
			seen[path.Backend.Service.Name] = true
			out = append(out, path.Backend.Service.Name)
		}
	}
	sort.Strings(out)
	return out
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
