package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	cryptoutil "github.com/zeturn/beancs-controller/internal/crypto"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CredentialService struct {
	db     *gorm.DB
	cipher cryptoutil.Cipher
}

func NewCredentialService(db *gorm.DB, cipher cryptoutil.Cipher) *CredentialService {
	return &CredentialService{db: db, cipher: cipher}
}

func (s *CredentialService) HasAccess(userID, typ string, id uint, ownerOnly bool) (bool, error) {
	var uc model.UserCredential
	err := s.db.Where("user_id = ? AND credential_type = ? AND credential_id = ?", userID, typ, id).First(&uc).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return !ownerOnly || uc.Role == model.CredentialRoleOwner, nil
}

func (s *CredentialService) RequireAccess(userID, typ string, id uint, ownerOnly bool) error {
	ok, err := s.HasAccess(userID, typ, id, ownerOnly)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("credential access denied")
	}
	return nil
}

func (s *CredentialService) CreateCloudflare(ctx context.Context, userID string, req dto.CreateCloudflareCredentialRequest) (*model.CloudflareCredential, error) {
	enc, err := s.cipher.EncryptString(req.APIToken)
	if err != nil {
		return nil, err
	}
	cred := &model.CloudflareCredential{Name: req.Name, APITokenEnc: enc, ZoneID: req.ZoneID, Domain: req.Domain, AccountID: req.AccountID, IsActive: true, CreatedBy: userID}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cred).Error; err != nil {
			return err
		}
		return tx.Create(&model.UserCredential{UserID: userID, CredentialType: model.CredentialTypeCloudflare, CredentialID: cred.ID, Role: model.CredentialRoleOwner}).Error
	})
	return cred, err
}

func (s *CredentialService) CreateGitHub(ctx context.Context, userID string, req dto.CreateGitHubCredentialRequest) (*model.GitHubCredential, error) {
	enc, err := s.cipher.EncryptString(req.Token)
	if err != nil {
		return nil, err
	}
	cred := &model.GitHubCredential{Name: req.Name, TokenEnc: enc, Org: req.Org, GitOpsRepo: req.GitOpsRepo, IsActive: true, CreatedBy: userID}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cred).Error; err != nil {
			return err
		}
		return tx.Create(&model.UserCredential{UserID: userID, CredentialType: model.CredentialTypeGitHub, CredentialID: cred.ID, Role: model.CredentialRoleOwner}).Error
	})
	return cred, err
}

func (s *CredentialService) CreateBasaltPass(ctx context.Context, userID string, req dto.CreateBasaltPassCredentialRequest) (*model.BasaltPassInstance, error) {
	if err := validateExternalHTTPSURL(req.BaseURL); err != nil {
		return nil, err
	}
	enc, err := s.cipher.EncryptString(req.ClientSecret)
	if err != nil {
		return nil, err
	}
	var serviceTokenEnc []byte
	if strings.TrimSpace(req.ServiceToken) != "" {
		serviceTokenEnc, err = s.cipher.EncryptString(strings.TrimSpace(req.ServiceToken))
		if err != nil {
			return nil, err
		}
	}
	cred := &model.BasaltPassInstance{Name: req.Name, BaseURL: req.BaseURL, ClientID: req.ClientID, ClientSecretEnc: enc, ServiceTokenEnc: serviceTokenEnc, IsActive: true, CreatedBy: userID}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cred).Error; err != nil {
			return err
		}
		return tx.Create(&model.UserCredential{UserID: userID, CredentialType: model.CredentialTypeBasaltPass, CredentialID: cred.ID, Role: model.CredentialRoleOwner}).Error
	})
	return cred, err
}

func (s *CredentialService) ListCloudflare(ctx context.Context, userID string) ([]model.CloudflareCredential, error) {
	var out []model.CloudflareCredential
	err := s.db.WithContext(ctx).Joins("JOIN user_credentials uc ON uc.credential_id = cloudflare_credentials.id AND uc.credential_type = ?", model.CredentialTypeCloudflare).
		Where("uc.user_id = ?", userID).Find(&out).Error
	return out, err
}

