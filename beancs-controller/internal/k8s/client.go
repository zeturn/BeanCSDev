package k8s

import (
	"fmt"
	"strings"

	"github.com/zeturn/beancs-controller/internal/config"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Manager struct {
	Clientset                kubernetes.Interface
	Dynamic                  dynamic.Interface
	PublicIngressNamespaces  []string
	PrivateIngressNamespaces []string
	CertManager              CertManagerOptions
	ControllerNamespace      string
	ClusterName              string
	K3sServerURL             string
	K3sJoinToken             string
	RegistryHost             string
	RegistryPullUsername     string
	RegistryPullToken        string
	RegistryPullSecret       string
	err                      error
}

type CertManagerOptions struct {
	IssuerName             string
	ACMEServer             string
	Email                  string
	CloudflareSecretName   string
	CloudflareSecretKey    string
	PrivateKeySecretSuffix string
}

func NewManager(cfg *config.Config) *Manager {
	var restCfg *rest.Config
	var err error
	if cfg.KubeConfig != "" {
		restCfg, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfig)
	} else {
		restCfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return &Manager{err: fmt.Errorf("kubernetes config unavailable: %w", err)}
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return &Manager{err: err}
	}
	dyn, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return &Manager{err: err}
	}
	return &Manager{
		Clientset:                clientset,
		Dynamic:                  dyn,
		ControllerNamespace:      strings.TrimSpace(cfg.ControllerNamespace),
		ClusterName:              cfg.ClusterName,
		K3sServerURL:             strings.TrimSpace(cfg.K3sServerURL),
		K3sJoinToken:             strings.TrimSpace(cfg.K3sJoinToken),
		RegistryHost:             strings.TrimPrefix(strings.TrimPrefix(strings.TrimRight(strings.TrimSpace(cfg.RegistryHost), "/"), "https://"), "http://"),
		RegistryPullUsername:     strings.TrimSpace(cfg.RegistryPullUsername),
		RegistryPullToken:        strings.TrimSpace(cfg.RegistryPullToken),
		RegistryPullSecret:       strings.TrimSpace(cfg.RegistryPullSecret),
		PublicIngressNamespaces:  splitNamespaces(cfg.PublicIngressNamespaces, []string{"kube-system", "traefik"}),
		PrivateIngressNamespaces: splitNamespaces(cfg.PrivateIngressNamespaces, []string{"tailscale", "tailscale-system"}),
		CertManager: CertManagerOptions{
			IssuerName:             cfg.CertManagerIssuerName,
			ACMEServer:             cfg.CertManagerACMEServer,
			Email:                  cfg.CertManagerEmail,
			CloudflareSecretName:   cfg.CertManagerCloudflareSecretName,
			CloudflareSecretKey:    cfg.CertManagerCloudflareSecretKey,
			PrivateKeySecretSuffix: cfg.CertManagerPrivateKeySecretSuffix,
		},
	}
}

func (m *Manager) ensure() error {
	if m == nil {
		return fmt.Errorf("kubernetes manager not configured")
	}
	if m.err != nil {
		return m.err
	}
	if m.Clientset == nil {
		return fmt.Errorf("kubernetes client not configured")
	}
	return nil
}

func Labels(projectName string) map[string]string {
	return map[string]string{"app": projectName, "managed-by": "beancs"}
}

func splitNamespaces(raw string, fallback []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, part := range strings.Split(raw, ",") {
		ns := strings.TrimSpace(part)
		if ns == "" || seen[ns] {
			continue
		}
		seen[ns] = true
		out = append(out, ns)
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
