package handler

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/k8s"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"github.com/zeturn/beancs-controller/internal/model"
	"github.com/zeturn/beancs-controller/internal/service"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type ProjectHandler struct {
	Base
	db      *gorm.DB
	service *service.ProjectService
	k8s     *k8s.Manager
}

func NewProjectHandler(db *gorm.DB, svc *service.ProjectService, k8sManager *k8s.Manager, v *validator.Validate) *ProjectHandler {
	return &ProjectHandler{Base: NewBase(v), db: db, service: svc, k8s: k8sManager}
}

func (h *ProjectHandler) Register(r fiber.Router) {
	r.Post("/projects/analyze", middleware.RequireAPIScope(service.ScopeProjectsRead), h.analyze)
	r.Post("/projects", middleware.RequireAPIScope(service.ScopeProjectsWrite), h.create)
	r.Get("/projects", middleware.RequireAPIScope(service.ScopeProjectsRead), h.list)
	r.Get("/projects/:id", middleware.RequireAPIScope(service.ScopeProjectsRead), middleware.ProjectAccess(h.db), h.get)
	r.Patch("/projects/:id", middleware.RequireAPIScope(service.ScopeProjectsWrite), middleware.ProjectAccess(h.db), h.update)
	r.Delete("/projects/:id", middleware.RequireAPIScope(service.ScopeProjectsDelete), middleware.ProjectOwner(h.db), h.delete)
	r.Get("/projects/:id/env", middleware.RequireAPIScope(service.ScopeProjectsRead), middleware.ProjectAccess(h.db), h.getEnv)
	r.Put("/projects/:id/env", middleware.RequireAPIScope(service.ScopeProjectsWrite), middleware.ProjectOwner(h.db), h.setEnv)
	r.Patch("/projects/:id/env", middleware.RequireAPIScope(service.ScopeProjectsWrite), middleware.ProjectOwner(h.db), h.patchEnv)
	r.Get("/projects/:id/dns", middleware.RequireAPIScope(service.ScopeProjectsRead), middleware.ProjectAccess(h.db), h.dns)
}

func (h *ProjectHandler) analyze(c *fiber.Ctx) error {
	var req dto.AnalyzeProjectRepositoryRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.AnalyzeRepository(c.UserContext(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(out)
}

func (h *ProjectHandler) create(c *fiber.Ctx) error {
	var req dto.CreateProjectRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if req.TeamID != "" && !middleware.HasScope(c, "beancs.admin") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "team_id requires beancs.admin until team membership checks are enabled"})
	}
	out, err := h.service.CreateProject(c.UserContext(), middleware.UserID(c), middleware.TenantID(c), middleware.TenantCode(c), req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(out)
}

func (h *ProjectHandler) list(c *fiber.Ctx) error {
	out, err := h.service.ListProjects(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *ProjectHandler) get(c *fiber.Ctx) error {
	return c.JSON(projectFromCtx(c))
}

func (h *ProjectHandler) update(c *fiber.Ctx) error {
	var req dto.UpdateProjectRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.service.UpdateProject(c.UserContext(), projectFromCtx(c), req, middleware.UserID(c))
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(out)
}

func (h *ProjectHandler) delete(c *fiber.Ctx) error {
	if err := h.service.DeleteProject(c.UserContext(), projectFromCtx(c)); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *ProjectHandler) getEnv(c *fiber.Ctx) error {
	project := projectFromCtx(c)
	out, err := h.k8s.SecretData(c.UserContext(), project.Namespace, "app-env-vars")
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *ProjectHandler) setEnv(c *fiber.Ctx) error {
	project := projectFromCtx(c)
	var req map[string]string
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	current, _ := h.k8s.SecretPlainData(c.UserContext(), project.Namespace, "app-env-vars")
	next := mergeMaskedSecretValues(current, req)
	if err := h.k8s.UpsertSecret(c.UserContext(), project.Namespace, "app-env-vars", project.Name, next); err != nil {
		return fail(c, 400, err)
	}
	if err := h.k8s.RestartDeployment(c.UserContext(), project.Namespace, project.Name); err != nil && !apierrors.IsNotFound(err) {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *ProjectHandler) patchEnv(c *fiber.Ctx) error {
	project := projectFromCtx(c)
	current, err := h.k8s.SecretPlainData(c.UserContext(), project.Namespace, "app-env-vars")
	if err != nil {
		return fail(c, 400, err)
	}
	var req map[string]string
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	for k, v := range req {
		if strings.TrimSpace(k) == "" {
			continue
		}
		if v == "********" {
			continue
		}
		current[k] = v
	}
	if err := h.k8s.UpsertSecret(c.UserContext(), project.Namespace, "app-env-vars", project.Name, current); err != nil {
		return fail(c, 400, err)
	}
	if err := h.k8s.RestartDeployment(c.UserContext(), project.Namespace, project.Name); err != nil && !apierrors.IsNotFound(err) {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func mergeMaskedSecretValues(current, requested map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range requested {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if value == "********" {
			if currentValue, ok := current[key]; ok {
				out[key] = currentValue
			}
			continue
		}
		out[key] = value
	}
	return out
}

func (h *ProjectHandler) dns(c *fiber.Ctx) error {
	project := projectFromCtx(c)
	var out []model.DNSRecord
	if err := h.db.Where("project_id = ?", project.ID).Find(&out).Error; err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func projectFromCtx(c *fiber.Ctx) *model.Project {
	if p, ok := c.Locals("project").(*model.Project); ok {
		return p
	}
	return &model.Project{}
}
