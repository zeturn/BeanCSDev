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
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zeturn/beancs-controller/internal/basaltpass"
	"github.com/zeturn/beancs-controller/internal/config"
	cryptoutil "github.com/zeturn/beancs-controller/internal/crypto"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/k8s"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CredentialService struct {
	db     *gorm.DB
	cipher cryptoutil.Cipher
	cfg    *config.Config
	k8s    *k8s.Manager
	dns    *DNSService
}

func NewCredentialService(db *gorm.DB, cipher cryptoutil.Cipher, cfg *config.Config) *CredentialService {
	return &CredentialService{db: db, cipher: cipher, cfg: cfg}
}

func (s *CredentialService) SetK8sManager(k8sManager *k8s.Manager) {
	s.k8s = k8sManager
}

func (s *CredentialService) SetDNSService(dns *DNSService) {
	s.dns = dns
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
	zones, err := s.cloudflareZones(ctx, req.APIToken, req.ZoneID, req.AccountID)
	if err != nil {
		return nil, err
	}
	if len(zones) == 0 && strings.TrimSpace(req.ZoneID) != "" && strings.TrimSpace(req.Domain) != "" {
		zones = []cloudflareZonePayload{{ID: strings.TrimSpace(req.ZoneID), Name: strings.TrimSpace(req.Domain), Account: cloudflareZoneAccount{ID: strings.TrimSpace(req.AccountID)}}}
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("no Cloudflare zones were returned for this token")
	}
	accountID := strings.TrimSpace(req.AccountID)
	for _, zone := range zones {
		if accountID == "" {
			accountID = strings.TrimSpace(zone.Account.ID)
		}
	}
	cred := model.CloudflareCredential{Name: name, APITokenEnc: enc, AccountID: accountID, AuthType: "api_token", IsActive: true, CreatedBy: userID}
	caches := cloudflareDomainCaches(0, accountID, zones)
	if len(caches) == 0 {
		return nil, fmt.Errorf("no usable Cloudflare zones were returned for this token")
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&cred).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.UserCredential{UserID: userID, CredentialType: model.CredentialTypeCloudflare, CredentialID: cred.ID, Role: model.CredentialRoleOwner}).Error; err != nil {
			return err
		}
		for i := range caches {
			caches[i].CloudflareCredentialID = cred.ID
			if err := upsertCloudflareDomainCache(tx, &caches[i]); err != nil {
				return err
			}
		}
		return nil
	})
	return []model.CloudflareCredential{cred}, err
}

func (s *CredentialService) CreateCloudflareOAuth(ctx context.Context, userID, name, code string) (*model.CloudflareCredential, error) {
	if s.cfg == nil || strings.TrimSpace(s.cfg.CloudflareOAuthClientID) == "" || strings.TrimSpace(s.cfg.CloudflareOAuthClientSecret) == "" {
		return nil, fmt.Errorf("Cloudflare OAuth app is not configured")
	}
	token, err := s.exchangeCloudflareOAuthCode(ctx, strings.TrimSpace(code))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return nil, fmt.Errorf("Cloudflare OAuth did not return an access token")
	}
	enc, err := s.cipher.EncryptString(token.AccessToken)
	if err != nil {
		return nil, err
	}
	var refreshEnc []byte
	if strings.TrimSpace(token.RefreshToken) != "" {
		refreshEnc, err = s.cipher.EncryptString(token.RefreshToken)
		if err != nil {
			return nil, err
		}
	}
	expiresAt := cloudflareTokenExpiry(token.ExpiresIn)
	zones, err := s.cloudflareZones(ctx, token.AccessToken, "", "")
	if err != nil {
		return nil, err
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("no Cloudflare zones were returned for this authorization")
	}
	accountID := ""
	for _, zone := range zones {
		if accountID == "" {
			accountID = strings.TrimSpace(zone.Account.ID)
		}
	}
	credentialName := strings.TrimSpace(name)
	if credentialName == "" {
		credentialName = "Cloudflare OAuth"
		if accountID != "" {
			credentialName = "Cloudflare " + accountID
		}
	}
	cred := &model.CloudflareCredential{
		Name:            credentialName,
		APITokenEnc:     enc,
		RefreshTokenEnc: refreshEnc,
		TokenExpiresAt:  expiresAt,
		AuthType:        "oauth",
		AccountID:       accountID,
		IsActive:        true,
		CreatedBy:       userID,
	}
	caches := cloudflareDomainCaches(0, accountID, zones)
	if len(caches) == 0 {
		return nil, fmt.Errorf("no usable Cloudflare zones were returned for this authorization")
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cred).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.UserCredential{UserID: userID, CredentialType: model.CredentialTypeCloudflare, CredentialID: cred.ID, Role: model.CredentialRoleOwner}).Error; err != nil {
			return err
		}
		for i := range caches {
			caches[i].CloudflareCredentialID = cred.ID
			if err := upsertCloudflareDomainCache(tx, &caches[i]); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return cred, nil
}

func (s *CredentialService) CreateGitHub(ctx context.Context, userID string, req dto.CreateGitHubCredentialRequest) (*model.GitHubCredential, error) {
	enc, err := s.cipher.EncryptString(req.Token)
	if err != nil {
		return nil, err
	}
	account, err := s.githubUserAccount(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = account.Login
	}
	org := strings.TrimSpace(req.Org)
	if org == "" {
		org = account.Login
	}
	cred := &model.GitHubCredential{Name: name, AuthType: "pat", TokenEnc: enc, AccountLogin: account.Login, AvatarURL: account.AvatarURL, Org: org, GitOpsRepo: strings.TrimSpace(req.GitOpsRepo), IsActive: true, CreatedBy: userID}
	err = s.createGitHubCredential(ctx, userID, cred)
	return cred, err
}

func (s *CredentialService) CreateGitHubApp(ctx context.Context, userID string, req dto.StartGitHubAppInstallRequest, installationID int64, accountLogin string) (*model.GitHubCredential, error) {
	account, err := s.GitHubAppInstallationAccount(ctx, installationID)
	if err == nil && strings.TrimSpace(account.Login) != "" {
		accountLogin = account.Login
	}
	accountLogin = strings.TrimSpace(accountLogin)
	name := accountLogin
	if name == "" {
		name = fmt.Sprintf("github-app-%d", installationID)
	}
	cred := &model.GitHubCredential{Name: name, AuthType: "app", InstallationID: installationID, AccountLogin: accountLogin, AvatarURL: account.AvatarURL, Org: accountLogin, GitOpsRepo: strings.TrimSpace(req.GitOpsRepo), IsActive: true, CreatedBy: userID}
	err = s.createGitHubCredential(ctx, userID, cred)
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
	req.BaseURL = strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	req.Name = strings.TrimSpace(req.Name)
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.TenantCode = strings.TrimSpace(req.TenantCode)
	req.ClientID = strings.TrimSpace(req.ClientID)
	if err := validateExternalHTTPSURL(req.BaseURL); err != nil {
		return nil, err
	}
	if req.TenantID == "" && req.TenantCode == "" {
		return nil, fmt.Errorf("tenant_id or tenant_code is required")
	}
	var clientSecretEnc []byte
	var automationTokenEnc []byte
	var err error
	if strings.TrimSpace(req.AutomationToken) != "" {
		automationTokenEnc, err = s.cipher.EncryptString(strings.TrimSpace(req.AutomationToken))
		if err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(req.ClientSecret) != "" {
		clientSecretEnc, err = s.cipher.EncryptString(req.ClientSecret)
		if err != nil {
			return nil, err
		}
	}
	var serviceTokenEnc []byte
	if strings.TrimSpace(req.ServiceToken) != "" {
		serviceTokenEnc, err = s.cipher.EncryptString(strings.TrimSpace(req.ServiceToken))
		if err != nil {
			return nil, err
		}
	}
	if len(automationTokenEnc) == 0 && (strings.TrimSpace(req.ClientID) == "" || len(clientSecretEnc) == 0) && len(serviceTokenEnc) == 0 {
		return nil, fmt.Errorf("automation_token is required unless client_id/client_secret or service_token are provided")
	}
	cred := &model.BasaltPassInstance{
		Name:               req.Name,
		BaseURL:            req.BaseURL,
		TenantID:           req.TenantID,
		TenantCode:         req.TenantCode,
		DeployMode:         "external",
		DeployStatus:       "ready",
		ClientID:           req.ClientID,
		ClientSecretEnc:    clientSecretEnc,
		ServiceTokenEnc:    serviceTokenEnc,
		AutomationTokenEnc: automationTokenEnc,
		IsActive:           true,
		CreatedBy:          userID,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cred).Error; err != nil {
			return err
		}
		return tx.Create(&model.UserCredential{UserID: userID, CredentialType: model.CredentialTypeBasaltPass, CredentialID: cred.ID, Role: model.CredentialRoleOwner}).Error
	})
	return cred, err
}

