package config

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port    string `env:"PORT" envDefault:"8080"`
	Version string `env:"VERSION" envDefault:"dev"`

	DatabaseURL string `env:"DATABASE_URL,required"`

	BPMgmtBaseURL      string `env:"BP_MGMT_BASE_URL,required"`
	BPMgmtClientID     string `env:"BP_MGMT_CLIENT_ID,required"`
	BPMgmtClientSecret string `env:"BP_MGMT_CLIENT_SECRET,required"`
	BPBrowserAuthURL   string `env:"BP_BROWSER_AUTH_URL"`
	BPBrowserClientID  string `env:"BP_BROWSER_CLIENT_ID"`
	BPBrowserSecret    string `env:"BP_BROWSER_CLIENT_SECRET"`

	WebhookSecret string `env:"WEBHOOK_SECRET,required"`
	EncryptionKey string `env:"ENCRYPTION_KEY,required"`
	CORSOrigins   string `env:"CORS_ORIGINS" envDefault:"*"`

	KubeConfig string `env:"KUBECONFIG"`
	IngressIP  string `env:"INGRESS_IP" envDefault:"127.0.0.1"`

	ControllerNamespace string `env:"POD_NAMESPACE" envDefault:"beancs-system"`
	ControllerName      string `env:"BEANCS_CONTROLLER_NAME" envDefault:"beancs-controller"`
	SelfManageIngress   bool   `env:"BEANCS_SELF_MANAGE_INGRESS" envDefault:"true"`
	SelfPublicHost      string `env:"BEANCS_PUBLIC_HOST"`
	SelfTailscaleHost   string `env:"BEANCS_TAILSCALE_HOST" envDefault:"beancs-controller"`
	SelfWebhookHost     string `env:"BEANCS_WEBHOOK_HOST"`
	SelfServicePort     int    `env:"BEANCS_SELF_SERVICE_PORT" envDefault:"8080"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	key, err := hex.DecodeString(cfg.EncryptionKey)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be 64 hex characters")
	}
	if cfg.BPBrowserAuthURL == "" {
		cfg.BPBrowserAuthURL = cfg.BPMgmtBaseURL
	}
	cfg.BPBrowserAuthURL = browserAuthBase(cfg.BPBrowserAuthURL)
	if cfg.BPBrowserClientID == "" {
		cfg.BPBrowserClientID = cfg.BPMgmtClientID
	}
	return &cfg, nil
}

func browserAuthBase(v string) string {
	v = strings.TrimRight(strings.TrimSpace(v), "/")
	return strings.TrimSuffix(v, "/api/v1")
}
