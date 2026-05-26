package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

const apiKeyMarker = "bcs"

const (
	ScopeLegacyAPI         = "beancs.api"
	ScopeAdmin             = "beancs.admin"
	ScopeProjectsRead      = "projects:read"
	ScopeProjectsWrite     = "projects:write"
	ScopeProjectsDelete    = "projects:delete"
	ScopeProjectsDeploy    = "projects:deploy"
	ScopeDeploymentsRead   = "deployments:read"
	ScopeDeploymentsWrite  = "deployments:write"
	ScopeProcessesRead     = "processes:read"
	ScopeCredentialsRead   = "credentials:read"
	ScopeCredentialsWrite  = "credentials:write"
	ScopeCredentialsDelete = "credentials:delete"
	ScopeRegistriesRead    = "registries:read"
	ScopeRegistriesWrite   = "registries:write"
	ScopeRegistriesDelete  = "registries:delete"
	ScopeRuntimeRead       = "runtime:read"
	ScopeRuntimeWrite      = "runtime:write"
	ScopeAPIKeysRead       = "api-keys:read"
	ScopeAPIKeysWrite      = "api-keys:write"
	ScopeAPIKeysRevoke     = "api-keys:revoke"
)

const (
	APIKeyPresetProjectDeveloper = "project-developer"
	APIKeyPresetProjectOperator  = "project-operator"
	APIKeyPresetReadOnly         = "read-only"
	APIKeyPresetAdmin            = "admin"
)

type APIKeyService struct {
	db *gorm.DB
}

type APIKeyIdentity struct {
	UserID   string
	TenantID string
	Scopes   []string
	KeyID    uint
	KeyName  string
}

func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

func (s *APIKeyService) Create(ctx context.Context, userID, tenantID string, currentScopes []string, restrictToCurrent bool, req dto.CreateAPIKeyRequest) (*dto.CreateAPIKeyResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	scopes, err := allowedAPIKeyScopes(req.Scopes, req.Preset, currentScopes, restrictToCurrent)
	if err != nil {
		return nil, err
	}
	expiresAt, err := parseAPIKeyExpiry(req.ExpiresAt)
	if err != nil {
		return nil, err
	}
	prefix, err := randomURLToken(6)
	if err != nil {
		return nil, err
	}
	secret, err := randomURLToken(32)
	if err != nil {
		return nil, err
	}
	plain := apiKeyMarker + "_" + prefix + "_" + secret
	key := model.APIKey{
		UserID:    userID,
		TenantID:  tenantID,
		Name:      name,
		Prefix:    prefix,
		Hash:      apiKeyHash(plain),
		Scopes:    strings.Join(scopes, " "),
		ExpiresAt: expiresAt,
	}
	if err := s.db.WithContext(ctx).Create(&key).Error; err != nil {
		return nil, err
	}
	out := apiKeyResponse(key)
	return &dto.CreateAPIKeyResponse{APIKeyResponse: out, Key: plain}, nil
}

func (s *APIKeyService) ScopeOptions(currentScopes []string) APIKeyScopeCatalog {
	return apiKeyScopeCatalog(currentScopes)
}

func (s *APIKeyService) List(ctx context.Context, userID string) ([]dto.APIKeyResponse, error) {
	var keys []model.APIKey
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&keys).Error; err != nil {
		return nil, err
	}
	out := make([]dto.APIKeyResponse, 0, len(keys))
	for _, key := range keys {
		out = append(out, apiKeyResponse(key))
	}
	return out, nil
}

