package k8s

import (
	"fmt"

	"github.com/zeturn/beancs-controller/internal/config"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Manager struct {
	Clientset kubernetes.Interface
	Dynamic   dynamic.Interface
	err       error
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
	return &Manager{Clientset: clientset, Dynamic: dyn}
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
