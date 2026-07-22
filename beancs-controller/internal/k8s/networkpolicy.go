package k8s

import (
	"context"
	"encoding/json"

	"github.com/zeturn/beancs-controller/internal/model"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const certManagerHTTP01SolverPort int32 = 8089

func (m *Manager) ApplyNetworkPolicies(ctx context.Context, namespace, projectName, exposureMode string) error {
	return m.ApplyNetworkPoliciesForPorts(ctx, namespace, projectName, model.ProjectPorts{{Name: "http", Port: 8080, Exposure: exposureMode}})
}

func (m *Manager) ApplyNetworkPoliciesForPorts(ctx context.Context, namespace, projectName string, ports model.ProjectPorts) error {
	if err := m.ensure(); err != nil {
		return err
	}
	defaultDeny := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default-deny-all", Namespace: namespace, Labels: Labels(projectName)},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
		},
	}
	if err := m.upsertNetworkPolicy(ctx, defaultDeny); err != nil {
		return err
	}

	allow := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "beancs-allow-ingress", Namespace: namespace, Labels: Labels(projectName)},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{From: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{}}}},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{{}},
		},
	}
	publicPorts := policyPortsForExposure(ports, model.ExposurePublic)
	if len(publicPorts) > 0 {
		for _, ns := range m.PublicIngressNamespaces {
			allow.Spec.Ingress = append(allow.Spec.Ingress, namespaceIngressPorts(ns, publicPorts))
		}
		allow.Spec.Ingress = append(allow.Spec.Ingress, publicIngressPorts(publicPorts))
		if err := m.upsertNetworkPolicy(ctx, acmeHTTP01SolverNetworkPolicy(namespace, projectName, m.PublicIngressNamespaces)); err != nil {
			return err
		}
	}
	privatePorts := policyPortsForExposure(ports, model.ExposurePrivate)
	if len(privatePorts) > 0 {
		for _, ns := range m.PrivateIngressNamespaces {
			allow.Spec.Ingress = append(allow.Spec.Ingress, namespaceIngressPorts(ns, privatePorts))
		}
	}
	return m.upsertNetworkPolicy(ctx, allow)
}

func acmeHTTP01SolverNetworkPolicy(namespace, projectName string, ingressNamespaces []string) *networkingv1.NetworkPolicy {
	peers := []networkingv1.NetworkPolicyPeer{{IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0"}}}
	for _, ns := range ingressNamespaces {
		peers = append(peers, networkingv1.NetworkPolicyPeer{
			NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": ns}},
		})
	}
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-acme-http-solver", Namespace: namespace, Labels: Labels(projectName)},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"acme.cert-manager.io/http01-solver": "true"}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: peers,
				Ports: []networkingv1.NetworkPolicyPort{{
					Protocol: ptr(corev1.ProtocolTCP),
					Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: certManagerHTTP01SolverPort},
				}},
			}},
		},
	}
}

func namespaceIngress(ns string) networkingv1.NetworkPolicyIngressRule {
	return namespaceIngressPorts(ns, nil)
}

func namespaceIngressPorts(ns string, ports []networkingv1.NetworkPolicyPort) networkingv1.NetworkPolicyIngressRule {
	return networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{{
			NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": ns}},
		}},
		Ports: ports,
	}
}

func publicIngressPorts(ports []networkingv1.NetworkPolicyPort) networkingv1.NetworkPolicyIngressRule {
	return networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{{
			IPBlock: &networkingv1.IPBlock{CIDR: "0.0.0.0/0"},
		}},
		Ports: ports,
	}
}

func policyPortsForExposure(ports model.ProjectPorts, exposure string) []networkingv1.NetworkPolicyPort {
	out := []networkingv1.NetworkPolicyPort{}
	for _, p := range ports {
		if p.Exposure != exposure {
			continue
		}
		out = append(out, networkingv1.NetworkPolicyPort{Port: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(p.Port)}})
	}
	return out
}

func ptr[T any](value T) *T {
	return &value
}

func (m *Manager) upsertNetworkPolicy(ctx context.Context, np *networkingv1.NetworkPolicy) error {
	_, err := m.Clientset.NetworkingV1().NetworkPolicies(np.Namespace).Create(ctx, np, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		current, getErr := m.Clientset.NetworkingV1().NetworkPolicies(np.Namespace).Get(ctx, np.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		if np.Name == "beancs-allow-ingress" {
			np.Spec.Ingress = mergeNetworkPolicyIngress(current.Spec.Ingress, np.Spec.Ingress)
		}
		current.Spec = np.Spec
		current.Labels = np.Labels
		_, err = m.Clientset.NetworkingV1().NetworkPolicies(np.Namespace).Update(ctx, current, metav1.UpdateOptions{})
	}
	return err
}

func mergeNetworkPolicyIngress(existing, desired []networkingv1.NetworkPolicyIngressRule) []networkingv1.NetworkPolicyIngressRule {
	out := make([]networkingv1.NetworkPolicyIngressRule, 0, len(existing)+len(desired))
	seen := map[string]bool{}
	for _, rule := range append(existing, desired...) {
		keyBytes, err := json.Marshal(rule)
		key := string(keyBytes)
		if err != nil {
			key = rule.String()
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, rule)
	}
	return out
}