type basaltPassDeployLogFunc func(step, line string)

func (s *CredentialService) CreateBasaltPassDeploymentProcess(ctx context.Context, userID string, req dto.DeployBasaltPassRequest) (*model.Process, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "BasaltPass"
	}
	process := &model.Process{
		Type:        model.ProcessTypeBasaltPassDeployment,
		Status:      model.ProcessStatusQueued,
		OwnerID:     userID,
		Title:       "BasaltPass deployment: " + name,
		TriggeredBy: userID,
	}
	jobs := []processJobSpec{
		{Name: "validate", DisplayName: "Validate deployment request"},
		{Name: "runtime", DisplayName: "Prepare images and apply runtime"},
		{Name: "dns", DisplayName: "Prepare DNS"},
		{Name: "health", DisplayName: "Wait for BasaltPass health"},
		{Name: "tenant", DisplayName: "Create tenant owner"},
		{Name: "store", DisplayName: "Store tenant credential"},
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(process).Error; err != nil {
			return err
		}
		for i, spec := range jobs {
			job := model.ProcessJob{ProcessID: process.ID, Name: spec.Name, DisplayName: spec.DisplayName, Status: model.ProcessStatusQueued, StepIndex: i}
			if err := tx.Create(&job).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var out model.Process
	if err := s.db.WithContext(ctx).Preload("Jobs", func(db *gorm.DB) *gorm.DB {
		return db.Order("step_index asc")
	}).First(&out, process.ID).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *CredentialService) StartBasaltPassDeployment(processID uint, userID string, req dto.DeployBasaltPassRequest) {
	go s.runBasaltPassDeploymentProcess(context.Background(), processID, userID, req)
}

func (s *CredentialService) DeployBasaltPass(ctx context.Context, userID string, req dto.DeployBasaltPassRequest) (*model.BasaltPassInstance, error) {
	return s.deployBasaltPass(ctx, userID, req, nil)
}

func (s *CredentialService) deployBasaltPass(ctx context.Context, userID string, req dto.DeployBasaltPassRequest, log basaltPassDeployLogFunc) (*model.BasaltPassInstance, error) {
	req.BaseURL = strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	req.Name = strings.TrimSpace(req.Name)
	req.TenantName = strings.TrimSpace(req.TenantName)
	req.TenantCode = strings.TrimSpace(req.TenantCode)
	req.Namespace = strings.TrimSpace(req.Namespace)
	if req.Namespace == "" {
		req.Namespace = "bp-" + harborName(req.Name)
	}
	req.BackendImage = strings.TrimSpace(req.BackendImage)
	req.FrontendImage = strings.TrimSpace(req.FrontendImage)
	req.GitHubRepo = strings.TrimSpace(req.GitHubRepo)
	req.GitHubBranch = strings.TrimSpace(req.GitHubBranch)
	req.PublicHost = strings.TrimSpace(req.PublicHost)
	req.ExposureMode = strings.ToLower(strings.TrimSpace(req.ExposureMode))
	if req.ExposureMode == "" {
		req.ExposureMode = model.ExposurePublic
	}
	req.JWTSecret = strings.TrimSpace(req.JWTSecret)
	req.CORSAllowOrigins = strings.TrimSpace(req.CORSAllowOrigins)
	req.PlatformAdminEmail = strings.TrimSpace(strings.ToLower(req.PlatformAdminEmail))
	req.PlatformAdminUsername = strings.TrimSpace(req.PlatformAdminUsername)
	req.OwnerEmail = strings.TrimSpace(req.OwnerEmail)
	req.OwnerUsername = strings.TrimSpace(req.OwnerUsername)
	req.Description = strings.TrimSpace(req.Description)
	req.ClientID = strings.TrimSpace(req.ClientID)
	if _, _, err := validatePublicHTTPSURLSyntax(req.BaseURL); err != nil {
		return nil, err
	}
	if req.TenantCode == "" {
		req.TenantCode = harborName(req.Name)
	}
	if req.TenantName == "" {
		req.TenantName = req.Name
	}
	if req.OwnerUsername == "" {
		req.OwnerUsername = strings.Split(req.OwnerEmail, "@")[0]
	}
	if req.PlatformAdminUsername == "" {
		req.PlatformAdminUsername = strings.Split(req.PlatformAdminEmail, "@")[0]
	}
	if req.MaxApps <= 0 {
		req.MaxApps = 50
	}
	if req.MaxUsers <= 0 {
		req.MaxUsers = 500
	}
	if req.MaxTokensPerHour <= 0 {
		req.MaxTokensPerHour = 10000
	}
	if log != nil {
		log("validate", "request normalized")
		log("validate", "namespace="+req.Namespace)
		log("validate", "base_url="+req.BaseURL)
	}
	if err := s.validateBasaltPassDatabaseCredential(ctx, userID, req.DatabaseDependencyID, req.DatabaseCredentialID, true); err != nil {
		return nil, err
	}
	if log != nil {
		log("validate", "database credential access verified")
		log("runtime", "preparing BasaltPass images")
	}
	backendImage, frontendImage, err := s.deployManagedBasaltPass(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	req.BackendImage = backendImage
	req.FrontendImage = frontendImage
	if log != nil {
		log("runtime", "backend_image="+backendImage)
		log("runtime", "frontend_image="+frontendImage)
		log("runtime", "runtime manifests applied")
		log("dns", "preparing route for "+coalesce(req.PublicHost, hostFromURL(req.BaseURL)))
	}
	if err := s.ensureBasaltPassDNS(ctx, userID, req); err != nil {
		return nil, err
	}
	if log != nil {
		log("dns", "DNS step completed")
	}
	runtimeReq := req
	runtimeReq.BaseURL = basaltPassInternalBaseURL(req.Namespace)
	if log != nil {
		log("health", "waiting for "+runtimeReq.BaseURL)
	}
	if err := waitForBasaltPassHealth(ctx, runtimeReq.BaseURL, runtimeReq.ClientID, runtimeReq.ClientSecret, runtimeReq.ServiceToken); err != nil {
		return nil, err
	}
	if log != nil {
		log("health", "BasaltPass health check succeeded")
		log("tenant", "creating tenant "+coalesce(req.TenantCode, req.Name))
	}
	created, err := s.createBasaltPassTenant(ctx, runtimeReq)
	if err != nil {
		return nil, err
	}
	if log != nil {
		log("tenant", fmt.Sprintf("tenant created id=%v code=%s", created.ID, coalesce(created.Code, req.TenantCode)))
		log("store", "encrypting automation token")
	}
	tenantID := fmt.Sprint(created.ID)
	tenantCode := coalesce(req.TenantCode, created.Code)
	automationToken := strings.TrimSpace(req.AutomationToken)
	if automationToken == "" {
		automationToken = strings.TrimSpace(created.Token)
	}
	if automationToken == "" {
		return nil, fmt.Errorf("BasaltPass tenant was created but no tenant automation token was returned; provide automation_token to store")
	}
	automationTokenEnc, err := s.cipher.EncryptString(automationToken)
	if err != nil {
		return nil, err
	}
	var clientSecretEnc []byte
	if strings.TrimSpace(req.ClientSecret) != "" {
		clientSecretEnc, err = s.cipher.EncryptString(req.ClientSecret)
		if err != nil {
			return nil, err
		}
	}
	if log != nil {
		log("store", "storing BasaltPass credential")
	}
	cred := &model.BasaltPassInstance{
		Name:                 req.TenantName,
		BaseURL:              req.BaseURL,
		TenantID:             tenantID,
		TenantCode:           tenantCode,
		DeployMode:           "managed",
		Namespace:            req.Namespace,
		BackendImage:         req.BackendImage,
		FrontendImage:        req.FrontendImage,
		PublicHost:           req.PublicHost,
		DatabaseDependencyID: &req.DatabaseDependencyID,
		DatabaseCredentialID: &req.DatabaseCredentialID,
		DeployStatus:         "ready",
		ClientID:             req.ClientID,
		ClientSecretEnc:      clientSecretEnc,
		AutomationTokenEnc:   automationTokenEnc,
		IsActive:             true,
		CreatedBy:            userID,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cred).Error; err != nil {
			return err
		}
		return tx.Create(&model.UserCredential{UserID: userID, CredentialType: model.CredentialTypeBasaltPass, CredentialID: cred.ID, Role: model.CredentialRoleOwner}).Error
	})
	if err == nil && log != nil {
		log("store", fmt.Sprintf("stored credential id=%d", cred.ID))
	}
	return cred, err
}

