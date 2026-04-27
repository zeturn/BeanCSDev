package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/basaltpass"
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
}

func NewCredentialHandler(db *gorm.DB, svc *service.CredentialService, registry *basaltpass.ClientRegistry, v *validator.Validate) *CredentialHandler {
	return &CredentialHandler{Base: NewBase(v), db: db, service: svc, registry: registry}
}

func (h *CredentialHandler) Register(r fiber.Router) {
	h.registerCloudflare(r.Group("/credentials/cloudflare"))
	h.registerGitHub(r.Group("/credentials/github"))
	h.registerBasaltPass(r.Group("/credentials/basaltpass"))
}

func (h *CredentialHandler) registerCloudflare(r fiber.Router) {
	r.Post("/", h.createCloudflare)
	r.Get("/", h.listCloudflare)
	r.Get("/:id", h.getCloudflare)
	r.Patch("/:id", h.updateCloudflare)
	r.Delete("/:id", h.delete(model.CredentialTypeCloudflare))
	r.Post("/:id/share", h.share(model.CredentialTypeCloudflare))
	r.Delete("/:id/share/:user_id", h.revoke(model.CredentialTypeCloudflare))
	r.Get("/:id/verify", h.verifyCloudflare)
}

func (h *CredentialHandler) registerGitHub(r fiber.Router) {
	r.Post("/", h.createGitHub)
	r.Get("/", h.listGitHub)
	r.Get("/:id", h.getGitHub)
	r.Patch("/:id", h.updateGitHub)
	r.Delete("/:id", h.delete(model.CredentialTypeGitHub))
	r.Post("/:id/share", h.share(model.CredentialTypeGitHub))
	r.Delete("/:id/share/:user_id", h.revoke(model.CredentialTypeGitHub))
	r.Get("/:id/verify", h.verifyOK)
}

func (h *CredentialHandler) registerBasaltPass(r fiber.Router) {
	r.Post("/", h.createBasaltPass)
	r.Get("/", h.listBasaltPass)
	r.Get("/:id", h.getBasaltPass)
	r.Patch("/:id", h.updateBasaltPass)
	r.Delete("/:id", h.delete(model.CredentialTypeBasaltPass))
	r.Post("/:id/share", h.share(model.CredentialTypeBasaltPass))
	r.Delete("/:id/share/:user_id", h.revoke(model.CredentialTypeBasaltPass))
	r.Get("/:id/health", h.healthBasaltPass)
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
	return c.Status(201).JSON(out)
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
