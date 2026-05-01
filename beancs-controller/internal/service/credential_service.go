package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zeturn/beancs-controller/internal/config"
	cryptoutil "github.com/zeturn/beancs-controller/internal/crypto"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CredentialService struct {
	db     *gorm.DB
	cipher cryptoutil.Cipher
	cfg    *config.Config
}

func NewCredentialService(db *gorm.DB, cipher cryptoutil.Cipher, cfg *config.Config) *CredentialService {
	return &CredentialService{db: db, cipher: cipher, cfg: cfg}
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

func (s *CredentialService) CreateCloudflare(ctx context.Context, userID string, req dto.CreateCloudflareCredentialRequest) ([]model.CloudflareCredential, error) {
	enc, err := s.cipher.EncryptString(req.APIToken)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Cloudflare"
	}
	zones, err := s.cloudflareZones(ctx, req.APIToken, req.ZoneID)
	if err != nil {
		return nil, err
	}
	if len(zones) == 0 && strings.TrimSpace(req.ZoneID) != "" && strings.TrimSpace(req.Domain) != "" {
		zones = []cloudflareZonePayload{{ID: strings.TrimSpace(req.ZoneID), Name: strings.TrimSpace(req.Domain), Account: cloudflareZoneAccount{ID: strings.TrimSpace(req.AccountID)}}}
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("no Cloudflare zones were returned for this token")
	}

	creds := make([]model.CloudflareCredential, 0, len(zones))
	for _, zone := range zones {
		if strings.TrimSpace(zone.ID) == "" || strings.TrimSpace(zone.Name) == "" {
			continue
		}
		accountID := strings.TrimSpace(req.AccountID)
		if accountID == "" {
			accountID = strings.TrimSpace(zone.Account.ID)
		}
		credName := name
		if len(zones) > 1 || !strings.Contains(strings.ToLower(name), strings.ToLower(zone.Name)) {
			credName = fmt.Sprintf("%s - %s", name, zone.Name)
		}
		creds = append(creds, model.CloudflareCredential{Name: credName, APITokenEnc: enc, ZoneID: zone.ID, Domain: zone.Name, AccountID: accountID, IsActive: true, CreatedBy: userID})
	}
	if len(creds) == 0 {
		return nil, fmt.Errorf("no usable Cloudflare zones were returned for this token")
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range creds {
			if err := tx.Create(&creds[i]).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.UserCredential{UserID: userID, CredentialType: model.CredentialTypeCloudflare, CredentialID: creds[i].ID, Role: model.CredentialRoleOwner}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return creds, err
}

func (s *CredentialService) CreateGitHub(ctx context.Context, userID string, req dto.CreateGitHubCredentialRequest) (*model.GitHubCredential, error) {
	enc, err := s.cipher.EncryptString(req.Token)
	if err != nil {
		return nil, err
	}
	accountLogin, err := s.githubUserLogin(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = accountLogin
	}
	org := strings.TrimSpace(req.Org)
	if org == "" {
		org = accountLogin
	}
	cred := &model.GitHubCredential{Name: name, AuthType: "pat", TokenEnc: enc, AccountLogin: accountLogin, Org: org, GitOpsRepo: strings.TrimSpace(req.GitOpsRepo), IsActive: true, CreatedBy: userID}
	err = s.createGitHubCredential(ctx, userID, cred)
	return cred, err
}

func (s *CredentialService) CreateGitHubApp(ctx context.Context, userID string, req dto.StartGitHubAppInstallRequest, installationID int64, accountLogin string) (*model.GitHubCredential, error) {
	accountLogin = strings.TrimSpace(accountLogin)
	name := accountLogin
	if name == "" {
		name = fmt.Sprintf("github-app-%d", installationID)
	}
	cred := &model.GitHubCredential{Name: name, AuthType: "app", InstallationID: installationID, AccountLogin: accountLogin, Org: accountLogin, GitOpsRepo: strings.TrimSpace(req.GitOpsRepo), IsActive: true, CreatedBy: userID}
	err := s.createGitHubCredential(ctx, userID, cred)
	return cred, err
}

func (s *CredentialService) createGitHubCredential(ctx context.Context, userID string, cred *model.GitHubCredential) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cred).Error; err != nil {
			return err
		}
		return tx.Create(&model.UserCredential{UserID: userID, CredentialType: model.CredentialTypeGitHub, CredentialID: cred.ID, Role: model.CredentialRoleOwner}).Error
	})
	return err
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

