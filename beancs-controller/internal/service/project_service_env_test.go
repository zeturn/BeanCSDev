package service

import (
	"testing"

	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/model"
)

func TestBasaltPassRuntimeEnvInjectsCanonicalAndCompatibilityNames(t *testing.T) {
	inst := &model.BasaltPassInstance{
		BaseURL:    "https://auth.example.com/",
		TenantID:   "tenant-1",
		TenantCode: "acme",
	}
	project := &model.Project{
		Name:           "demo",
		Namespace:      "proj-demo",
		BasaltClientID: "client-id",
		BasaltAppID:    42,
	}

	env := basaltPassRuntimeEnv(inst, project, "client-secret", map[string]string{"APP_MODE": "prod"})

	assertEnv(t, env, "BASALTPASS_BASE_URL", "https://auth.example.com")
	assertEnv(t, env, "BASALTPASS_CLIENT_ID", "client-id")
	assertEnv(t, env, "BASALTPASS_CLIENT_SECRET", "client-secret")
	assertEnv(t, env, "BASALTPASS_REDIRECT_URI", "http://demo.proj-demo.svc.cluster.local/callback")
	assertEnv(t, env, "BASALTPASS_OAUTH_CLIENT_ID", "client-id")
	assertEnv(t, env, "BASALTPASS_OAUTH_CLIENT_SECRET", "client-secret")
	assertEnv(t, env, "BASALTPASS_OAUTH_REDIRECT_URI", "http://demo.proj-demo.svc.cluster.local/callback")
	assertEnv(t, env, "BASALT_BASE_URL", "https://auth.example.com")
	assertEnv(t, env, "BASALT_CLIENT_ID", "client-id")
	assertEnv(t, env, "BASALT_CLIENT_SECRET", "client-secret")
	assertEnv(t, env, "BASALT_REDIRECT_URI", "http://demo.proj-demo.svc.cluster.local/callback")
	assertEnv(t, env, "BASALTPASS_TENANT_ID", "tenant-1")
	assertEnv(t, env, "BASALTPASS_TENANT_CODE", "acme")
	assertEnv(t, env, "BASALTPASS_APP_ID", "42")
	assertEnv(t, env, "APP_MODE", "prod")
}

func TestBasaltPassRuntimeEnvKeepsExplicitCompatibilityAlias(t *testing.T) {
	inst := &model.BasaltPassInstance{BaseURL: "https://auth.example.com"}
	project := &model.Project{Name: "demo", Namespace: "proj-demo", BasaltClientID: "client-id"}

	env := basaltPassRuntimeEnv(inst, project, "client-secret", map[string]string{
		"BASALTPASS_OAUTH_REDIRECT_URI": "https://app.example.com/custom/callback",
	})

	assertEnv(t, env, "BASALTPASS_REDIRECT_URI", "http://demo.proj-demo.svc.cluster.local/callback")
	assertEnv(t, env, "BASALTPASS_OAUTH_REDIRECT_URI", "https://app.example.com/custom/callback")
}

func TestBasaltPassRuntimeEnvEnablesOAuthAndKeepsCanonicalRedirect(t *testing.T) {
	inst := &model.BasaltPassInstance{BaseURL: "https://auth.example.com"}
	project := &model.Project{Name: "demo", Namespace: "proj-demo", BasaltClientID: "client-id"}

	env := basaltPassRuntimeEnv(inst, project, "client-secret", map[string]string{
		"BASALTPASS_REDIRECT_URI": "https://app.example.com/api/auth/basaltpass/callback/",
	})

	assertEnv(t, env, "BASALTPASS_ENABLED", "true")
	assertEnv(t, env, "BASALTPASS_OAUTH_ENABLED", "true")
	assertEnv(t, env, "BASALTPASS_REDIRECT_URI", "https://app.example.com/api/auth/basaltpass/callback/")
	assertEnv(t, env, "BASALTPASS_OAUTH_REDIRECT_URI", "https://app.example.com/api/auth/basaltpass/callback/")
}

func TestBasaltPassComponentRuntimeBaseEnvUsesCallbackPath(t *testing.T) {
	project := &model.Project{
		Name:           "demo",
		Namespace:      "proj-demo",
		Domain:         "demo.example.com",
		BasaltClientID: "client-id",
	}

	env := basaltPassComponentRuntimeBaseEnv(project, &dto.BasaltPassComponentConfig{CallbackPath: "/api/auth/basaltpass/callback/"}, map[string]string{"APP_MODE": "prod"})

	assertEnv(t, env, "APP_MODE", "prod")
	assertEnv(t, env, "BASALTPASS_ENABLED", "true")
	assertEnv(t, env, "BASALTPASS_REDIRECT_URI", "https://demo.example.com/api/auth/basaltpass/callback/")
}

func assertEnv(t *testing.T, env map[string]string, key, want string) {
	t.Helper()
	if got := env[key]; got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}