func (s *CredentialService) runBasaltPassDeploymentProcess(ctx context.Context, processID uint, userID string, req dto.DeployBasaltPassRequest) {
	started := time.Now().UTC()
	_ = s.db.WithContext(ctx).Model(&model.Process{}).Where("id = ?", processID).Updates(map[string]any{
		"status": model.ProcessStatusRunning, "started_at": &started,
	}).Error
	var active *model.ProcessJob
	log := func(step, line string) {
		if active == nil || active.Name != step {
			if active != nil {
				_ = s.finishBasaltPassProcessJob(ctx, active, model.ProcessStatusSucceeded, "")
			}
			job, err := s.startBasaltPassProcessJob(ctx, processID, step)
			if err != nil {
				return
			}
			active = job
		}
		s.appendBasaltPassProcessJobLog(ctx, active, line)
	}
	log("validate", "starting BasaltPass deployment")
	_, err := s.deployBasaltPass(ctx, userID, req, log)
	if err != nil {
		if active != nil {
			_ = s.finishBasaltPassProcessJob(ctx, active, model.ProcessStatusFailed, err.Error())
		}
		finished := time.Now().UTC()
		_ = s.db.WithContext(ctx).Model(&model.Process{}).Where("id = ?", processID).Updates(map[string]any{
			"status": model.ProcessStatusFailed, "finished_at": &finished, "failure_reason": truncateFailure(err.Error()),
		}).Error
		return
	}
	if active != nil {
		_ = s.finishBasaltPassProcessJob(ctx, active, model.ProcessStatusSucceeded, "")
	}
	finished := time.Now().UTC()
	_ = s.db.WithContext(ctx).Model(&model.Process{}).Where("id = ?", processID).Updates(map[string]any{
		"status": model.ProcessStatusSucceeded, "finished_at": &finished, "failure_reason": "",
	}).Error
}

func (s *CredentialService) startBasaltPassProcessJob(ctx context.Context, processID uint, name string) (*model.ProcessJob, error) {
	now := time.Now().UTC()
	var job model.ProcessJob
	if err := s.db.WithContext(ctx).Where("process_id = ? AND name = ?", processID, name).First(&job).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&job).Updates(map[string]any{"status": model.ProcessStatusRunning, "started_at": &now}).Error; err != nil {
		return nil, err
	}
	job.Status = model.ProcessStatusRunning
	job.StartedAt = &now
	return &job, nil
}

func (s *CredentialService) appendBasaltPassProcessJobLog(ctx context.Context, job *model.ProcessJob, line string) {
	if job == nil {
		return
	}
	entry := fmt.Sprintf("[%s] %s\n", time.Now().UTC().Format(time.RFC3339), strings.TrimRight(line, "\n"))
	job.Logs += entry
	_ = s.db.WithContext(ctx).Model(job).Update("logs", job.Logs).Error
}

func (s *CredentialService) finishBasaltPassProcessJob(ctx context.Context, job *model.ProcessJob, status, failure string) error {
	if job == nil {
		return nil
	}
	if failure != "" {
		s.appendBasaltPassProcessJobLog(ctx, job, "ERROR: "+failure)
	}
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(job).Updates(map[string]any{
		"status": status, "finished_at": &now, "failure_reason": truncateFailure(failure),
	}).Error
}

func basaltPassInternalBaseURL(namespace string) string {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return ""
	}
	return "http://backend." + namespace + ".svc.cluster.local:8101"
}

func (s *CredentialService) ensureBasaltPassDNS(ctx context.Context, userID string, req dto.DeployBasaltPassRequest) error {
	if req.ExposureMode != model.ExposurePublic || req.CloudflareCredentialID == 0 || strings.TrimSpace(req.CloudflareZoneID) == "" {
		return nil
	}
	if s.dns == nil {
		return fmt.Errorf("dns service is not configured")
	}
	host := strings.TrimSpace(req.PublicHost)
	if host == "" {
		host = hostFromURL(req.BaseURL)
	}
	if host == "" {
		return fmt.Errorf("public_host is required to create BasaltPass DNS")
	}
	if err := s.RequireAccess(userID, model.CredentialTypeCloudflare, req.CloudflareCredentialID, false); err != nil {
		return err
	}
	cred, token, err := s.cloudflareCredentialZoneToken(ctx, req.CloudflareCredentialID, req.CloudflareZoneID)
	if err != nil {
		return err
	}
	_, err = s.dns.CreateRecordForHost(ctx, token, cred, req.Name, host)
	return err
}

