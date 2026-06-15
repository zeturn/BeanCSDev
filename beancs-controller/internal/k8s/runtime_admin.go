package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zeturn/beancs-controller/internal/dto"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
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

func (m *Manager) PodLogs(ctx context.Context, namespace, name string, tail int64, container string) (string, error) {
	if err := m.ensure(); err != nil {
		return "", err
	}
	pod, err := m.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return m.logsForTargets(ctx, logTargetsForPods([]corev1.Pod{*pod}, container), tail), nil
}

func (m *Manager) PodLogTargets(ctx context.Context, namespace, name, container string) ([]LogTarget, error) {
	if err := m.ensure(); err != nil {
		return nil, err
	}
	pod, err := m.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return logTargetsForPods([]corev1.Pod{*pod}, container), nil
}

func (m *Manager) PatchNodeLabels(ctx context.Context, name string, labels map[string]string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	node, err := m.Clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if node.Labels == nil {
		node.Labels = map[string]string{}
	}
	for key, value := range labels {
		if strings.TrimSpace(value) == "" {
			delete(node.Labels, key)
		} else {
			node.Labels[key] = value
		}
	}
	_, err = m.Clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

func (m *Manager) UpdateNodeTaints(ctx context.Context, name string, req dto.RuntimeTaintRequest) error {
	if err := m.ensure(); err != nil {
		return err
	}
	node, err := m.Clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	taints := make([]corev1.Taint, 0, len(req.Taints))
	for _, item := range req.Taints {
		taints = append(taints, corev1.Taint{
			Key:    strings.TrimSpace(item.Key),
			Value:  strings.TrimSpace(item.Value),
			Effect: corev1.TaintEffect(item.Effect),
		})
	}
	node.Spec.Taints = taints
	_, err = m.Clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

func (m *Manager) SetNodeSchedulable(ctx context.Context, name string, schedulable bool) error {
	if err := m.ensure(); err != nil {
		return err
	}
	node, err := m.Clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	node.Spec.Unschedulable = !schedulable
	_, err = m.Clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

func (m *Manager) DrainNode(ctx context.Context, name string, req dto.DrainNodeRequest) (*DrainNodeResult, error) {
	if err := m.SetNodeSchedulable(ctx, name, false); err != nil {
		return nil, err
	}
	pods, err := m.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{FieldSelector: "spec.nodeName=" + name})
	if err != nil {
		return nil, err
	}
	result := &DrainNodeResult{Node: name, EvictedPods: []string{}, SkippedPods: []string{}}
	grace := req.GracePeriodSeconds
	for _, pod := range pods.Items {
		key := pod.Namespace + "/" + pod.Name
		if shouldSkipDrainPod(pod, req) {
			result.SkippedPods = append(result.SkippedPods, key)
			continue
		}
		eviction := &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{Name: pod.Name, Namespace: pod.Namespace},
			DeleteOptions: &metav1.DeleteOptions{
				GracePeriodSeconds: &grace,
			},
		}
		err := m.Clientset.CoreV1().Pods(pod.Namespace).EvictV1(ctx, eviction)
		if apierrors.IsNotFound(err) {
			continue
		}
		if apierrors.IsTooManyRequests(err) || apierrors.IsMethodNotSupported(err) {
			err = m.Clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{GracePeriodSeconds: &grace})
		}
		if err != nil {
			return result, fmt.Errorf("drain pod %s: %w", key, err)
		}
		result.EvictedPods = append(result.EvictedPods, key)
	}
	return result, nil
}

