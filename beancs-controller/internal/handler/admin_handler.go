package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/k8s"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

type AdminHandler struct {
	Base
	db  *gorm.DB
	k8s *k8s.Manager
}

func NewAdminHandler(db *gorm.DB, k8sManager *k8s.Manager, v *validator.Validate) *AdminHandler {
	return &AdminHandler{Base: NewBase(v), db: db, k8s: k8sManager}
}

func (h *AdminHandler) Register(r fiber.Router) {
	r.Get("/admin/overview", h.overview)
	r.Get("/admin/nodes", h.nodes)
	r.Get("/admin/quotas", h.quotas)
	r.Patch("/admin/quotas/:team_id", h.updateQuota)
}

func (h *AdminHandler) overview(c *fiber.Ctx) error {
	var projects, deployments int64
	_ = h.db.Model(&model.Project{}).Count(&projects).Error
	_ = h.db.Model(&model.Deployment{}).Count(&deployments).Error
	nodes, _ := h.k8s.Nodes(c.UserContext())
	return c.JSON(fiber.Map{"projects": projects, "deployments": deployments, "nodes": len(nodes)})
}

func (h *AdminHandler) nodes(c *fiber.Ctx) error {
	out, err := h.k8s.Nodes(c.UserContext())
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *AdminHandler) quotas(c *fiber.Ctx) error {
	var out []model.ResourceQuota
	if err := h.db.Find(&out).Error; err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *AdminHandler) updateQuota(c *fiber.Ctx) error {
	var req map[string]int
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	var q model.ResourceQuota
	if err := h.db.FirstOrCreate(&q, model.ResourceQuota{TeamID: c.Params("team_id")}).Error; err != nil {
		return fail(c, 400, err)
	}
	updates := map[string]any{}
	for _, key := range []string{"max_projects", "max_cpu_millis", "max_memory_mb"} {
		if v, ok := req[key]; ok {
			updates[key] = v
		}
	}
	if len(updates) > 0 {
		if err := h.db.Model(&q).Updates(updates).Error; err != nil {
			return fail(c, 400, err)
		}
	}
	return c.JSON(q)
}
