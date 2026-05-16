package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/basaltpass"
	"github.com/zeturn/beancs-controller/internal/config"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"github.com/zeturn/beancs-controller/internal/model"
	"github.com/zeturn/beancs-controller/internal/service"
	"gorm.io/gorm"
)

type CredentialHandler struct {
	Base
	db       *gorm.DB
	service  *service.CredentialService
	registry *basaltpass.ClientRegistry
	cfg      *config.Config
}

func NewCredentialHandler(db *gorm.DB, svc *service.CredentialService, registry *basaltpass.ClientRegistry, cfg *config.Config, v *validator.Validate) *CredentialHandler {
	return &CredentialHandler{Base: NewBase(v), db: db, service: svc, registry: registry, cfg: cfg}
}

func (h *CredentialHandler) Register(r fiber.Router) {
	h.registerCloudflare(r.Group("/credentials/cloudflare"))
	h.registerGitHub(r.Group("/credentials/github"))
	h.registerBasaltPass(r.Group("/credentials/basaltpass"))
}

func (h *CredentialHandler) registerCloudflare(r fiber.Router) {
	r.Post("/", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.createCloudflare)
	r.Get("/", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.listCloudflare)
	r.Get("/domains", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.listCloudflareDomains)
	r.Get("/:id/dns-records", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.listCloudflareDNSRecords)
	r.Post("/:id/dns-records", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.createCloudflareDNSRecord)
	r.Put("/:id/dns-records/:record_id", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.updateCloudflareDNSRecord)
	r.Delete("/:id/dns-records/:record_id", middleware.RequireAPIScope(service.ScopeCredentialsDelete), h.deleteCloudflareDNSRecord)
	r.Get("/:id", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.getCloudflare)
	r.Patch("/:id", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.updateCloudflare)
	r.Delete("/:id", middleware.RequireAPIScope(service.ScopeCredentialsDelete), h.delete(model.CredentialTypeCloudflare))
	r.Post("/:id/share", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.share(model.CredentialTypeCloudflare))
	r.Delete("/:id/share/:user_id", middleware.RequireAPIScope(service.ScopeCredentialsDelete), h.revoke(model.CredentialTypeCloudflare))
	r.Get("/:id/verify", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.verifyCloudflare)
}

func (h *CredentialHandler) registerGitHub(r fiber.Router) {
	r.Post("/app/start", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.startGitHubAppInstall)
	r.Post("/", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.createGitHub)
	r.Get("/", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.listGitHub)
	r.Get("/:id/repositories", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.listGitHubRepositories)
	r.Get("/:id", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.getGitHub)
	r.Patch("/:id", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.updateGitHub)
	r.Delete("/:id", middleware.RequireAPIScope(service.ScopeCredentialsDelete), h.delete(model.CredentialTypeGitHub))
	r.Post("/:id/share", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.share(model.CredentialTypeGitHub))
	r.Delete("/:id/share/:user_id", middleware.RequireAPIScope(service.ScopeCredentialsDelete), h.revoke(model.CredentialTypeGitHub))
	r.Get("/:id/verify", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.verifyOK)
}

func (h *CredentialHandler) RegisterGitHubAppCallback(r fiber.Router) {
	r.Get("/app/callback", h.githubAppCallback)
}

func (h *CredentialHandler) registerBasaltPass(r fiber.Router) {
	r.Post("/", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.createBasaltPass)
	r.Get("/", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.listBasaltPass)
	r.Get("/:id", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.getBasaltPass)
	r.Patch("/:id", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.updateBasaltPass)
	r.Delete("/:id", middleware.RequireAPIScope(service.ScopeCredentialsDelete), h.delete(model.CredentialTypeBasaltPass))
	r.Post("/:id/share", middleware.RequireAPIScope(service.ScopeCredentialsWrite), h.share(model.CredentialTypeBasaltPass))
	r.Delete("/:id/share/:user_id", middleware.RequireAPIScope(service.ScopeCredentialsDelete), h.revoke(model.CredentialTypeBasaltPass))
	r.Get("/:id/health", middleware.RequireAPIScope(service.ScopeCredentialsRead), h.healthBasaltPass)
}