func (s *CredentialService) ListGitHub(ctx context.Context, userID string) ([]model.GitHubCredential, error) {
	var out []model.GitHubCredential
	err := s.db.WithContext(ctx).Joins("JOIN user_credentials uc ON uc.credential_id = git_hub_credentials.id AND uc.credential_type = ?", model.CredentialTypeGitHub).
		Where("uc.user_id = ?", userID).Find(&out).Error
	return out, err
}

func (s *CredentialService) ListBasaltPass(ctx context.Context, userID string) ([]model.BasaltPassInstance, error) {
	var out []model.BasaltPassInstance
	err := s.db.WithContext(ctx).Joins("JOIN user_credentials uc ON uc.credential_id = basalt_pass_instances.id AND uc.credential_type = ?", model.CredentialTypeBasaltPass).
		Where("uc.user_id = ?", userID).Find(&out).Error
	return out, err
}

func (s *CredentialService) Share(ctx context.Context, ownerID, typ string, id uint, req dto.ShareCredentialRequest) error {
	if err := s.RequireAccess(ownerID, typ, id, true); err != nil {
		return err
	}
	role := req.Role
	if role == "" {
		role = model.CredentialRoleUser
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "credential_type"}, {Name: "credential_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role"}),
	}).Create(&model.UserCredential{UserID: req.UserID, CredentialType: typ, CredentialID: id, Role: role}).Error
}

func (s *CredentialService) Revoke(ctx context.Context, ownerID, typ string, id uint, targetUser string) error {
	if err := s.RequireAccess(ownerID, typ, id, true); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Where("user_id = ? AND credential_type = ? AND credential_id = ?", targetUser, typ, id).Delete(&model.UserCredential{}).Error
}

func (s *CredentialService) Delete(ctx context.Context, userID, typ string, id uint) error {
	if err := s.RequireAccess(userID, typ, id, true); err != nil {
		return err
	}
	var count int64
	q := s.db.WithContext(ctx).Model(&model.Project{})
	switch typ {
	case model.CredentialTypeCloudflare:
		q = q.Where("cloudflare_credential_id = ?", id)
	case model.CredentialTypeGitHub:
		q = q.Where("git_hub_credential_id = ?", id)
	case model.CredentialTypeBasaltPass:
		q = q.Where("basalt_pass_instance_id = ?", id)
	}
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("credential is still referenced by projects")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("credential_type = ? AND credential_id = ?", typ, id).Delete(&model.UserCredential{}).Error; err != nil {
			return err
		}
		switch typ {
		case model.CredentialTypeCloudflare:
			return tx.Delete(&model.CloudflareCredential{}, id).Error
		case model.CredentialTypeGitHub:
			return tx.Delete(&model.GitHubCredential{}, id).Error
		case model.CredentialTypeBasaltPass:
			return tx.Delete(&model.BasaltPassInstance{}, id).Error
		default:
			return fmt.Errorf("unknown credential type")
		}
	})
}

func (s *CredentialService) DecryptCloudflareToken(cred model.CloudflareCredential) (string, error) {
	return s.cipher.DecryptString(cred.APITokenEnc)
}

func (s *CredentialService) VerifyCloudflare(ctx context.Context, id uint) (map[string]any, error) {
	var cred model.CloudflareCredential
	if err := s.db.WithContext(ctx).First(&cred, id).Error; err != nil {
		return nil, err
	}
	token, err := s.DecryptCloudflareToken(cred)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	verifyURL := "https://api.cloudflare.com/client/v4/user/tokens/verify"
	if cred.AccountID != "" {
		verifyURL = fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/tokens/verify", url.PathEscape(cred.AccountID))
	}
	verify, err := cloudflareGET(ctx, client, token, verifyURL)
	if err != nil {
		return nil, err
	}
	zoneURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?per_page=1", url.PathEscape(cred.ZoneID))
	zone, err := cloudflareGET(ctx, client, token, zoneURL)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status":     "ok",
		"credential": cred.Name,
		"domain":     cred.Domain,
		"zone_id":    cred.ZoneID,
		"account_id": cred.AccountID,
		"token":      verify["result"],
		"zone_check": map[string]any{"success": zone["success"]},
	}, nil
}