func (s *CredentialService) ListCloudflareDomains(ctx context.Context, userID string) ([]dto.CloudflareDomainResponse, error) {
	creds, err := s.ListCloudflare(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.CloudflareDomainResponse, 0, len(creds))
	for _, cred := range creds {
		if cred.Domain == "" || cred.ZoneID == "" {
			continue
		}
		out = append(out, dto.CloudflareDomainResponse{
			CredentialID: cred.ID,
			Credential:   cred.Name,
			ZoneID:       cred.ZoneID,
			Domain:       cred.Domain,
			AccountID:    cred.AccountID,
			Active:       cred.IsActive,
		})
	}
	return out, nil
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
	if cred.ZoneID == "" {
		zones, err := s.cloudflareZones(ctx, token, "")
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"status":     "ok",
			"credential": cred.Name,
			"account_id": cred.AccountID,
			"token":      verify["result"],
			"zones":      len(zones),
		}, nil
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

type cloudflareZoneAccount struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cloudflareZonePayload struct {
	ID      string                `json:"id"`
	Name    string                `json:"name"`
	Status  string                `json:"status"`
	Account cloudflareZoneAccount `json:"account"`
}

func (s *CredentialService) cloudflareZones(ctx context.Context, token, zoneID string) ([]cloudflareZonePayload, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	endpoint := "https://api.cloudflare.com/client/v4/zones?per_page=100"
	if strings.TrimSpace(zoneID) != "" {
		endpoint = fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s", url.PathEscape(strings.TrimSpace(zoneID)))
	}
	result, err := cloudflareGET(ctx, client, token, endpoint)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(result["result"])
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(zoneID) != "" {
		var zone cloudflareZonePayload
		if err := json.Unmarshal(raw, &zone); err != nil {
			return nil, fmt.Errorf("cloudflare returned invalid zone JSON")
		}
		return []cloudflareZonePayload{zone}, nil
	}
	var zones []cloudflareZonePayload
	if err := json.Unmarshal(raw, &zones); err != nil {
		return nil, fmt.Errorf("cloudflare returned invalid zones JSON")
	}
	return zones, nil
}

func (s *CredentialService) DecryptGitHubToken(cred model.GitHubCredential) (string, error) {
	return s.cipher.DecryptString(cred.TokenEnc)
}

func (s *CredentialService) GitHubToken(ctx context.Context, cred model.GitHubCredential) (string, error) {
	if cred.AuthType == "app" || cred.InstallationID != 0 {
		return s.githubInstallationToken(ctx, cred.InstallationID)
	}
	return s.DecryptGitHubToken(cred)
}

func (s *CredentialService) githubInstallationToken(ctx context.Context, installationID int64) (string, error) {
	if s.cfg == nil || s.cfg.GitHubAppID == 0 || strings.TrimSpace(s.cfg.GitHubAppPrivateKey) == "" {
		return "", fmt.Errorf("GitHub App is not configured")
	}
	appJWT, err := s.githubAppJWT()
	if err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GitHub App token request failed: %s", strings.TrimSpace(string(body)))
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.Token == "" {
		return "", fmt.Errorf("GitHub App token response was invalid")
	}
	return out.Token, nil
}

func (s *CredentialService) GitHubAppInstallationAccount(ctx context.Context, installationID int64) (string, error) {
	if s.cfg == nil || s.cfg.GitHubAppID == 0 || strings.TrimSpace(s.cfg.GitHubAppPrivateKey) == "" {
		return "", fmt.Errorf("GitHub App is not configured")
	}
	appJWT, err := s.githubAppJWT()
	if err != nil {
		return "", err
	}
	var out struct {
		Account struct {
			Login string `json:"login"`
		} `json:"account"`
	}
	endpoint := fmt.Sprintf("https://api.github.com/app/installations/%d", installationID)
	if err := githubJSON(ctx, http.MethodGet, endpoint, appJWT, &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Account.Login) == "" {
		return "", fmt.Errorf("GitHub installation account was not returned")
	}
	return out.Account.Login, nil
}

func (s *CredentialService) ListGitHubRepositories(ctx context.Context, id uint) ([]dto.GitHubRepositoryResponse, error) {
	var cred model.GitHubCredential
	if err := s.db.WithContext(ctx).First(&cred, id).Error; err != nil {
		return nil, err
	}
	token, err := s.GitHubToken(ctx, cred)
	if err != nil {
		return nil, err
	}
	endpoint := "https://api.github.com/user/repos?affiliation=owner,collaborator,organization_member&sort=updated&per_page=100"
	if cred.AuthType == "app" || cred.InstallationID != 0 {
		endpoint = "https://api.github.com/installation/repositories?per_page=100"
	}
	repos := make([]dto.GitHubRepositoryResponse, 0)
	for page := 1; page <= 10; page++ {
		pageURL := fmt.Sprintf("%s&page=%d", endpoint, page)
		var batch []githubRepositoryPayload
		if cred.AuthType == "app" || cred.InstallationID != 0 {
			var out struct {
				Repositories []githubRepositoryPayload `json:"repositories"`
			}
			if err := githubJSON(ctx, http.MethodGet, pageURL, token, &out); err != nil {
				return nil, err
			}
			batch = out.Repositories
		} else if err := githubJSON(ctx, http.MethodGet, pageURL, token, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		for _, repo := range batch {
			repos = append(repos, repo.toResponse())
		}
		if len(batch) < 100 {
			break
		}
	}
	return repos, nil
}

func (s *CredentialService) githubUserLogin(ctx context.Context, token string) (string, error) {
	var out struct {
		Login string `json:"login"`
	}
	if err := githubJSON(ctx, http.MethodGet, "https://api.github.com/user", token, &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Login) == "" {
		return "", fmt.Errorf("GitHub user login was not returned")
	}
	return out.Login, nil
}

func (s *CredentialService) githubAppJWT() (string, error) {
	rawKey := strings.TrimSpace(s.cfg.GitHubAppPrivateKey)
	if decoded, err := base64.StdEncoding.DecodeString(rawKey); err == nil && strings.Contains(string(decoded), "PRIVATE KEY") {
		rawKey = string(decoded)
	}
	rawKey = strings.ReplaceAll(rawKey, `\n`, "\n")
	block, _ := pem.Decode([]byte(rawKey))
	if block == nil {
		return "", fmt.Errorf("GitHub App private key must be PEM or base64 PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsed, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			return "", err
		}
		var ok bool
		key, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("GitHub App private key must be RSA")
		}
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Add(-time.Minute).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": s.cfg.GitHubAppID,
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
}

type githubRepositoryPayload struct {
	FullName      string `json:"full_name"`
	Name          string `json:"name"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func (r githubRepositoryPayload) toResponse() dto.GitHubRepositoryResponse {
	return dto.GitHubRepositoryResponse{
		FullName:      r.FullName,
		Name:          r.Name,
		Owner:         r.Owner.Login,
		Private:       r.Private,
		DefaultBranch: r.DefaultBranch,
		HTMLURL:       r.HTMLURL,
	}
}

func githubJSON(ctx context.Context, method, endpoint, token string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API request failed: %s", strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("GitHub returned invalid JSON")
	}
	return nil
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