func (s *CredentialService) createBasaltPassTenant(ctx context.Context, req dto.DeployBasaltPassRequest) (*basaltpass.CreateTenantResponse, error) {
	tenantName := coalesce(req.TenantName, req.Name)
	if strings.TrimSpace(req.ServiceToken) != "" {
		client := basaltpass.NewHTTPClientWithServiceToken(req.BaseURL, req.ClientID, req.ClientSecret, strings.TrimSpace(req.ServiceToken))
		return client.CreateTenant(ctx, &basaltpass.CreateTenantRequest{
			Name:             tenantName,
			Code:             req.TenantCode,
			Description:      req.Description,
			OwnerEmail:       req.OwnerEmail,
			OwnerUsername:    req.OwnerUsername,
			OwnerPassword:    req.OwnerPassword,
			MaxApps:          req.MaxApps,
			MaxUsers:         req.MaxUsers,
			MaxTokensPerHour: req.MaxTokensPerHour,
		})
	}
	if req.PlatformAdminEmail != "" && strings.TrimSpace(req.PlatformAdminPassword) != "" {
		client := basaltpass.NewHTTPClientWithAdminCredentials(req.BaseURL, req.PlatformAdminEmail, req.PlatformAdminPassword)
		return client.CreateTenant(ctx, &basaltpass.CreateTenantRequest{
			Name:             tenantName,
			Code:             req.TenantCode,
			Description:      req.Description,
			OwnerEmail:       req.OwnerEmail,
			OwnerUsername:    req.OwnerUsername,
			OwnerPassword:    req.OwnerPassword,
			MaxApps:          req.MaxApps,
			MaxUsers:         req.MaxUsers,
			MaxTokensPerHour: req.MaxTokensPerHour,
		})
	}
	if s.cfg == nil || strings.TrimSpace(s.cfg.BPMgmtBaseURL) == "" || strings.TrimSpace(s.cfg.BPMgmtClientID) == "" || strings.TrimSpace(s.cfg.BPMgmtClientSecret) == "" {
		return nil, fmt.Errorf("BasaltPass management credentials are not configured")
	}
	client := basaltpass.NewHTTPClient(s.cfg.BPMgmtBaseURL, s.cfg.BPMgmtClientID, s.cfg.BPMgmtClientSecret)
	return client.CreateTenant(ctx, &basaltpass.CreateTenantRequest{
		Name:             tenantName,
		Code:             req.TenantCode,
		Description:      req.Description,
		OwnerEmail:       req.OwnerEmail,
		OwnerUsername:    req.OwnerUsername,
		OwnerPassword:    req.OwnerPassword,
		MaxApps:          req.MaxApps,
		MaxUsers:         req.MaxUsers,
		MaxTokensPerHour: req.MaxTokensPerHour,
	})
}

func (s *CredentialService) deployManagedBasaltPass(ctx context.Context, userID string, req dto.DeployBasaltPassRequest) (string, string, error) {
	if s.k8s == nil {
		return "", "", fmt.Errorf("kubernetes manager is not configured")
	}
	if req.BackendImage == "" || req.FrontendImage == "" {
		return "", "", fmt.Errorf("backend_image and frontend_image are required for managed BasaltPass deployment")
	}
	backendImage, frontendImage, pullSecret, err := s.prepareBasaltPassImages(ctx, userID, req)
	if err != nil {
		return "", "", err
	}
	dep, cred, outputs, err := s.basaltPassDatabaseCredential(ctx, userID, req.DatabaseDependencyID, req.DatabaseCredentialID)
	if err != nil {
		return "", "", err
	}
	dsn := strings.TrimSpace(outputs["url"])
	if dsn == "" {
		return "", "", fmt.Errorf("selected database credential does not expose url output")
	}
	driver := "mysql"
	if isPostgreSQLCompatibleDependency(dep.Type) {
		driver = "postgres"
	}
	jwtSecret := strings.TrimSpace(req.JWTSecret)
	if jwtSecret == "" {
		jwtSecret = randomToken(48)
	}
	corsOrigins := strings.TrimSpace(req.CORSAllowOrigins)
	if corsOrigins == "" || corsOrigins == "*" {
		corsOrigins = req.BaseURL
	}
	host := strings.TrimSpace(req.PublicHost)
	if host == "" {
		host = hostFromURL(req.BaseURL)
	}
	if err := s.k8s.ApplyBasaltPass(ctx, k8s.BasaltPassRuntime{
		Name:          harborName(req.Name),
		Namespace:     req.Namespace,
		BackendImage:  backendImage,
		FrontendImage: frontendImage,
		PullSecret:    pullSecret,
		Host:          host,
		Exposure:      req.ExposureMode,
		Env: map[string]string{
			"JWT_SECRET":                              jwtSecret,
			"BASALTPASS_ENV":                          "production",
			"BASALTPASS_SERVER_ADDRESS":               ":8101",
			"BASALTPASS_UI_BASE_URL":                  req.BaseURL,
			"BASALTPASS_DATABASE_DRIVER":              driver,
			"BASALTPASS_DATABASE_DSN":                 dsn,
			"BASALTPASS_CORS_ALLOW_ORIGINS":           corsOrigins,
			"BASALTPASS_SERVER_CORS_ALLOWED_ORIGINS":  corsOrigins,
			"BASALTPASS_ADMIN_EMAIL":                  req.PlatformAdminEmail,
			"BASALTPASS_ADMIN_USERNAME":               req.PlatformAdminUsername,
			"BASALTPASS_ADMIN_PASSWORD":               req.PlatformAdminPassword,
			"BEANCS_DATABASE_DEPENDENCY_ID":           strconv.FormatUint(uint64(dep.ID), 10),
			"BEANCS_DATABASE_CREDENTIAL_ID":           strconv.FormatUint(uint64(cred.ID), 10),
			"BEANCS_DATABASE_CREDENTIAL_NAME":         cred.Name,
			"BEANCS_DATABASE_DEPENDENCY_NAME":         dep.Name,
			"BEANCS_DATABASE_DEPENDENCY_DEFINITION":   dep.DefinitionName,
			"BEANCS_DATABASE_DEPENDENCY_RUNTIME_HOST": outputs["host"],
			"BEANCS_DATABASE_DEPENDENCY_RUNTIME_PORT": outputs["port"],
			"BEANCS_DATABASE_DEPENDENCY_RUNTIME_DB":   outputs["database"],
			"BEANCS_DATABASE_DEPENDENCY_RUNTIME_USER": outputs["username"],
		},
	}); err != nil {
		return "", "", err
	}
	return backendImage, frontendImage, nil
}