func (s *CredentialService) DecryptGitHubToken(cred model.GitHubCredential) (string, error) {
	return s.cipher.DecryptString(cred.TokenEnc)
}

func cloudflareGET(ctx context.Context, client *http.Client, token, endpoint string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("cloudflare returned invalid JSON")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || out["success"] != true {
		return nil, fmt.Errorf("cloudflare verification failed")
	}
	return out, nil
}

func (s *CredentialService) UpdateCloudflare(ctx context.Context, id uint, req dto.UpdateCloudflareCredentialRequest) (*model.CloudflareCredential, error) {
	var cred model.CloudflareCredential
	if err := s.db.WithContext(ctx).First(&cred, id).Error; err != nil {
		return nil, err
	}
	if req.Name != nil {
		cred.Name = *req.Name
	}
	if req.APIToken != nil {
		enc, err := s.cipher.EncryptString(*req.APIToken)
		if err != nil {
			return nil, err
		}
		cred.APITokenEnc = enc
	}
	if req.ZoneID != nil {
		cred.ZoneID = *req.ZoneID
	}
	if req.Domain != nil {
		cred.Domain = *req.Domain
	}
	if req.AccountID != nil {
		cred.AccountID = *req.AccountID
	}
	if req.IsActive != nil {
		cred.IsActive = *req.IsActive
	}
	return &cred, s.db.WithContext(ctx).Save(&cred).Error
}

func (s *CredentialService) UpdateGitHub(ctx context.Context, id uint, req dto.UpdateGitHubCredentialRequest) (*model.GitHubCredential, error) {
	var cred model.GitHubCredential
	if err := s.db.WithContext(ctx).First(&cred, id).Error; err != nil {
		return nil, err
	}
	if req.Name != nil {
		cred.Name = *req.Name
	}
	if req.Token != nil {
		enc, err := s.cipher.EncryptString(*req.Token)
		if err != nil {
			return nil, err
		}
		cred.TokenEnc = enc
	}
	if req.Org != nil {
		cred.Org = *req.Org
	}
	if req.GitOpsRepo != nil {
		cred.GitOpsRepo = *req.GitOpsRepo
	}
	if req.IsActive != nil {
		cred.IsActive = *req.IsActive
	}
	return &cred, s.db.WithContext(ctx).Save(&cred).Error
}

func (s *CredentialService) UpdateBasaltPass(ctx context.Context, id uint, req dto.UpdateBasaltPassCredentialRequest) (*model.BasaltPassInstance, error) {
	var cred model.BasaltPassInstance
	if err := s.db.WithContext(ctx).First(&cred, id).Error; err != nil {
		return nil, err
	}
	if req.Name != nil {
		cred.Name = *req.Name
	}
	if req.BaseURL != nil {
		if err := validateExternalHTTPSURL(*req.BaseURL); err != nil {
			return nil, err
		}
		cred.BaseURL = *req.BaseURL
	}
	if req.ClientID != nil {
		cred.ClientID = *req.ClientID
	}
	if req.ClientSecret != nil {
		enc, err := s.cipher.EncryptString(*req.ClientSecret)
		if err != nil {
			return nil, err
		}
		cred.ClientSecretEnc = enc
	}
	if req.ServiceToken != nil {
		if strings.TrimSpace(*req.ServiceToken) == "" {
			cred.ServiceTokenEnc = nil
		} else {
			enc, err := s.cipher.EncryptString(strings.TrimSpace(*req.ServiceToken))
			if err != nil {
				return nil, err
			}
			cred.ServiceTokenEnc = enc
		}
	}
	if req.IsActive != nil {
		cred.IsActive = *req.IsActive
	}
	return &cred, s.db.WithContext(ctx).Save(&cred).Error
}

func validateExternalHTTPSURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return fmt.Errorf("base_url must be a valid https URL")
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("base_url must not target localhost")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("base_url must not target private or local addresses")
		}
		return nil
	}
	addrs, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("base_url hostname could not be resolved")
	}
	for _, ip := range addrs {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("base_url resolves to a private or local address")
		}
	}
	return nil
}
