package service

import "testing"

func TestAllowedAPIKeyScopesPresetAndCustom(t *testing.T) {
	scopes, err := allowedAPIKeyScopes([]string{ScopeAPIKeysRead}, APIKeyPresetProjectDeveloper, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !containsScope(scopes, ScopeProjectsWrite) || !containsScope(scopes, ScopeProjectsDeploy) || !containsScope(scopes, ScopeAPIKeysRead) {
		t.Fatalf("expected preset and custom scopes, got %#v", scopes)
	}
}

func TestAllowedAPIKeyScopesRejectsUnknown(t *testing.T) {
	if _, err := allowedAPIKeyScopes([]string{"projects:root"}, "", nil, false); err == nil {
		t.Fatal("expected unknown scope error")
	}
}

func TestAllowedAPIKeyScopesAdminRequiresSession(t *testing.T) {
	if _, err := allowedAPIKeyScopes([]string{ScopeAdmin}, "", nil, false); err == nil {
		t.Fatal("expected admin scope to require admin session")
	}
	if _, err := allowedAPIKeyScopes([]string{ScopeAdmin}, "", []string{ScopeAdmin}, false); err != nil {
		t.Fatal(err)
	}
}

func TestAllowedAPIKeyScopesRestrictsAPIKeyPrivilegeEscalation(t *testing.T) {
	if _, err := allowedAPIKeyScopes([]string{ScopeProjectsDelete}, "", []string{ScopeAPIKeysWrite, ScopeProjectsRead}, true); err == nil {
		t.Fatal("expected privilege escalation error")
	}
	if _, err := allowedAPIKeyScopes([]string{ScopeProjectsRead}, "", []string{ScopeAPIKeysWrite, ScopeProjectsRead}, true); err != nil {
		t.Fatal(err)
	}
}

func containsScope(scopes []string, want string) bool {
	for _, scope := range scopes {
		if scope == want {
			return true
		}
	}
	return false
}