func (h *CredentialHandler) createCloudflare(c *fiber.Ctx) error {
	var req dto.CreateCloudflareCredentialRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.CreateCloudflare(c.UserContext(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(fiber.Map{"data": out})
}

func (h *CredentialHandler) createGitHub(c *fiber.Ctx) error {
	var req dto.CreateGitHubCredentialRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.CreateGitHub(c.UserContext(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(out)
}

type githubAppState struct {
	UserID     string    `json:"user_id"`
	GitOpsRepo string    `json:"gitops_repo,omitempty"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func (h *CredentialHandler) startGitHubAppInstall(c *fiber.Ctx) error {
	if h.cfg == nil || h.cfg.GitHubAppID == 0 || h.cfg.GitHubAppSlug == "" || strings.TrimSpace(h.cfg.GitHubAppPrivateKey) == "" {
		return fail(c, 400, fmt.Errorf("GitHub App is not configured"))
	}
	var req dto.StartGitHubAppInstallRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	state, err := h.signGitHubAppState(githubAppState{
		UserID:     middleware.UserID(c),
		GitOpsRepo: strings.TrimSpace(req.GitOpsRepo),
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	})
	if err != nil {
		return fail(c, 500, err)
	}
	installURL := fmt.Sprintf("https://github.com/apps/%s/installations/new?state=%s", url.PathEscape(h.cfg.GitHubAppSlug), url.QueryEscape(state))
	return c.JSON(fiber.Map{"install_url": installURL})
}

func (h *CredentialHandler) githubAppCallback(c *fiber.Ctx) error {
	installationID, err := strconv.ParseInt(c.Query("installation_id"), 10, 64)
	if err != nil || installationID <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("GitHub App installation was missing.")
	}
	state, err := h.verifyGitHubAppState(c.Query("state"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("GitHub App state was invalid or expired.")
	}
	account, err := h.service.GitHubAppInstallationAccount(c.UserContext(), installationID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if _, err := h.service.CreateGitHubApp(c.UserContext(), state.UserID, dto.StartGitHubAppInstallRequest{
		GitOpsRepo: state.GitOpsRepo,
	}, installationID, account.Login); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	return c.Redirect("/?github_app=connected", fiber.StatusFound)
}

func (h *CredentialHandler) createBasaltPass(c *fiber.Ctx) error {
	var req dto.CreateBasaltPassCredentialRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.CreateBasaltPass(c.UserContext(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(out)
}

func (h *CredentialHandler) listCloudflare(c *fiber.Ctx) error {
	out, err := h.service.ListCloudflare(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *CredentialHandler) listCloudflareDomains(c *fiber.Ctx) error {
	out, err := h.service.ListCloudflareDomains(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *CredentialHandler) listGitHub(c *fiber.Ctx) error {
	out, err := h.service.ListGitHub(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *CredentialHandler) listBasaltPass(c *fiber.Ctx) error {
	out, err := h.service.ListBasaltPass(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *CredentialHandler) getCloudflare(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeCloudflare, id, false); err != nil {
		return fail(c, 403, err)
	}
	var out model.CloudflareCredential
	if err := h.db.First(&out, id).Error; err != nil {
		return fail(c, 404, err)
	}
	return c.JSON(out)
}

func (h *CredentialHandler) getGitHub(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeGitHub, id, false); err != nil {
		return fail(c, 403, err)
	}
	var out model.GitHubCredential
	if err := h.db.First(&out, id).Error; err != nil {
		return fail(c, 404, err)
	}
	return c.JSON(out)
}

func (h *CredentialHandler) listGitHubRepositories(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeGitHub, id, false); err != nil {
		return fail(c, 403, err)
	}
	out, err := h.service.ListGitHubRepositories(c.UserContext(), id)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *CredentialHandler) getBasaltPass(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeBasaltPass, id, false); err != nil {
		return fail(c, 403, err)
	}
	var out model.BasaltPassInstance
	if err := h.db.First(&out, id).Error; err != nil {
		return fail(c, 404, err)
	}
	return c.JSON(out)
}

func (h *CredentialHandler) updateCloudflare(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeCloudflare, id, true); err != nil {
		return fail(c, 403, err)
	}
	var req dto.UpdateCloudflareCredentialRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.UpdateCloudflare(c.UserContext(), id, req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(out)
}

func (h *CredentialHandler) updateGitHub(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeGitHub, id, true); err != nil {
		return fail(c, 403, err)
	}
	var req dto.UpdateGitHubCredentialRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.UpdateGitHub(c.UserContext(), id, req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(out)
}

func (h *CredentialHandler) updateBasaltPass(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeBasaltPass, id, true); err != nil {
		return fail(c, 403, err)
	}
	var req dto.UpdateBasaltPassCredentialRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.UpdateBasaltPass(c.UserContext(), id, req)
	if err != nil {
		return fail(c, 400, err)
	}
	h.registry.Invalidate(id)
	return c.JSON(out)
}

func (h *CredentialHandler) delete(typ string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := idParam(c, "id")
		if err != nil {
			return fail(c, 400, err)
		}
		if err := h.service.Delete(c.UserContext(), middleware.UserID(c), typ, id); err != nil {
			return fail(c, 400, err)
		}
		if typ == model.CredentialTypeBasaltPass {
			h.registry.Invalidate(id)
		}
		return c.SendStatus(204)
	}
}

func (h *CredentialHandler) share(typ string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := idParam(c, "id")
		if err != nil {
			return fail(c, 400, err)
		}
		var req dto.ShareCredentialRequest
		if err := h.parseAndValidate(c, &req); err != nil {
			return err
		}
		if err := h.service.Share(c.UserContext(), middleware.UserID(c), typ, id, req); err != nil {
			return fail(c, 403, err)
		}
		return c.JSON(fiber.Map{"status": "ok"})
	}
}

func (h *CredentialHandler) revoke(typ string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := idParam(c, "id")
		if err != nil {
			return fail(c, 400, err)
		}
		if err := h.service.Revoke(c.UserContext(), middleware.UserID(c), typ, id, c.Params("user_id")); err != nil {
			return fail(c, 403, err)
		}
		return c.SendStatus(204)
	}
}

func (h *CredentialHandler) verifyCloudflare(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeCloudflare, id, false); err != nil {
		return fail(c, 403, err)
	}
	out, err := h.service.VerifyCloudflare(c.UserContext(), id)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(out)
}

func (h *CredentialHandler) listCloudflareDNSRecords(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeCloudflare, id, false); err != nil {
		return fail(c, 403, err)
	}
	out, err := h.service.ListCloudflareDNSRecords(c.UserContext(), id, c.Query("zone_id"))
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *CredentialHandler) createCloudflareDNSRecord(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeCloudflare, id, true); err != nil {
		return fail(c, 403, err)
	}
	var req dto.CreateCloudflareDNSRecordRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.CreateCloudflareDNSRecord(c.UserContext(), id, c.Query("zone_id"), req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(out)
}

func (h *CredentialHandler) updateCloudflareDNSRecord(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeCloudflare, id, true); err != nil {
		return fail(c, 403, err)
	}
	var req dto.UpdateCloudflareDNSRecordRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.UpdateCloudflareDNSRecord(c.UserContext(), id, c.Query("zone_id"), c.Params("record_id"), req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(out)
}

func (h *CredentialHandler) deleteCloudflareDNSRecord(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeCloudflare, id, true); err != nil {
		return fail(c, 403, err)
	}
	if err := h.service.DeleteCloudflareDNSRecord(c.UserContext(), id, c.Query("zone_id"), c.Params("record_id")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *CredentialHandler) verifyOK(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *CredentialHandler) healthBasaltPass(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.RequireAccess(middleware.UserID(c), model.CredentialTypeBasaltPass, id, false); err != nil {
		return fail(c, 403, err)
	}
	client, err := h.registry.GetClientForInstance(id)
	if err != nil {
		return fail(c, 400, err)
	}
	out, err := client.HealthCheck(c.UserContext())
	if err != nil {
		return fail(c, 502, err)
	}
	return c.JSON(out)
}

func (h *CredentialHandler) signGitHubAppState(state githubAppState) (string, error) {
	body, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	sig := hmacSHA256(payload, h.cfg.WebhookSecret)
	return payload + "." + sig, nil
}

func (h *CredentialHandler) verifyGitHubAppState(raw string) (*githubAppState, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid state")
	}
	expected := hmacSHA256(parts[0], h.cfg.WebhookSecret)
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return nil, fmt.Errorf("invalid state signature")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var state githubAppState
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, err
	}
	if state.UserID == "" || time.Now().After(state.ExpiresAt) {
		return nil, fmt.Errorf("invalid state payload")
	}
	return &state, nil
}

func hmacSHA256(value, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