func (s *CredentialService) prepareBasaltPassImages(ctx context.Context, userID string, req dto.DeployBasaltPassRequest) (string, string, string, error) {
	backendImage := strings.TrimSpace(req.BackendImage)
	frontendImage := strings.TrimSpace(req.FrontendImage)
	if req.GitHubCredentialID == 0 || !isGHCRImage(backendImage) || !isGHCRImage(frontendImage) {
		pullSecret, err := s.ensureBasaltPassRegistryPullSecret(ctx, req, backendImage, frontendImage)
		if err != nil {
			return "", "", "", err
		}
		if pullSecret != "" {
			return backendImage, frontendImage, pullSecret, nil
		}
		return backendImage, frontendImage, "", nil
	}
	if s.cfg == nil || strings.TrimSpace(s.cfg.RegistryHost) == "" || strings.TrimSpace(s.cfg.RegistryUsername) == "" || strings.TrimSpace(s.cfg.RegistryToken) == "" {
		return "", "", "", fmt.Errorf("BeanCS registry credentials are required to mirror BasaltPass images")
	}
	if err := s.RequireAccess(userID, model.CredentialTypeGitHub, req.GitHubCredentialID, false); err != nil {
		return "", "", "", err
	}
	var ghCred model.GitHubCredential
	if err := s.db.WithContext(ctx).First(&ghCred, req.GitHubCredentialID).Error; err != nil {
		return "", "", "", err
	}
	ghToken, err := s.GitHubToken(ctx, ghCred)
	if err != nil {
		return "", "", "", err
	}
	name := harborName(req.Name)
	projectName := harborProjectName(coalesce(req.TenantCode, name))
	if projectName == "" {
		return "", "", "", fmt.Errorf("tenant_code is required to mirror BasaltPass images")
	}
	if err := ensureHarborProject(ctx, s.cfg, projectName); err != nil {
		return "", "", "", err
	}
	host := normalizeRegistryHost(s.cfg.RegistryHost)
	tag := coalesce(imageTag(backendImage), imageTag(frontendImage))
	if tag == "" {
		tag = "latest"
	}
	backendTarget := fmt.Sprintf("%s/%s/%s-backend:%s", host, projectName, name, tag)
	frontendTarget := fmt.Sprintf("%s/%s/%s-frontend:%s", host, projectName, name, coalesce(imageTag(frontendImage), tag))
	sourceUsername := strings.TrimSpace(ghCred.AccountLogin)
	if ghCred.AuthType == "app" || ghCred.InstallationID != 0 || sourceUsername == "" {
		sourceUsername = "x-access-token"
	}
	sourceAuth := registryAuth{Username: sourceUsername, Password: ghToken}
	targetAuth := registryAuth{Username: strings.TrimSpace(s.cfg.RegistryUsername), Password: s.cfg.RegistryToken}
	if err := copyContainerImage(ctx, backendImage, backendTarget, sourceAuth, targetAuth); err != nil {
		return "", "", "", err
	}
	if err := copyContainerImage(ctx, frontendImage, frontendTarget, sourceAuth, targetAuth); err != nil {
		return "", "", "", err
	}
	project := &model.Project{
		Name:                   name,
		Namespace:              req.Namespace,
		RegistryHost:           host,
		RegistryProject:        projectName,
		RegistryRepository:     name,
		RegistryPullSecretName: strings.TrimSpace(s.cfg.RegistryPullSecret),
	}
	if project.RegistryPullSecretName == "" {
		project.RegistryPullSecretName = "beancs-registry-pull"
	}
	creds, err := ensureHarborPullRobot(ctx, s.cfg, project)
	if err != nil {
		return "", "", "", err
	}
	if s.k8s == nil {
		return "", "", "", fmt.Errorf("kubernetes manager is not configured")
	}
	if err := s.k8s.CreateNamespace(ctx, req.Namespace, name); err != nil {
		return "", "", "", err
	}
	if err := s.k8s.UpsertRegistryPullSecretWithCredentials(ctx, req.Namespace, name, project.RegistryPullSecretName, creds.Host, creds.Username, creds.Token); err != nil {
		return "", "", "", err
	}
	return backendTarget, frontendTarget, project.RegistryPullSecretName, nil
}

func (s *CredentialService) ensureBasaltPassRegistryPullSecret(ctx context.Context, req dto.DeployBasaltPassRequest, backendImage, frontendImage string) (string, error) {
	if s.cfg == nil || strings.TrimSpace(s.cfg.RegistryHost) == "" {
		return "", nil
	}
	host := normalizeRegistryHost(s.cfg.RegistryHost)
	backendProject, ok := registryProjectFromImage(backendImage, host)
	if !ok {
		return "", nil
	}
	frontendProject, ok := registryProjectFromImage(frontendImage, host)
	if !ok || frontendProject != backendProject {
		return "", nil
	}
	if s.k8s == nil {
		return "", fmt.Errorf("kubernetes manager is not configured")
	}
	name := harborName(req.Name)
	pullSecret := strings.TrimSpace(s.cfg.RegistryPullSecret)
	if pullSecret == "" {
		pullSecret = "beancs-registry-pull"
	}
	project := &model.Project{
		Name:                   name,
		Namespace:              req.Namespace,
		RegistryHost:           host,
		RegistryProject:        backendProject,
		RegistryRepository:     name,
		RegistryPullSecretName: pullSecret,
	}
	creds, err := ensureHarborPullRobot(ctx, s.cfg, project)
	if err != nil {
		return "", err
	}
	if err := s.k8s.CreateNamespace(ctx, req.Namespace, name); err != nil {
		return "", err
	}
	if err := s.k8s.UpsertRegistryPullSecretWithCredentials(ctx, req.Namespace, name, pullSecret, creds.Host, creds.Username, creds.Token); err != nil {
		return "", err
	}
	return pullSecret, nil
}

func isGHCRImage(image string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(image)), "ghcr.io/")
}

func registryProjectFromImage(image, registryHost string) (string, bool) {
	image = strings.TrimSpace(image)
	registryHost = strings.Trim(strings.TrimSpace(registryHost), "/")
	if image == "" || registryHost == "" {
		return "", false
	}
	prefix := registryHost + "/"
	if !strings.HasPrefix(strings.ToLower(image), strings.ToLower(prefix)) {
		return "", false
	}
	rest := image[len(prefix):]
	project, _, ok := strings.Cut(rest, "/")
	project = strings.TrimSpace(project)
	return project, ok && project != ""
}

func imageTag(image string) string {
	image = strings.TrimSpace(image)
	if image == "" {
		return ""
	}
	if idx := strings.LastIndex(image, "@sha256:"); idx >= 0 {
		digest := image[idx+len("@sha256:"):]
		if len(digest) > 12 {
			digest = digest[:12]
		}
		return "sha256-" + harborName(digest)
	}
	lastSlash := strings.LastIndex(image, "/")
	lastColon := strings.LastIndex(image, ":")
	if lastColon <= lastSlash {
		return ""
	}
	return harborName(image[lastColon+1:])
}

func (s *CredentialService) basaltPassDatabaseCredential(ctx context.Context, userID string, dependencyID, credentialID uint) (model.ManagedDependency, model.DependencyCredential, map[string]string, error) {
	var dep model.ManagedDependency
	if err := s.db.WithContext(ctx).
		Joins("JOIN applications a ON a.id = managed_dependencies.application_id").
		Where("managed_dependencies.id = ? AND a.owner_id = ?", dependencyID, userID).
		First(&dep).Error; err != nil {
		return dep, model.DependencyCredential{}, nil, fmt.Errorf("database dependency not found or not accessible")
	}
	var cred model.DependencyCredential
	if err := s.db.WithContext(ctx).Where("id = ? AND dependency_id = ?", credentialID, dependencyID).First(&cred).Error; err != nil {
		return dep, cred, nil, fmt.Errorf("database credential not found for dependency")
	}
	return dep, cred, flattenCredentialOutputs(cred.Outputs), nil
}

func flattenCredentialOutputs(outputs model.JSONMap) map[string]string {
	out := map[string]string{}
	for key, raw := range outputs {
		if m, ok := raw.(map[string]any); ok {
			out[key] = fmt.Sprint(m["value"])
			continue
		}
		out[key] = fmt.Sprint(raw)
	}
	return out
}

func hostFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func waitForBasaltPassHealth(ctx context.Context, baseURL, clientID, clientSecret, serviceToken string) error {
	client := basaltpass.NewHTTPClientWithServiceToken(baseURL, clientID, clientSecret, strings.TrimSpace(serviceToken))
	deadline := time.Now().Add(8 * time.Minute)
	var lastErr error
	for {
		if _, err := client.HealthCheck(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("BasaltPass health check did not become ready: %w", lastErr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
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
	seen := map[string]bool{}
	for _, cred := range creds {
		var cached []model.CloudflareDomainCache
		_ = s.db.WithContext(ctx).Where("cloudflare_credential_id = ?", cred.ID).Order("domain asc").Find(&cached).Error
		for _, domain := range cached {
			if domain.Domain == "" || domain.ZoneID == "" {
				continue
			}
			key := strings.ToLower(domain.AccountID + "|" + domain.ZoneID + "|" + domain.Domain)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, dto.CloudflareDomainResponse{
				CredentialID: cred.ID,
				Credential:   cred.Name,
				ZoneID:       domain.ZoneID,
				Domain:       domain.Domain,
				AccountID:    domain.AccountID,
				Status:       domain.Status,
				Active:       cred.IsActive,
			})
		}
		if cred.Domain != "" && cred.ZoneID != "" {
			key := strings.ToLower(cred.AccountID + "|" + cred.ZoneID + "|" + cred.Domain)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, dto.CloudflareDomainResponse{
				CredentialID: cred.ID,
				Credential:   cred.Name,
				ZoneID:       cred.ZoneID,
				Domain:       cred.Domain,
				AccountID:    cred.AccountID,
				Active:       cred.IsActive,
			})
		}
	}
	return out, nil
}

func (s *CredentialService) RefreshCloudflareDomains(ctx context.Context, id uint) ([]dto.CloudflareDomainResponse, error) {
	cred, token, err := s.cloudflareCredentialToken(ctx, id)
	if err != nil {
		return nil, err
	}
	zones, err := s.cloudflareZones(ctx, token, cred.ZoneID, cred.AccountID)
	if err != nil {
		return nil, err
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("no Cloudflare zones were returned for this token")
	}
	accountID := strings.TrimSpace(cred.AccountID)
	for _, zone := range zones {
		if accountID == "" {
			accountID = strings.TrimSpace(zone.Account.ID)
		}
	}
	caches := cloudflareDomainCaches(cred.ID, accountID, zones)
	if len(caches) == 0 {
		return nil, fmt.Errorf("no usable Cloudflare zones were returned for this token")
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if accountID != "" && cred.AccountID == "" {
			if err := tx.Model(&model.CloudflareCredential{}).Where("id = ?", cred.ID).Update("account_id", accountID).Error; err != nil {
				return err
			}
			cred.AccountID = accountID
		}
		for i := range caches {
			if err := upsertCloudflareDomainCache(tx, &caches[i]); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return cloudflareDomainResponsesForCredential(ctx, s.db, cred), nil
}

func cloudflareDomainResponsesForCredential(ctx context.Context, db *gorm.DB, cred model.CloudflareCredential) []dto.CloudflareDomainResponse {
	out := []dto.CloudflareDomainResponse{}
	seen := map[string]bool{}
	var cached []model.CloudflareDomainCache
	_ = db.WithContext(ctx).Where("cloudflare_credential_id = ?", cred.ID).Order("domain asc").Find(&cached).Error
	for _, domain := range cached {
		if domain.Domain == "" || domain.ZoneID == "" {
			continue
		}
		key := strings.ToLower(domain.AccountID + "|" + domain.ZoneID + "|" + domain.Domain)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, dto.CloudflareDomainResponse{
			CredentialID: cred.ID,
			Credential:   cred.Name,
			ZoneID:       domain.ZoneID,
			Domain:       domain.Domain,
			AccountID:    domain.AccountID,
			Status:       domain.Status,
			Active:       cred.IsActive,
		})
	}
	if cred.Domain != "" && cred.ZoneID != "" {
		key := strings.ToLower(cred.AccountID + "|" + cred.ZoneID + "|" + cred.Domain)
		if !seen[key] {
			out = append(out, dto.CloudflareDomainResponse{
				CredentialID: cred.ID,
				Credential:   cred.Name,
				ZoneID:       cred.ZoneID,
				Domain:       cred.Domain,
				AccountID:    cred.AccountID,
				Active:       cred.IsActive,
			})
		}
	}
	return out
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

func (s *CredentialService) CloudflareToken(ctx context.Context, cred model.CloudflareCredential) (string, error) {
	token, err := s.DecryptCloudflareToken(cred)
	if err != nil {
		return "", err
	}
	if cred.AuthType != "oauth" || cred.TokenExpiresAt == nil || cred.TokenExpiresAt.After(time.Now().Add(2*time.Minute)) {
		return token, nil
	}
	if len(cred.RefreshTokenEnc) == 0 {
		return token, nil
	}
	refreshToken, err := s.cipher.DecryptString(cred.RefreshTokenEnc)
	if err != nil {
		return "", err
	}
	refreshed, err := s.refreshCloudflareOAuthToken(ctx, refreshToken)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(refreshed.AccessToken) == "" {
		return "", fmt.Errorf("Cloudflare OAuth refresh did not return an access token")
	}
	accessEnc, err := s.cipher.EncryptString(refreshed.AccessToken)
	if err != nil {
		return "", err
	}
	refreshEnc := cred.RefreshTokenEnc
	if strings.TrimSpace(refreshed.RefreshToken) != "" {
		refreshEnc, err = s.cipher.EncryptString(refreshed.RefreshToken)
		if err != nil {
			return "", err
		}
	}
	updates := map[string]any{
		"api_token_enc":     accessEnc,
		"refresh_token_enc": refreshEnc,
		"token_expires_at":  cloudflareTokenExpiry(refreshed.ExpiresIn),
	}
	if err := s.db.WithContext(ctx).Model(&model.CloudflareCredential{}).Where("id = ?", cred.ID).Updates(updates).Error; err != nil {
		return "", err
	}
	return refreshed.AccessToken, nil
}

func (s *CredentialService) VerifyCloudflare(ctx context.Context, id uint) (map[string]any, error) {
	var cred model.CloudflareCredential
	if err := s.db.WithContext(ctx).First(&cred, id).Error; err != nil {
		return nil, err
	}
	token, err := s.CloudflareToken(ctx, cred)
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
		var zones int64
		_ = s.db.WithContext(ctx).Model(&model.CloudflareDomainCache{}).Where("cloudflare_credential_id = ?", cred.ID).Count(&zones).Error
		return map[string]any{
			"status":     "ok",
			"credential": cred.Name,
			"account_id": cred.AccountID,
			"token":      verify["result"],
			"zones":      zones,
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

func (s *CredentialService) ListCloudflareDNSRecords(ctx context.Context, id uint, zoneID string) ([]dto.CloudflareDNSRecordResponse, error) {
	cred, token, err := s.cloudflareCredentialZoneToken(ctx, id, zoneID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cred.ZoneID) == "" {
		return nil, fmt.Errorf("cloudflare credential has no zone_id")
	}
	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?per_page=100", url.PathEscape(cred.ZoneID))
	result, err := cloudflareGET(ctx, &http.Client{Timeout: 15 * time.Second}, token, endpoint)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(result["result"])
	if err != nil {
		return nil, err
	}
	var records []dto.CloudflareDNSRecordResponse
	if err := json.Unmarshal(raw, &records); err != nil {
		return nil, fmt.Errorf("cloudflare returned invalid DNS record JSON")
	}
	return records, nil
}

func (s *CredentialService) CreateCloudflareDNSRecord(ctx context.Context, id uint, zoneID string, req dto.CreateCloudflareDNSRecordRequest) (dto.CloudflareDNSRecordResponse, error) {
	return s.writeCloudflareDNSRecord(ctx, id, zoneID, "", dto.UpdateCloudflareDNSRecordRequest(req), http.MethodPost)
}

func (s *CredentialService) UpdateCloudflareDNSRecord(ctx context.Context, id uint, zoneID, recordID string, req dto.UpdateCloudflareDNSRecordRequest) (dto.CloudflareDNSRecordResponse, error) {
	return s.writeCloudflareDNSRecord(ctx, id, zoneID, recordID, req, http.MethodPut)
}

func (s *CredentialService) DeleteCloudflareDNSRecord(ctx context.Context, id uint, zoneID, recordID string) error {
	cred, token, err := s.cloudflareCredentialZoneToken(ctx, id, zoneID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(recordID) == "" {
		return fmt.Errorf("record id is required")
	}
	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", url.PathEscape(cred.ZoneID), url.PathEscape(recordID))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Cloudflare DNS delete failed: %s", strings.TrimSpace(string(body)))
	}
	return nil
}

func (s *CredentialService) writeCloudflareDNSRecord(ctx context.Context, id uint, zoneID, recordID string, req dto.UpdateCloudflareDNSRecordRequest, method string) (dto.CloudflareDNSRecordResponse, error) {
	cred, token, err := s.cloudflareCredentialZoneToken(ctx, id, zoneID)
	if err != nil {
		return dto.CloudflareDNSRecordResponse{}, err
	}
	if strings.TrimSpace(cred.ZoneID) == "" {
		return dto.CloudflareDNSRecordResponse{}, fmt.Errorf("cloudflare credential has no zone_id")
	}
	ttl := req.TTL
	if ttl == 0 {
		ttl = 1
	}
	payload := map[string]any{
		"type":    strings.ToUpper(strings.TrimSpace(req.Type)),
		"name":    strings.TrimSpace(req.Name),
		"content": strings.TrimSpace(req.Content),
		"ttl":     ttl,
		"proxied": req.Proxied,
		"comment": strings.TrimSpace(req.Comment),
	}
	body, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", url.PathEscape(cred.ZoneID))
	if method == http.MethodPut {
		if strings.TrimSpace(recordID) == "" {
			return dto.CloudflareDNSRecordResponse{}, fmt.Errorf("record id is required")
		}
		endpoint += "/" + url.PathEscape(recordID)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return dto.CloudflareDNSRecordResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	result, err := cloudflareDo(ctx, &http.Client{Timeout: 15 * time.Second}, httpReq)
	if err != nil {
		return dto.CloudflareDNSRecordResponse{}, err
	}
	raw, err := json.Marshal(result["result"])
	if err != nil {
		return dto.CloudflareDNSRecordResponse{}, err
	}
	var out dto.CloudflareDNSRecordResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return dto.CloudflareDNSRecordResponse{}, fmt.Errorf("cloudflare returned invalid DNS record JSON")
	}
	return out, nil
}

func (s *CredentialService) cloudflareCredentialToken(ctx context.Context, id uint) (model.CloudflareCredential, string, error) {
	var cred model.CloudflareCredential
	if err := s.db.WithContext(ctx).First(&cred, id).Error; err != nil {
		return cred, "", err
	}
	token, err := s.CloudflareToken(ctx, cred)
	return cred, token, err
}

func (s *CredentialService) cloudflareCredentialZoneToken(ctx context.Context, id uint, zoneID string) (model.CloudflareCredential, string, error) {
	cred, token, err := s.cloudflareCredentialToken(ctx, id)
	if err != nil {
		return cred, "", err
	}
	zoneID = strings.TrimSpace(zoneID)
	if zoneID != "" {
		if cred.ZoneID != "" && cred.ZoneID != zoneID {
			return cred, "", fmt.Errorf("cloudflare zone does not belong to this credential")
		}
		if cred.ZoneID == "" {
			var cached model.CloudflareDomainCache
			if err := s.db.WithContext(ctx).Where("cloudflare_credential_id = ? AND zone_id = ?", id, zoneID).First(&cached).Error; err != nil {
				return cred, "", fmt.Errorf("cloudflare zone is not cached for this credential")
			}
			cred.ZoneID = cached.ZoneID
			cred.Domain = cached.Domain
			if cred.AccountID == "" {
				cred.AccountID = cached.AccountID
			}
		}
	}
	return cred, token, nil
}

func (s *CredentialService) CloudflareCredentialForDomain(ctx context.Context, credentialID uint, zoneID, fqdn string) (model.CloudflareCredential, error) {
	var cred model.CloudflareCredential
	if err := s.db.WithContext(ctx).First(&cred, credentialID).Error; err != nil {
		return cred, err
	}
	if cred.ZoneID != "" {
		if strings.TrimSpace(zoneID) != "" && cred.ZoneID != strings.TrimSpace(zoneID) {
			return cred, fmt.Errorf("cloudflare zone does not belong to this credential")
		}
		return cred, nil
	}
	var cached model.CloudflareDomainCache
	q := s.db.WithContext(ctx).Where("cloudflare_credential_id = ?", credentialID)
	if strings.TrimSpace(zoneID) != "" {
		q = q.Where("zone_id = ?", strings.TrimSpace(zoneID))
	} else {
		q = q.Order("length(domain) desc")
		for _, suffix := range domainSuffixes(fqdn) {
			var row model.CloudflareDomainCache
			if err := q.Where("domain = ?", suffix).First(&row).Error; err == nil {
				cached = row
				break
			}
		}
		if cached.ID == 0 {
			return cred, fmt.Errorf("no cached Cloudflare zone matches %q", fqdn)
		}
	}
	if cached.ID == 0 {
		if err := q.First(&cached).Error; err != nil {
			return cred, fmt.Errorf("cloudflare zone is not cached for this credential")
		}
	}
	cred.ZoneID = cached.ZoneID
	cred.Domain = cached.Domain
	if cred.AccountID == "" {
		cred.AccountID = cached.AccountID
	}
	return cred, nil
}

func cloudflareDomainCaches(credentialID uint, accountID string, zones []cloudflareZonePayload) []model.CloudflareDomainCache {
	seen := map[string]bool{}
	out := make([]model.CloudflareDomainCache, 0, len(zones))
	for _, zone := range zones {
		zoneID := strings.TrimSpace(zone.ID)
		domain := strings.ToLower(strings.TrimSpace(zone.Name))
		if zoneID == "" || domain == "" {
			continue
		}
		key := zoneID + "|" + domain
		if seen[key] {
			continue
		}
		seen[key] = true
		rowAccountID := strings.TrimSpace(accountID)
		if rowAccountID == "" {
			rowAccountID = strings.TrimSpace(zone.Account.ID)
		}
		out = append(out, model.CloudflareDomainCache{
			CloudflareCredentialID: credentialID,
			ZoneID:                 zoneID,
			Domain:                 domain,
			AccountID:              rowAccountID,
			Status:                 strings.TrimSpace(zone.Status),
		})
	}
	return out
}

func upsertCloudflareDomainCache(tx *gorm.DB, row *model.CloudflareDomainCache) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cloudflare_credential_id"}, {Name: "zone_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"domain", "account_id", "status", "updated_at"}),
	}).Create(row).Error
}

