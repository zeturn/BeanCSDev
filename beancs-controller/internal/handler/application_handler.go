package handler

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"github.com/zeturn/beancs-controller/internal/service"
)

type ApplicationHandler struct {
	Base
	service *service.ApplicationService
}

func NewApplicationHandler(svc *service.ApplicationService, v *validator.Validate) *ApplicationHandler {
	return &ApplicationHandler{Base: NewBase(v), service: svc}
}

func (h *ApplicationHandler) Register(r fiber.Router) {
	r.Post("/applications/monorepo", middleware.RequireAPIScope(service.ScopeProjectsWrite), h.createMonorepo)
	r.Get("/applications", middleware.RequireAPIScope(service.ScopeProjectsRead), h.list)
	r.Get("/applications/:id", middleware.RequireAPIScope(service.ScopeProjectsRead), h.get)
	r.Delete("/applications/:id", middleware.RequireAPIScope(service.ScopeProjectsDelete), h.delete)
}

func (h *ApplicationHandler) createMonorepo(c *fiber.Ctx) error {
	var req dto.CreateMonorepoApplicationRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if req.TeamID != "" && !middleware.HasScope(c, "beancs.admin") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "team_id requires beancs.admin until team membership checks are enabled"})
	}
	out, err := h.service.CreateMonorepo(c.UserContext(), middleware.UserID(c), middleware.TenantID(c), middleware.TenantCode(c), req)
	if err != nil {
		if out != nil {
			return c.Status(fiber.StatusMultiStatus).JSON(fiber.Map{"data": out, "error": err.Error()})
		}
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(out)
}

func (h *ApplicationHandler) list(c *fiber.Ctx) error {
	out, err := h.service.List(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *ApplicationHandler) get(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid application id"})
	}
	out, err := h.service.Get(c.UserContext(), middleware.UserID(c), uint(id))
	if err != nil {
		return fail(c, 404, err)
	}
	return c.JSON(out)
}

func (h *ApplicationHandler) delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid application id"})
	}
	if err := h.service.Delete(c.UserContext(), middleware.UserID(c), uint(id)); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}
