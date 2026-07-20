package config

import (
	"encoding/hex"
	"fmt"
	"net/url"
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
	ArgoCDURL          string `env:"BEANCS_ARGOCD_URL"`

	WebhookSecret string `env:"WEBHOOK_SECRET,required"`
	EncryptionKey string `env:"ENCRYPTION_KEY,required"`
	CORSOrigins   string `env:"CORS_ORIGINS" envDefault:"*"`

	APIRateLimitPerMinute     int `env:"BEANCS_API_RATE_LIMIT_PER_MINUTE" envDefault:"600"`
	WebhookRateLimitPerMinute int `env:"BEANCS_WEBHOOK_RATE_LIMIT_PER_MINUTE" envDefault:"300"`

	KubeConfig string `env:"KUBECONFIG"`
	IngressIP  string `env:"INGRESS_IP" envDefault:"127.0.0.1"`

	PublicIngressNamespaces  string `env:"BEANCS_PUBLIC_INGRESS_NAMESPACES" envDefault:"kube-system,traefik"`
	PrivateIngressNamespaces string `env:"BEANCS_PRIVATE_INGRESS_NAMESPACES" envDefault:"tailscale,tailscale-system"`

	CertManagerIssuerName             string `env:"BEANCS_CERT_MANAGER_ISSUER_NAME" envDefault:"beancs-letsencrypt"`
	CertManagerACMEServer             string `env:"BEANCS_CERT_MANAGER_ACME_SERVER" envDefault:"https://acme-v02.api.letsencrypt.org/directory"`
	CertManagerEmail                  string `env:"BEANCS_CERT_MANAGER_EMAIL"`
	CertManagerCloudflareSecretName   string `env:"BEANCS_CERT_MANAGER_CLOUDFLARE_SECRET_NAME" envDefault:"beancs-cloudflare-dns01"`
	CertManagerCloudflareSecretKey    string `env:"BEANCS_CERT_MANAGER_CLOUDFLARE_SECRET_KEY" envDefault:"api-token"`
	CertManagerPrivateKeySecretSuffix string `env:"BEANCS_CERT_MANAGER_PRIVATE_KEY_SECRET_SUFFIX" envDefault:"account-key"`

	ControllerNamespace string `env:"POD_NAMESPACE" envDefault:"beancs-system"`
	ControllerName      string `env:"BEANCS_CONTROLLER_NAME" envDefault:"beancs-controller"`
	ClusterName         string `env:"BEANCS_CLUSTER_NAME" envDefault:"production-k3s"`
	K3sServerURL        string `env:"BEANCS_K3S_SERVER_URL"`
	K3sJoinToken        string `env:"BEANCS_K3S_JOIN_TOKEN"`
	SelfManageIngress   bool   `env:"BEANCS_SELF_MANAGE_INGRESS" envDefault:"true"`
	SelfPublicHost      string `env:"BEANCS_PUBLIC_HOST"`
	SelfTailscaleHost   string `env:"BEANCS_TAILSCALE_HOST" envDefault:"beancs-controller"`
	SelfWebhookHost     string `env:"BEANCS_WEBHOOK_HOST"`
	SelfServicePort     int    `env:"BEANCS_SELF_SERVICE_PORT" envDefault:"8080"`

	GitHubAppID         int64  `env:"BEANCS_GITHUB_APP_ID"`
	GitHubAppSlug       string `env:"BEANCS_GITHUB_APP_SLUG"`
	GitHubAppPrivateKey string `env:"BEANCS_GITHUB_APP_PRIVATE_KEY"`

	CloudflareOAuthClientID     string `env:"BEANCS_CLOUDFLARE_OAUTH_CLIENT_ID"`
	CloudflareOAuthClientSecret string `env:"BEANCS_CLOUDFLARE_OAUTH_CLIENT_SECRET"`
	CloudflareOAuthScopes       string `env:"BEANCS_CLOUDFLARE_OAUTH_SCOPES" envDefault:"zone.read dns.write"`
	CloudflareOAuthRedirectURL  string `env:"BEANCS_CLOUDFLARE_OAUTH_REDIRECT_URL" envDefault:"https://beancs.com/api/v1/credentials/cloudflare/app/callback"`

	RegistryHost         string `env:"BEANCS_REGISTRY_HOST" envDefault:"registry.beancs.hollowdata.com"`
	RegistryUsername     string `env:"BEANCS_REGISTRY_USERNAME"`
	RegistryToken        string `env:"BEANCS_REGISTRY_TOKEN"`
	RegistryPullUsername string `env:"BEANCS_REGISTRY_PULL_USERNAME"`
	RegistryPullToken    string `env:"BEANCS_REGISTRY_PULL_TOKEN"`
	RegistryPullSecret   string `env:"BEANCS_REGISTRY_PULL_SECRET" envDefault:"beancs-registry-pull"`
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
	cfg.BPBrowserAuthURL = basaltAPIBaseURL(cfg.BPBrowserAuthURL)
	if cfg.BPBrowserClientID == "" {
		cfg.BPBrowserClientID = cfg.BPMgmtClientID
	}
	cfg.ArgoCDURL = publicURL(cfg.ArgoCDURL)
	cfg.SelfWebhookHost = strings.TrimRight(strings.TrimSpace(cfg.SelfWebhookHost), "/")
	return &cfg, nil
}

func (c *Config) WebhookBaseURL() string {
	if c.SelfWebhookHost != "" {
		return publicURL(c.SelfWebhookHost)
	}
	if c.SelfPublicHost != "" {
		return publicURL(c.SelfPublicHost)
	}
	return ""
}

func publicURL(host string) string {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		return ""
	}
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host
	}
	return "https://" + strings.Trim(host, "/")
}

func basaltAPIBaseURL(v string) string {
	v = strings.TrimRight(strings.TrimSpace(v), "/")
	if v == "" {
		return v
	}
	parsed, err := url.Parse(v)
	if err != nil {
		return v + "/api/v1"
	}
	if strings.HasSuffix(strings.TrimRight(parsed.Path, "/"), "/api/v1") {
		return v
	}
	return v + "/api/v1"
}
