package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"github.com/zeturn/beancs-controller/internal/service"
)

type ApplicationSpecHandler struct {
	Base
	service *service.ApplicationSpecService
}

func NewApplicationSpecHandler(svc *service.ApplicationSpecService, v *validator.Validate) *ApplicationSpecHandler {
	return &ApplicationSpecHandler{Base: NewBase(v), service: svc}
}

func (h *ApplicationSpecHandler) Register(r fiber.Router) {
	r.Post("/application-specs/validate", middleware.RequireAPIScope(service.ScopeProjectsRead), h.validate)
	r.Post("/application-specs/plan", middleware.RequireAPIScope(service.ScopeProjectsRead), h.plan)
	r.Post("/application-specs/apply", middleware.RequireAPIScope(service.ScopeProjectsWrite), h.apply)
	r.Post("/applications/from-repo-config", middleware.RequireAPIScope(service.ScopeProjectsWrite), h.fromRepoConfig)
}

func (h *ApplicationSpecHandler) validate(c *fiber.Ctx) error {
	var req dto.ApplicationSpecRepoRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.ValidateFromRepo(c.UserContext(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, service.ApplicationSpecHTTPStatus(err), err)
	}
	return c.JSON(out)
}

func (h *ApplicationSpecHandler) plan(c *fiber.Ctx) error {
	var req dto.ApplicationSpecRepoRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.PlanFromRepo(c.UserContext(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, service.ApplicationSpecHTTPStatus(err), err)
	}
	return c.JSON(out)
}

func (h *ApplicationSpecHandler) apply(c *fiber.Ctx) error {
	return h.applyRequest(c)
}

func (h *ApplicationSpecHandler) fromRepoConfig(c *fiber.Ctx) error {
	return h.applyRequest(c)
}

func (h *ApplicationSpecHandler) applyRequest(c *fiber.Ctx) error {
	var req dto.ApplicationSpecRepoRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	specResp, appResp, err := h.service.ApplyFromRepo(c.UserContext(), middleware.UserID(c), middleware.TenantID(c), middleware.TenantCode(c), req)
	if err != nil {
		if specResp != nil {
			return c.Status(service.ApplicationSpecHTTPStatus(err)).JSON(fiber.Map{"spec": specResp, "application": appResp, "error": err.Error()})
		}
		return fail(c, service.ApplicationSpecHTTPStatus(err), err)
	}
	return c.JSON(fiber.Map{"spec": specResp, "application": appResp})
}
