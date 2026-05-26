package middleware

import (
	"testing"

	"github.com/zeturn/beancs-controller/internal/service"
)

func TestScopeMatchesLegacyAPI(t *testing.T) {
	if !scopeMatches(service.ScopeLegacyAPI, service.ScopeProjectsWrite) {
		t.Fatal("expected legacy api scope to satisfy non-admin scope")
	}
	if scopeMatches(service.ScopeLegacyAPI, service.ScopeAdmin) {
		t.Fatal("legacy api scope must not satisfy admin")
	}
}

func TestScopeMatchesWildcard(t *testing.T) {
	if !scopeMatches("projects:*", service.ScopeProjectsDeploy) {
		t.Fatal("expected wildcard scope to match same namespace")
	}
	if scopeMatches("projects:*", service.ScopeRuntimeRead) {
		t.Fatal("wildcard scope matched different namespace")
	}
}