func domainSuffixes(fqdn string) []string {
	parts := strings.Split(strings.ToLower(strings.Trim(fqdn, ".")), ".")
	out := make([]string, 0, len(parts))
	for i := 0; i < len(parts)-1; i++ {
		out = append(out, strings.Join(parts[i:], "."))
	}
	return out
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

type cloudflareOAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

func cloudflareTokenExpiry(expiresIn int) *time.Time {
	if expiresIn <= 0 {
		return nil
	}
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	return &expiresAt
}

func (s *CredentialService) exchangeCloudflareOAuthCode(ctx context.Context, code string) (cloudflareOAuthTokenResponse, error) {
	if code == "" {
		return cloudflareOAuthTokenResponse{}, fmt.Errorf("Cloudflare authorization code is required")
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", strings.TrimSpace(s.cfg.CloudflareOAuthRedirectURL))
	return s.cloudflareOAuthToken(ctx, form)
}

func (s *CredentialService) refreshCloudflareOAuthToken(ctx context.Context, refreshToken string) (cloudflareOAuthTokenResponse, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return cloudflareOAuthTokenResponse{}, fmt.Errorf("Cloudflare refresh token is required")
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", strings.TrimSpace(refreshToken))
	return s.cloudflareOAuthToken(ctx, form)
}

func (s *CredentialService) cloudflareOAuthToken(ctx context.Context, form url.Values) (cloudflareOAuthTokenResponse, error) {
	if s.cfg == nil || strings.TrimSpace(s.cfg.CloudflareOAuthClientID) == "" || strings.TrimSpace(s.cfg.CloudflareOAuthClientSecret) == "" {
		return cloudflareOAuthTokenResponse{}, fmt.Errorf("Cloudflare OAuth app is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://dash.cloudflare.com/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return cloudflareOAuthTokenResponse{}, err
	}
	req.SetBasicAuth(strings.TrimSpace(s.cfg.CloudflareOAuthClientID), strings.TrimSpace(s.cfg.CloudflareOAuthClientSecret))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return cloudflareOAuthTokenResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return cloudflareOAuthTokenResponse{}, fmt.Errorf("Cloudflare OAuth token request failed: %s", strings.TrimSpace(string(body)))
	}
	var out cloudflareOAuthTokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return cloudflareOAuthTokenResponse{}, fmt.Errorf("Cloudflare OAuth token response was invalid")
	}
	return out, nil
}

func (s *CredentialService) cloudflareZones(ctx context.Context, token, zoneID, accountID string) ([]cloudflareZonePayload, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	endpoint := "https://api.cloudflare.com/client/v4/zones?per_page=100"
	if strings.TrimSpace(accountID) != "" {
		endpoint += "&account.id=" + url.QueryEscape(strings.TrimSpace(accountID))
	}
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

type githubAccount struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

func (s *CredentialService) GitHubAppInstallationAccount(ctx context.Context, installationID int64) (githubAccount, error) {
	if s.cfg == nil || s.cfg.GitHubAppID == 0 || strings.TrimSpace(s.cfg.GitHubAppPrivateKey) == "" {
		return githubAccount{}, fmt.Errorf("GitHub App is not configured")
	}
	appJWT, err := s.githubAppJWT()
	if err != nil {
		return githubAccount{}, err
	}
	var out struct {
		Account githubAccount `json:"account"`
	}
	endpoint := fmt.Sprintf("https://api.github.com/app/installations/%d", installationID)
	if err := githubJSON(ctx, http.MethodGet, endpoint, appJWT, &out); err != nil {
		return githubAccount{}, err
	}
	if strings.TrimSpace(out.Account.Login) == "" {
		return githubAccount{}, fmt.Errorf("GitHub installation account was not returned")
	}
	return out.Account, nil
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

func (s *CredentialService) githubUserAccount(ctx context.Context, token string) (githubAccount, error) {
	var out githubAccount
	if err := githubJSON(ctx, http.MethodGet, "https://api.github.com/user", token, &out); err != nil {
		return githubAccount{}, err
	}
	if strings.TrimSpace(out.Login) == "" {
		return githubAccount{}, fmt.Errorf("GitHub user login was not returned")
	}
	return out, nil
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

func cloudflareDo(ctx context.Context, client *http.Client, req *http.Request) (map[string]any, error) {
	resp, err := client.Do(req.WithContext(ctx))
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
		return nil, fmt.Errorf("cloudflare request failed: %s", strings.TrimSpace(string(body)))
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
		cred.RefreshTokenEnc = nil
		cred.TokenExpiresAt = nil
		cred.AuthType = "api_token"
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
		cred.Name = strings.TrimSpace(*req.Name)
	}
	if req.BaseURL != nil {
		baseURL := strings.TrimRight(strings.TrimSpace(*req.BaseURL), "/")
		if err := validateExternalHTTPSURL(baseURL); err != nil {
			return nil, err
		}
		cred.BaseURL = baseURL
	}
	if req.TenantID != nil {
		cred.TenantID = strings.TrimSpace(*req.TenantID)
	}
	if req.TenantCode != nil {
		cred.TenantCode = strings.TrimSpace(*req.TenantCode)
	}
	if req.AutomationToken != nil {
		if strings.TrimSpace(*req.AutomationToken) == "" {
			cred.AutomationTokenEnc = nil
		} else {
			enc, err := s.cipher.EncryptString(strings.TrimSpace(*req.AutomationToken))
			if err != nil {
				return nil, err
			}
			cred.AutomationTokenEnc = enc
		}
	}
	if req.ClientID != nil {
		cred.ClientID = strings.TrimSpace(*req.ClientID)
	}
	if req.ClientSecret != nil {
		if strings.TrimSpace(*req.ClientSecret) == "" {
			cred.ClientSecretEnc = nil
		} else {
			enc, err := s.cipher.EncryptString(*req.ClientSecret)
			if err != nil {
				return nil, err
			}
			cred.ClientSecretEnc = enc
		}
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
	if strings.TrimSpace(cred.TenantID) == "" && strings.TrimSpace(cred.TenantCode) == "" {
		return nil, fmt.Errorf("tenant_id or tenant_code is required")
	}
	if len(cred.AutomationTokenEnc) == 0 && (strings.TrimSpace(cred.ClientID) == "" || len(cred.ClientSecretEnc) == 0) && len(cred.ServiceTokenEnc) == 0 {
		return nil, fmt.Errorf("automation_token is required unless client_id/client_secret or service_token are provided")
	}
	return &cred, s.db.WithContext(ctx).Save(&cred).Error
}

func (s *CredentialService) validateBasaltPassDatabaseCredential(ctx context.Context, userID string, dependencyID, credentialID uint, required bool) error {
	if dependencyID == 0 && credentialID == 0 {
		if required {
			return fmt.Errorf("database_dependency_id and database_credential_id are required for managed BasaltPass")
		}
		return nil
	}
	if dependencyID == 0 || credentialID == 0 {
		return fmt.Errorf("database_dependency_id and database_credential_id must be provided together")
	}
	var dep model.ManagedDependency
	if err := s.db.WithContext(ctx).
		Joins("JOIN applications a ON a.id = managed_dependencies.application_id").
		Where("managed_dependencies.id = ? AND a.owner_id = ?", dependencyID, userID).
		First(&dep).Error; err != nil {
		return fmt.Errorf("database dependency not found or not accessible")
	}
	if dep.Type != "mysql" && !isPostgreSQLCompatibleDependency(dep.Type) {
		return fmt.Errorf("BasaltPass database dependency must be mysql, postgresql, or timescaledb")
	}
	var cred model.DependencyCredential
	if err := s.db.WithContext(ctx).
		Where("id = ? AND dependency_id = ?", credentialID, dependencyID).
		First(&cred).Error; err != nil {
		return fmt.Errorf("database credential not found for dependency")
	}
	return nil
}

func isPostgreSQLCompatibleDependency(depType string) bool {
	switch strings.ToLower(strings.TrimSpace(depType)) {
	case "postgresql", "timescaledb":
		return true
	default:
		return false
	}
}

func validateExternalHTTPSURL(raw string) error {
	host, ip, err := validatePublicHTTPSURLSyntax(raw)
	if err != nil {
		return err
	}
	if ip != nil {
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

func validatePublicHTTPSURLSyntax(raw string) (string, net.IP, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return "", nil, fmt.Errorf("base_url must be a valid https URL")
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "", nil, fmt.Errorf("base_url must not target localhost")
	}
	return host, net.ParseIP(host), nil
}
