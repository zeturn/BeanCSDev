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

func (s *APIKeyService) Create(ctx context.Context, userID, tenantID string, currentScopes []string, req dto.CreateAPIKeyRequest) (*dto.CreateAPIKeyResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	scopes, err := allowedAPIKeyScopes(req.Scopes, currentScopes)
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

func allowedAPIKeyScopes(requested []string, current []string) ([]string, error) {
	currentSet := map[string]bool{}
	for _, scope := range current {
		currentSet[strings.TrimSpace(scope)] = true
	}
	out := []string{}
	if len(requested) == 0 {
		out = append(out, "beancs.api")
		if currentSet["beancs.admin"] {
			out = append(out, "beancs.admin")
		}
		return out, nil
	}
	seen := map[string]bool{}
	for _, scope := range requested {
		scope = strings.TrimSpace(scope)
		if scope == "" || seen[scope] {
			continue
		}
		if scope == "beancs.admin" && !currentSet["beancs.admin"] {
			return nil, fmt.Errorf("beancs.admin scope requires an admin session")
		}
		seen[scope] = true
		out = append(out, scope)
	}
	if len(out) == 0 {
		out = []string{"beancs.api"}
	}
	return out, nil
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