func (m *Manager) DeleteNode(ctx context.Context, name string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	err := m.Clientset.CoreV1().Nodes().Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (m *Manager) NodeHealth(ctx context.Context, name string) (*NodeHealthCheck, error) {
	detail, err := m.NodeDetail(ctx, name)
	if err != nil {
		return nil, err
	}
	out := &NodeHealthCheck{Name: name, Healthy: true, Status: "Healthy", CheckedAt: time.Now().UTC()}
	for _, condition := range detail.Conditions {
		item := nodeConditionHealth(condition)
		out.Checks = append(out.Checks, item)
		if item.Severity == "critical" || item.Severity == "warning" {
			out.Healthy = false
		}
	}
	for _, pod := range detail.Pods {
		if pod.Status != string(corev1.PodRunning) || pod.ReadyContainers < pod.TotalContainers || pod.Restarts > 5 {
			out.AbnormalPods = append(out.AbnormalPods, pod)
		}
	}
	if len(out.AbnormalPods) > 0 {
		out.Healthy = false
		out.Checks = append(out.Checks, NodeHealthItem{Name: "Pods", Status: "Warning", Severity: "warning", Message: fmt.Sprintf("%d abnormal pods on this node", len(out.AbnormalPods))})
	}
	if detail.MetricsError != "" {
		out.MetricsError = detail.MetricsError
		out.Checks = append(out.Checks, NodeHealthItem{Name: "Metrics", Status: "Unknown", Severity: "warning", Message: detail.MetricsError})
	}
	if !out.Healthy {
		out.Status = "Degraded"
	}
	return out, nil
}

func (m *Manager) K3sJoinCommand(role string) NodeJoinCommand {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "server" {
		role = "agent"
	}
	out := NodeJoinCommand{Role: role, Configured: m.K3sServerURL != "" && m.K3sJoinToken != ""}
	if !out.Configured {
		out.Message = "Set BEANCS_K3S_SERVER_URL and BEANCS_K3S_JOIN_TOKEN on the controller to generate a node join command."
		return out
	}
	if role == "server" {
		out.Command = fmt.Sprintf("curl -sfL https://get.k3s.io | K3S_TOKEN='%s' sh -s - server --server '%s'", shellQuoteValue(m.K3sJoinToken), shellQuoteValue(m.K3sServerURL))
		return out
	}
	out.Command = fmt.Sprintf("curl -sfL https://get.k3s.io | K3S_URL='%s' K3S_TOKEN='%s' sh -", shellQuoteValue(m.K3sServerURL), shellQuoteValue(m.K3sJoinToken))
	return out
}

func shouldSkipDrainPod(pod corev1.Pod, req dto.DrainNodeRequest) bool {
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return true
	}
	if isMirrorPod(pod) {
		return true
	}
	if isDaemonSetPod(pod) && req.IgnoreDaemonSets {
		return true
	}
	if !req.DeleteEmptyDirData && podUsesEmptyDir(pod) {
		return true
	}
	if !req.Force && len(pod.OwnerReferences) == 0 {
		return true
	}
	return false
}

func isMirrorPod(pod corev1.Pod) bool {
	_, ok := pod.Annotations["kubernetes.io/config.mirror"]
	return ok
}

func isDaemonSetPod(pod corev1.Pod) bool {
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

func podUsesEmptyDir(pod corev1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.EmptyDir != nil {
			return true
		}
	}
	return false
}

func nodeConditionHealth(condition NodeConditionSummary) NodeHealthItem {
	if condition.Type == string(corev1.NodeReady) {
		if condition.Status == string(corev1.ConditionTrue) {
			return NodeHealthItem{Name: condition.Type, Status: "OK", Severity: "ok", Message: condition.Message}
		}
		return NodeHealthItem{Name: condition.Type, Status: condition.Status, Severity: "critical", Message: condition.Message}
	}
	if condition.Status == string(corev1.ConditionTrue) {
		return NodeHealthItem{Name: condition.Type, Status: condition.Status, Severity: "warning", Message: condition.Message}
	}
	return NodeHealthItem{Name: condition.Type, Status: condition.Status, Severity: "ok", Message: condition.Message}
}

