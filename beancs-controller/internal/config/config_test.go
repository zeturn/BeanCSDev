package config

import "testing"

func TestWebhookBaseURLAddsHTTPSForBareWebhookHost(t *testing.T) {
	cfg := &Config{SelfWebhookHost: "beancs.example.com"}
	if got := cfg.WebhookBaseURL(); got != "https://beancs.example.com" {
		t.Fatalf("WebhookBaseURL() = %q", got)
	}
}

func TestWebhookBaseURLPreservesExplicitScheme(t *testing.T) {
	cfg := &Config{SelfWebhookHost: "http://beancs.example.com/"}
	if got := cfg.WebhookBaseURL(); got != "http://beancs.example.com" {
		t.Fatalf("WebhookBaseURL() = %q", got)
	}
}

func TestWebhookBaseURLFallsBackToPublicHost(t *testing.T) {
	cfg := &Config{SelfPublicHost: "beancs.example.com"}
	if got := cfg.WebhookBaseURL(); got != "https://beancs.example.com" {
		t.Fatalf("WebhookBaseURL() = %q", got)
	}
}

func TestPublicURLNormalizesConfiguredExternalService(t *testing.T) {
	if got := publicURL("argocd.example.com/"); got != "https://argocd.example.com" {
		t.Fatalf("publicURL() = %q", got)
	}
}
