package handler

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/middleware"
	"github.com/zeturn/beancs-controller/internal/service"
)

type ContainerRegistryHandler struct {
	Base
	svc *service.ContainerRegistryService
}

func NewContainerRegistryHandler(svc *service.ContainerRegistryService, v *validator.Validate) *ContainerRegistryHandler {
	return &ContainerRegistryHandler{Base: NewBase(v), svc: svc}
}

func (h *ContainerRegistryHandler) Register(r fiber.Router) {
	r.Get("/container-registries/presets", middleware.RequireAPIScope(service.ScopeRegistriesRead), h.presets)
	r.Get("/container-registries", middleware.RequireAPIScope(service.ScopeRegistriesRead), h.listRegistries)
	r.Post("/container-registries", middleware.RequireAPIScope(service.ScopeRegistriesWrite), h.createRegistry)
	r.Patch("/container-registries/:id", middleware.RequireAPIScope(service.ScopeRegistriesWrite), h.updateRegistry)
	r.Delete("/container-registries/:id", middleware.RequireAPIScope(service.ScopeRegistriesDelete), h.deleteRegistry)
	r.Get("/container-registries/:id/tags", middleware.RequireAPIScope(service.ScopeRegistriesRead), h.listTagsLive)

	r.Get("/container-images", middleware.RequireAPIScope(service.ScopeRegistriesRead), h.listImages)
	r.Post("/container-images", middleware.RequireAPIScope(service.ScopeRegistriesWrite), h.createImage)
	r.Post("/container-images/:id/refresh", middleware.RequireAPIScope(service.ScopeRegistriesWrite), h.refreshImage)
	r.Delete("/container-images/:id", middleware.RequireAPIScope(service.ScopeRegistriesDelete), h.deleteImage)
}

func (h *ContainerRegistryHandler) presets(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"data": h.svc.Presets()})
}

func (h *ContainerRegistryHandler) listRegistries(c *fiber.Ctx) error {
	out, err := h.svc.ListRegistries(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *ContainerRegistryHandler) createRegistry(c *fiber.Ctx) error {
	var req dto.CreateContainerRegistryRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.svc.Create(c.UserContext(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(out)
}

func (h *ContainerRegistryHandler) updateRegistry(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	var req dto.UpdateContainerRegistryRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.svc.Update(c.UserContext(), middleware.UserID(c), id, req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(out)
}

func (h *ContainerRegistryHandler) deleteRegistry(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.svc.Delete(c.UserContext(), middleware.UserID(c), id); err != nil {
		return fail(c, 404, err)
	}
	return c.SendStatus(204)
}

func (h *ContainerRegistryHandler) listTagsLive(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	repo := strings.TrimSpace(c.Query("repository"))
	if repo == "" {
		return fail(c, 400, fmt.Errorf("repository query required"))
	}
	out, err := h.svc.ListTagsLive(c.UserContext(), middleware.UserID(c), id, repo)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *ContainerRegistryHandler) listImages(c *fiber.Ctx) error {
	out, err := h.svc.ListImages(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return fail(c, 500, err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *ContainerRegistryHandler) createImage(c *fiber.Ctx) error {
	var req dto.CreateContainerImageRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	out, err := h.svc.CreateImage(c.UserContext(), middleware.UserID(c), req)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.Status(201).JSON(out)
}

func (h *ContainerRegistryHandler) refreshImage(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	out, err := h.svc.RefreshImage(c.UserContext(), middleware.UserID(c), id)
	if err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(out)
}

func (h *ContainerRegistryHandler) deleteImage(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, 400, err)
	}
	if err := h.svc.DeleteImage(c.UserContext(), middleware.UserID(c), id); err != nil {
		return fail(c, 404, err)
	}
	return c.SendStatus(204)
}
