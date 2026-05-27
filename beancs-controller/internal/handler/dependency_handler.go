package handler

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"github.com/zeturn/beancs-controller/internal/service"
)

type DependencyHandler struct {
	Base
	service *service.DependencyService
}

func NewDependencyHandler(svc *service.DependencyService, v *validator.Validate) *DependencyHandler {
	return &DependencyHandler{Base: NewBase(v), service: svc}
}

func (h *DependencyHandler) Register(r fiber.Router) {
	r.Get("/dependency-definitions", middleware.RequireAPIScope(service.ScopeProjectsRead), h.listDefinitions)
	r.Get("/dependency-definitions/:name", middleware.RequireAPIScope(service.ScopeProjectsRead), h.getDefinition)
	r.Get("/dependencies", middleware.RequireAPIScope(service.ScopeProjectsRead), h.listReusableDependencies)
	r.Post("/dependencies", middleware.RequireAPIScope(service.ScopeProjectsWrite), h.createStandaloneDependency)
	r.Get("/applications/:id/dependencies", middleware.RequireAPIScope(service.ScopeProjectsRead), h.listDependencies)
	r.Post("/applications/:id/dependencies", middleware.RequireAPIScope(service.ScopeProjectsWrite), h.createDependency)
	r.Get("/dependencies/:id/credentials", middleware.RequireAPIScope(service.ScopeProjectsRead), h.listCredentials)
	r.Post("/dependencies/:id/credentials", middleware.RequireAPIScope(service.ScopeProjectsWrite), h.createCredential)
	r.Post("/projects/:id/dependencies", middleware.RequireAPIScope(service.ScopeProjectsWrite), h.linkProjectDependency)
}

func (h *DependencyHandler) listDefinitions(c *fiber.Ctx) error {
	defs := h.service.Registry().List()
	out := make([]dto.DependencyDefinitionSummary, 0, len(defs))
	for _, def := range defs {
		out = append(out, dto.DependencyDefinitionSummary{
			Name:                   def.Metadata.Name,
			DisplayName:            def.Metadata.DisplayName,
			Category:               def.Metadata.Category,
			Type:                   def.Spec.Type,
			SupportedDeployMethods: def.Spec.SupportedDeployMethods,
			DefaultDeployMethod:    def.Spec.DefaultDeployMethod,
		})
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *DependencyHandler) getDefinition(c *fiber.Ctx) error {
	def, ok := h.service.Registry().Get(c.Params("name"))
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "dependency definition not found"})
	}
	return c.JSON(def)
}

func (h *DependencyHandler) listDependencies(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid application id"})
	}
	out, err := h.service.List(c.UserContext(), middleware.UserID(c), id)
	if err != nil {
		return fail(c, 404, err)
	}
	return c.JSON(fiber.Map{"data": h.service.MaskList(out)})
}

func (h *DependencyHandler) listReusableDependencies(c *fiber.Ctx) error {
	out, err := h.service.ListReusable(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": h.service.MaskList(out)})
}

func (h *DependencyHandler) createDependency(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid application id"})
	}
	var req dto.CreateManagedDependencyRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.Create(c.UserContext(), middleware.UserID(c), id, req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(fiber.StatusCreated).JSON(h.service.Mask(*out))
}

func (h *DependencyHandler) createStandaloneDependency(c *fiber.Ctx) error {
	var req dto.CreateManagedDependencyRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.CreateStandalone(c.UserContext(), middleware.UserID(c), middleware.TenantID(c), middleware.TenantCode(c), req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(fiber.StatusCreated).JSON(h.service.Mask(*out))
}

func (h *DependencyHandler) listCredentials(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid dependency id"})
	}
	out, err := h.service.ListCredentials(c.UserContext(), middleware.UserID(c), id)
	if err != nil {
		return fail(c, 404, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *DependencyHandler) createCredential(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid dependency id"})
	}
	var req dto.CreateDependencyCredentialRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.CreateCredential(c.UserContext(), middleware.UserID(c), id, req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(fiber.StatusCreated).JSON(h.service.MaskCredential(*out))
}

func (h *DependencyHandler) linkProjectDependency(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
	}
	var req dto.LinkProjectDependencyRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.LinkProject(c.UserContext(), middleware.UserID(c), id, req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(out)
}

func parseUintParam(c *fiber.Ctx, name string) (uint, error) {
	id, err := strconv.ParseUint(c.Params(name), 10, 64)
	return uint(id), err
}