func (s *APIKeyService) Revoke(ctx context.Context, userID string, id uint) error {
	now := time.Now().UTC()
	tx := s.db.WithContext(ctx).Model(&model.APIKey{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", id, userID).
		Update("revoked_at", &now)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

func (s *APIKeyService) Authenticate(ctx context.Context, plain string) (*APIKeyIdentity, bool, error) {
	prefix, ok := apiKeyPrefix(plain)
	if !ok {
		return nil, false, nil
	}
	var key model.APIKey
	if err := s.db.WithContext(ctx).Where("prefix = ? AND revoked_at IS NULL", prefix).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	if key.ExpiresAt != nil && !key.ExpiresAt.After(time.Now().UTC()) {
		return nil, false, nil
	}
	if subtle.ConstantTimeCompare([]byte(apiKeyHash(plain)), []byte(key.Hash)) != 1 {
		return nil, false, nil
	}
	now := time.Now().UTC()
	_ = s.db.WithContext(ctx).Model(&key).Update("last_used_at", &now).Error
	return &APIKeyIdentity{
		UserID:   key.UserID,
		TenantID: key.TenantID,
		Scopes:   scopesFromString(key.Scopes),
		KeyID:    key.ID,
		KeyName:  key.Name,
	}, true, nil
}

func allowedAPIKeyScopes(requested []string, preset string, current []string, restrictToCurrent bool) ([]string, error) {
	currentSet := map[string]bool{}
	for _, scope := range current {
		currentSet[strings.TrimSpace(scope)] = true
	}
	if strings.TrimSpace(preset) != "" {
		presetScopes, ok := apiKeyPresetScopes(strings.TrimSpace(preset))
		if !ok {
			return nil, fmt.Errorf("unknown api key preset %q", preset)
		}
		requested = append(append([]string{}, presetScopes...), requested...)
	}
	out := []string{}
	if len(requested) == 0 {
		out = append(out, apiKeyPresetScopesMust(APIKeyPresetProjectDeveloper)...)
		return out, nil
	}
	seen := map[string]bool{}
	for _, scope := range requested {
		scope = strings.TrimSpace(scope)
		if scope == "" || seen[scope] {
			continue
		}
		if !knownAPIKeyScope(scope) {
			return nil, fmt.Errorf("unknown api key scope %q", scope)
		}
		if scope == ScopeAdmin && !currentSet[ScopeAdmin] {
			return nil, fmt.Errorf("beancs.admin scope requires an admin session")
		}
		if restrictToCurrent && !apiKeyScopeGranted(current, scope) {
			return nil, fmt.Errorf("scope %q exceeds the current api key permissions", scope)
		}
		seen[scope] = true
		out = append(out, scope)
	}
	if len(out) == 0 {
		out = apiKeyPresetScopesMust(APIKeyPresetProjectDeveloper)
	}
	return out, nil
}

func apiKeyScopeGranted(granted []string, required string) bool {
	for _, scope := range granted {
		scope = strings.TrimSpace(scope)
		if scope == required || scope == ScopeAdmin {
			return true
		}
		if scope == ScopeLegacyAPI && required != ScopeAdmin {
			return true
		}
		if strings.HasSuffix(scope, ":*") && strings.HasPrefix(required, strings.TrimSuffix(scope, "*")) {
			return true
		}
	}
	return false
}

type APIKeyScopeCatalog struct {
	Scopes  []dto.APIKeyScopeOption `json:"scopes"`
	Presets []dto.APIKeyScopePreset `json:"presets"`
}

func apiKeyScopeCatalog(currentScopes []string) APIKeyScopeCatalog {
	admin := hasScope(currentScopes, ScopeAdmin)
	scopes := []dto.APIKeyScopeOption{
		{ID: ScopeProjectsRead, Label: "Read projects", Description: "List projects, inspect project settings, DNS, env keys, and release history."},
		{ID: ScopeProjectsWrite, Label: "Write projects", Description: "Create and update projects and project environment variables."},
		{ID: ScopeProjectsDelete, Label: "Delete projects", Description: "Delete owned projects and their managed resources."},
		{ID: ScopeProjectsDeploy, Label: "Deploy projects", Description: "Start rebuilds, rollbacks, restarts, and project deploy actions."},
		{ID: ScopeDeploymentsRead, Label: "Read deployments", Description: "List deployment records, logs, tracking, and release history."},
		{ID: ScopeDeploymentsWrite, Label: "Write deployments", Description: "Create deployment records and rollback requests."},
		{ID: ScopeProcessesRead, Label: "Read processes", Description: "Inspect deployment process records and job logs."},
		{ID: ScopeCredentialsRead, Label: "Read credentials", Description: "List and verify GitHub, Cloudflare, and BasaltPass credentials."},
		{ID: ScopeCredentialsWrite, Label: "Write credentials", Description: "Create and update credentials and DNS records."},
		{ID: ScopeCredentialsDelete, Label: "Delete credentials", Description: "Delete credentials and revoke credential shares."},
		{ID: ScopeRegistriesRead, Label: "Read registries", Description: "List container registries, tracked images, and live tags."},
		{ID: ScopeRegistriesWrite, Label: "Write registries", Description: "Create registries, track images, and refresh tracked tags."},
		{ID: ScopeRegistriesDelete, Label: "Delete registries", Description: "Delete registries and tracked images."},
		{ID: ScopeRuntimeRead, Label: "Read runtime", Description: "Inspect Kubernetes runtime, logs, events, namespaces, pods, and services."},
		{ID: ScopeRuntimeWrite, Label: "Write runtime", Description: "Change runtime objects such as services, ingresses, namespace settings, and scaling."},
		{ID: ScopeAPIKeysRead, Label: "Read API keys", Description: "List API keys for the same user."},
		{ID: ScopeAPIKeysWrite, Label: "Write API keys", Description: "Create API keys for the same user."},
		{ID: ScopeAPIKeysRevoke, Label: "Revoke API keys", Description: "Revoke API keys for the same user."},
		{ID: ScopeLegacyAPI, Label: "Legacy API", Description: "Compatibility scope for existing BeanCS API keys."},
	}
	if admin {
		scopes = append(scopes, dto.APIKeyScopeOption{ID: ScopeAdmin, Label: "Admin", Description: "Administrative access across tenants and cluster admin APIs."})
	}
	presets := []dto.APIKeyScopePreset{
		{ID: APIKeyPresetProjectDeveloper, Label: "Project developer", Description: "Create projects, trigger builds, and inspect release history.", Scopes: apiKeyPresetScopesMust(APIKeyPresetProjectDeveloper)},
		{ID: APIKeyPresetProjectOperator, Label: "Project operator", Description: "Manage projects, credentials, registries, deployments, and runtime operations.", Scopes: apiKeyPresetScopesMust(APIKeyPresetProjectOperator)},
		{ID: APIKeyPresetReadOnly, Label: "Read only", Description: "Inspect projects, deployments, credentials, registries, and runtime state.", Scopes: apiKeyPresetScopesMust(APIKeyPresetReadOnly)},
	}
	if admin {
		presets = append(presets, dto.APIKeyScopePreset{ID: APIKeyPresetAdmin, Label: "Admin", Description: "Full BeanCS admin API access.", Scopes: apiKeyPresetScopesMust(APIKeyPresetAdmin)})
	}
	return APIKeyScopeCatalog{Scopes: scopes, Presets: presets}
}

func apiKeyPresetScopes(preset string) ([]string, bool) {
	switch preset {
	case APIKeyPresetProjectDeveloper:
		return []string{ScopeProjectsRead, ScopeProjectsWrite, ScopeProjectsDeploy, ScopeDeploymentsRead, ScopeDeploymentsWrite, ScopeProcessesRead, ScopeCredentialsRead, ScopeRegistriesRead, ScopeRuntimeRead}, true
	case APIKeyPresetProjectOperator:
		return []string{ScopeProjectsRead, ScopeProjectsWrite, ScopeProjectsDelete, ScopeProjectsDeploy, ScopeDeploymentsRead, ScopeDeploymentsWrite, ScopeProcessesRead, ScopeCredentialsRead, ScopeCredentialsWrite, ScopeCredentialsDelete, ScopeRegistriesRead, ScopeRegistriesWrite, ScopeRegistriesDelete, ScopeRuntimeRead, ScopeRuntimeWrite}, true
	case APIKeyPresetReadOnly:
		return []string{ScopeProjectsRead, ScopeDeploymentsRead, ScopeProcessesRead, ScopeCredentialsRead, ScopeRegistriesRead, ScopeRuntimeRead}, true
	case APIKeyPresetAdmin:
		return []string{ScopeAdmin}, true
	default:
		return nil, false
	}
}

func apiKeyPresetScopesMust(preset string) []string {
	scopes, _ := apiKeyPresetScopes(preset)
	return scopes
}

func knownAPIKeyScope(scope string) bool {
	switch scope {
	case ScopeLegacyAPI, ScopeAdmin, ScopeProjectsRead, ScopeProjectsWrite, ScopeProjectsDelete, ScopeProjectsDeploy, ScopeDeploymentsRead, ScopeDeploymentsWrite, ScopeProcessesRead, ScopeCredentialsRead, ScopeCredentialsWrite, ScopeCredentialsDelete, ScopeRegistriesRead, ScopeRegistriesWrite, ScopeRegistriesDelete, ScopeRuntimeRead, ScopeRuntimeWrite, ScopeAPIKeysRead, ScopeAPIKeysWrite, ScopeAPIKeysRevoke:
		return true
	default:
		return false
	}
}

func hasScope(scopes []string, scope string) bool {
	for _, s := range scopes {
		if strings.TrimSpace(s) == scope {
			return true
		}
	}
	return false
}

func parseAPIKeyExpiry(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("expires_at must be RFC3339")
	}
	if !parsed.After(time.Now().UTC()) {
		return nil, fmt.Errorf("expires_at must be in the future")
	}
	return &parsed, nil
}

func apiKeyPrefix(plain string) (string, bool) {
	parts := strings.Split(plain, "_")
	if len(parts) != 3 || parts[0] != apiKeyMarker || parts[1] == "" || parts[2] == "" {
		return "", false
	}
	return parts[1], true
}

func apiKeyHash(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func randomURLToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func apiKeyResponse(key model.APIKey) dto.APIKeyResponse {
	return dto.APIKeyResponse{
		ID:         key.ID,
		Name:       key.Name,
		Prefix:     key.Prefix,
		Scopes:     scopesFromString(key.Scopes),
		LastUsedAt: key.LastUsedAt,
		ExpiresAt:  key.ExpiresAt,
		RevokedAt:  key.RevokedAt,
		CreatedAt:  key.CreatedAt,
	}
}

func scopesFromString(value string) []string {
	return strings.Fields(value)
}
