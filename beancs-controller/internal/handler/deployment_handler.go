package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"github.com/zeturn/beancs-controller/internal/model"
	"github.com/zeturn/beancs-controller/internal/service"
	"gorm.io/gorm"
)

type DeploymentHandler struct {
	Base
	db      *gorm.DB
	service *service.DeploymentService
}

func NewDeploymentHandler(db *gorm.DB, svc *service.DeploymentService, v *validator.Validate) *DeploymentHandler {
	return &DeploymentHandler{Base: NewBase(v), db: db, service: svc}
}

func (h *DeploymentHandler) Register(r fiber.Router) {
	r.Post("/projects/:id/deployments", middleware.ProjectAccess(h.db), h.create)
	r.Get("/projects/:id/deployments", middleware.ProjectAccess(h.db), h.list)
	r.Get("/projects/:id/deployments/:did", middleware.ProjectAccess(h.db), h.get)
	r.Get("/projects/:id/deployments/:did/logs", middleware.ProjectAccess(h.db), h.logs)
	r.Post("/projects/:id/deployments/:did/rollback", middleware.ProjectOwner(h.db), h.rollback)
}

func (h *DeploymentHandler) create(c *fiber.Ctx) error {
	var req dto.CreateDeploymentRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	project := projectFromCtx(c)
	out, err := h.service.Create(c.UserContext(), project.ID, req.Tag, req.CommitSHA, middleware.UserID(c))
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(out)
}

func (h *DeploymentHandler) list(c *fiber.Ctx) error {
	project := projectFromCtx(c)
	out, err := h.service.List(c.UserContext(), project.ID)
	if err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *DeploymentHandler) get(c *fiber.Ctx) error {
	project := projectFromCtx(c)
	did, err := idParam(c, "did")
	if err != nil {
		return fail(c, 400, err)
	}
	var out model.Deployment
	if err := h.db.Where("project_id = ? AND id = ?", project.ID, did).First(&out).Error; err != nil {
		return fail(c, 404, err)
	}
	return c.JSON(out)
}

func (h *DeploymentHandler) logs(c *fiber.Ctx) error {
	project := projectFromCtx(c)
	did, err := idParam(c, "did")
	if err != nil {
		return fail(c, 400, err)
	}
	out, err := h.service.Logs(c.UserContext(), *project, did)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"logs": out})
}

func (h *DeploymentHandler) rollback(c *fiber.Ctx) error {
	project := projectFromCtx(c)
	did, err := idParam(c, "did")
	if err != nil {
		return fail(c, 400, err)
	}
	dep := model.Deployment{ProjectID: project.ID, Status: "deploying", TriggeredBy: middleware.UserID(c), Tag: "rollback"}
	if err := h.db.First(&model.Deployment{}, "project_id = ? AND id = ?", project.ID, did).Error; err != nil {
		return fail(c, 404, err)
	}
	if err := h.db.Create(&dep).Error; err != nil {
		return fail(c, 400, err)
	}
	return c.Status(202).JSON(dep)
}
