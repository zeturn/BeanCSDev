package k8s

import (
	"context"
	"testing"

	"github.com/zeturn/beancs-controller/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestApplyNetworkPoliciesForPublicPortsAllowsACMESolver(t *testing.T) {
	client := fake.NewSimpleClientset()
	manager := &Manager{
		Clientset:               client,
		PublicIngressNamespaces: []string{"traefik"},
	}
	err := manager.ApplyNetworkPoliciesForPorts(context.Background(), "proj-demo", "demo", model.ProjectPorts{{
		Name:     "http",
		Port:     8080,
		Exposure: model.ExposurePublic,
	}})
	if err != nil {
		t.Fatalf("ApplyNetworkPoliciesForPorts() error = %v", err)
	}

	policy, err := client.NetworkingV1().NetworkPolicies("proj-demo").Get(context.Background(), "allow-acme-http-solver", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected ACME solver policy: %v", err)
	}
	if got := policy.Spec.PodSelector.MatchLabels["acme.cert-manager.io/http01-solver"]; got != "true" {
		t.Fatalf("unexpected ACME selector: %#v", policy.Spec.PodSelector.MatchLabels)
	}
	if len(policy.Spec.Ingress) != 1 || len(policy.Spec.Ingress[0].Ports) != 1 {
		t.Fatalf("unexpected ingress rules: %#v", policy.Spec.Ingress)
	}
	if got := policy.Spec.Ingress[0].Ports[0].Port.IntVal; got != certManagerHTTP01SolverPort {
		t.Fatalf("solver port = %d, want %d", got, certManagerHTTP01SolverPort)
	}
}

func TestApplyNetworkPoliciesForInternalOnlyPortsSkipsACMESolver(t *testing.T) {
	client := fake.NewSimpleClientset()
	manager := &Manager{
		Clientset:               client,
		PublicIngressNamespaces: []string{"traefik"},
	}
	err := manager.ApplyNetworkPoliciesForPorts(context.Background(), "proj-demo", "demo", model.ProjectPorts{{
		Name:     "http",
		Port:     8080,
		Exposure: model.ExposureInternalOnly,
	}})
	if err != nil {
		t.Fatalf("ApplyNetworkPoliciesForPorts() error = %v", err)
	}

	policies, err := client.NetworkingV1().NetworkPolicies("proj-demo").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("list policies: %v", err)
	}
	for _, policy := range policies.Items {
		if policy.Name == "allow-acme-http-solver" {
			t.Fatalf("unexpected ACME solver policy for internal-only project")
		}
	}
}
