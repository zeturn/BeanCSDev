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
	r.Post("/runtime/namespaces", h.createNamespace)
	r.Patch("/runtime/namespaces/:name", h.patchNamespace)
	r.Delete("/runtime/namespaces/:name", h.deleteNamespace)
	r.Get("/runtime/pods/:namespace/:name/logs", h.podLogs)
	r.Delete("/runtime/pods/:namespace/:name", h.deletePod)
	r.Post("/runtime/services", h.createService)
	r.Put("/runtime/services/:namespace/:name", h.updateService)
	r.Delete("/runtime/services/:namespace/:name", h.deleteService)
	r.Get("/projects/:id/status", middleware.ProjectAccess(h.db), h.status)
	r.Get("/projects/:id/logs", middleware.ProjectAccess(h.db), h.logs)
	r.Post("/projects/:id/restart", middleware.ProjectOwner(h.db), h.restart)
	r.Post("/projects/:id/scale", middleware.ProjectOwner(h.db), h.scale)
}

func (h *RuntimeHandler) createNamespace(c *fiber.Ctx) error {
	var req dto.CreateNamespaceRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.CreateNamespaceWithLabels(c.UserContext(), req.Name, req.Labels); err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) patchNamespace(c *fiber.Ctx) error {
	var req dto.RuntimeLabelPatchRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.PatchNamespaceLabels(c.UserContext(), c.Params("name"), req.Labels); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) deleteNamespace(c *fiber.Ctx) error {
	if err := h.k8s.DeleteNamespace(c.UserContext(), c.Params("name")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *RuntimeHandler) podLogs(c *fiber.Ctx) error {
	tail, _ := strconv.ParseInt(c.Query("tail", "160"), 10, 64)
	out, err := h.k8s.PodLogs(c.UserContext(), c.Params("namespace"), c.Params("name"), tail)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"logs": out})
}

func (h *RuntimeHandler) deletePod(c *fiber.Ctx) error {
	if err := h.k8s.DeletePod(c.UserContext(), c.Params("namespace"), c.Params("name")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
}

func (h *RuntimeHandler) createService(c *fiber.Ctx) error {
	var req dto.CreateServiceRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.k8s.UpsertService(c.UserContext(), req); err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) updateService(c *fiber.Ctx) error {
	var req dto.UpdateServiceRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	createReq := dto.CreateServiceRequest{
		Namespace: c.Params("namespace"),
		Name:      c.Params("name"),
		Type:      req.Type,
		Selector:  req.Selector,
		Ports:     req.Ports,
		Labels:    req.Labels,
	}
	if err := h.k8s.UpsertService(c.UserContext(), createReq); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RuntimeHandler) deleteService(c *fiber.Ctx) error {
	if err := h.k8s.DeleteService(c.UserContext(), c.Params("namespace"), c.Params("name")); err != nil {
		return fail(c, 400, err)
	}
	return c.SendStatus(204)
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
	out, err := h.k8s.ProjectRuntimeStatus(c.UserContext(), p.Namespace, p.Name)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(out)
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
