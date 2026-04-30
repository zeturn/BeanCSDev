package handler

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/k8s"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"gorm.io/gorm"
)

type RuntimeHandler struct {
	Base
	db  *gorm.DB
	k8s *k8s.Manager
}

func NewRuntimeHandler(db *gorm.DB, k8sManager *k8s.Manager, v *validator.Validate) *RuntimeHandler {
	return &RuntimeHandler{Base: NewBase(v), db: db, k8s: k8sManager}
}

func (h *RuntimeHandler) Register(r fiber.Router) {
	r.Get("/runtime/overview", h.overview)
	r.Get("/runtime/namespaces", h.namespaces)
	r.Get("/projects/:id/status", middleware.ProjectAccess(h.db), h.status)
	r.Get("/projects/:id/logs", middleware.ProjectAccess(h.db), h.logs)
	r.Post("/projects/:id/restart", middleware.ProjectOwner(h.db), h.restart)
	r.Post("/projects/:id/scale", middleware.ProjectOwner(h.db), h.scale)
}

func (h *RuntimeHandler) namespaces(c *fiber.Ctx) error {
	out, err := h.k8s.ListNamespaces(c.UserContext())
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *RuntimeHandler) overview(c *fiber.Ctx) error {
	out, err := h.k8s.RuntimeOverview(c.UserContext())
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *RuntimeHandler) status(c *fiber.Ctx) error {
	p := projectFromCtx(c)
	pods, err := h.k8s.PodStatus(c.UserContext(), p.Namespace, p.Name)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"pods": pods})
}

func (h *RuntimeHandler) logs(c *fiber.Ctx) error {
	p := projectFromCtx(c)
	tail, _ := strconv.ParseInt(c.Query("tail", "100"), 10, 64)
	out, err := h.k8s.Logs(c.UserContext(), p.Namespace, p.Name, tail)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"logs": out})
}

func (h *RuntimeHandler) restart(c *fiber.Ctx) error {
	p := projectFromCtx(c)
	if err := h.k8s.RestartDeployment(c.UserContext(), p.Namespace, p.Name); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) scale(c *fiber.Ctx) error {
	p := projectFromCtx(c)
	var req dto.ScaleProjectRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.ScaleDeployment(c.UserContext(), p.Namespace, p.Name, int32(req.Replicas)); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
