package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"github.com/zeturn/beancs-controller/internal/service"
)

type APIKeyHandler struct {
	Base
	service *service.APIKeyService
}

func NewAPIKeyHandler(svc *service.APIKeyService, v *validator.Validate) *APIKeyHandler {
	return &APIKeyHandler{Base: NewBase(v), service: svc}
}

func (h *APIKeyHandler) Register(r fiber.Router) {
	r.Get("/api-keys/scopes", h.scopes)
	r.Get("/api-keys", middleware.RequireAPIScope(service.ScopeAPIKeysRead), h.list)
	r.Post("/api-keys", middleware.RequireAPIScope(service.ScopeAPIKeysWrite), h.create)
	r.Delete("/api-keys/:id", middleware.RequireAPIScope(service.ScopeAPIKeysRevoke), h.revoke)
}

func (h *APIKeyHandler) scopes(c *fiber.Ctx) error {
	return c.JSON(h.service.ScopeOptions(middleware.Scopes(c)))
}

func (h *APIKeyHandler) list(c *fiber.Ctx) error {
	out, err := h.service.List(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *APIKeyHandler) create(c *fiber.Ctx) error {
	var req dto.CreateAPIKeyRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.Create(c.UserContext(), middleware.UserID(c), middleware.TenantID(c), middleware.Scopes(c), middleware.AuthMethod(c) == "api_key", req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(out)
}

func (h *APIKeyHandler) revoke(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.service.Revoke(c.UserContext(), middleware.UserID(c), id); err != nil {
		return fail(c, 404, err)
	}
	return c.SendStatus(204)
}
