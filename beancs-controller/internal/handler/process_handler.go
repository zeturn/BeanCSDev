package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"github.com/zeturn/beancs-controller/internal/service"
	"gorm.io/gorm"
)

type ProcessHandler struct {
	db      *gorm.DB
	service *service.ProcessService
}

func NewProcessHandler(db *gorm.DB, svc *service.ProcessService) *ProcessHandler {
	return &ProcessHandler{db: db, service: svc}
}

func (h *ProcessHandler) Register(r fiber.Router) {
	r.Get("/processes", middleware.RequireAPIScope(service.ScopeProcessesRead), h.list)
	r.Get("/processes/:id", middleware.RequireAPIScope(service.ScopeProcessesRead), h.get)
}

func (h *ProcessHandler) list(c *fiber.Ctx) error {
	out, err := h.service.List(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *ProcessHandler) get(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	out, err := h.service.Get(c.UserContext(), id)
	if err != nil {
		return fail(c, 404, err)
	}
	userID := middleware.UserID(c)
	if out.ProjectID != 0 && out.Project.OwnerID != userID && !middleware.HasScope(c, "beancs.admin") {
		return fail(c, fiber.StatusForbidden, errors.New("forbidden"))
	}
	if out.ProjectID == 0 && out.OwnerID != userID && !middleware.HasScope(c, "beancs.admin") {
		return fail(c, fiber.StatusForbidden, errors.New("forbidden"))
	}
	return c.JSON(out)
}