func shellQuoteValue(value string) string {
	return strings.ReplaceAll(value, "'", "'\"'\"'")
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
			Type:                  serviceType,
			Selector:              req.Selector,
			Ports:                 runtimeServicePorts(req.Ports),
			LoadBalancerIP:        req.LoadBalancerIP,
			ExternalIPs:           req.ExternalIPs,
			ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicy(req.ExternalTrafficPolicy),
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
	current.Spec.LoadBalancerIP = req.LoadBalancerIP
	current.Spec.ExternalIPs = req.ExternalIPs
	current.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicy(req.ExternalTrafficPolicy)
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

func (m *Manager) UpsertIngress(ctx context.Context, req dto.CreateIngressRequest) error {
	if err := m.ensure(); err != nil {
		return err
	}
	className := strings.TrimSpace(req.ClassName)
	if className == "" {
		className = "traefik"
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		path = "/"
	}
	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &className,
			Rules: []networkingv1.IngressRule{{
				Host: ingressRuleHost(className, req.Host),
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     path,
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
								Name: req.ServiceName,
								Port: networkingv1.ServiceBackendPort{Number: req.ServicePort},
							}},
						}},
					},
				},
			}},
		},
	}
	if isTailscaleIngress(className) && strings.TrimSpace(req.Host) != "" {
		ing.Spec.TLS = []networkingv1.IngressTLS{{Hosts: []string{req.Host}}}
	} else if req.TLSSecretName != "" {
		ing.Spec.TLS = []networkingv1.IngressTLS{{Hosts: []string{req.Host}, SecretName: req.TLSSecretName}}
	}
	current, err := m.Clientset.NetworkingV1().Ingresses(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = m.Clientset.NetworkingV1().Ingresses(req.Namespace).Create(ctx, ing, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	current.Labels = req.Labels
	current.Annotations = req.Annotations
	current.Spec = ing.Spec
	_, err = m.Clientset.NetworkingV1().Ingresses(req.Namespace).Update(ctx, current, metav1.UpdateOptions{})
	return err
}

func (m *Manager) DeleteIngress(ctx context.Context, namespace, name string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	err := m.Clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (m *Manager) UpsertNetworkPolicy(ctx context.Context, req dto.UpsertNetworkPolicyRequest) error {
	if err := m.ensure(); err != nil {
		return err
	}
	policyTypes := make([]networkingv1.PolicyType, 0, len(req.PolicyTypes))
	for _, item := range req.PolicyTypes {
		policyTypes = append(policyTypes, networkingv1.PolicyType(item))
	}
	if len(policyTypes) == 0 {
		policyTypes = []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}
	}
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace, Labels: req.Labels},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: req.PodSelector},
			PolicyTypes: policyTypes,
		},
	}
	if req.AllowSameNamespace {
		policy.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{{From: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{}}}}}
	}
	if req.AllowDNS {
		policy.Spec.Egress = append(policy.Spec.Egress, networkingv1.NetworkPolicyEgressRule{Ports: []networkingv1.NetworkPolicyPort{
			{Protocol: protocolPtr(corev1.ProtocolUDP), Port: intstrPtr(53)},
			{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intstrPtr(53)},
		}})
		hasEgress := false
		for _, item := range policy.Spec.PolicyTypes {
			hasEgress = hasEgress || item == networkingv1.PolicyTypeEgress
		}
		if !hasEgress {
			policy.Spec.PolicyTypes = append(policy.Spec.PolicyTypes, networkingv1.PolicyTypeEgress)
		}
	}
	current, err := m.Clientset.NetworkingV1().NetworkPolicies(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = m.Clientset.NetworkingV1().NetworkPolicies(req.Namespace).Create(ctx, policy, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	current.Labels = req.Labels
	current.Spec = policy.Spec
	_, err = m.Clientset.NetworkingV1().NetworkPolicies(req.Namespace).Update(ctx, current, metav1.UpdateOptions{})
	return err
}

func (m *Manager) DeleteNetworkPolicy(ctx context.Context, namespace, name string) error {
	if err := m.ensure(); err != nil {
		return err
	}
	err := m.Clientset.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func protocolPtr(protocol corev1.Protocol) *corev1.Protocol {
	return &protocol
}

func intstrPtr(port int) *intstr.IntOrString {
	value := intstr.FromInt(port)
	return &value
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
			NodePort:   p.NodePort,
			Protocol:   protocol,
		})
	}
	return out
}
