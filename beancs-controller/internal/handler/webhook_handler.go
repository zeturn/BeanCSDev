package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/dto"
	"github.com/zeturn/beancs-controller/internal/service"
)

type WebhookHandler struct {
	Base
	deployments *service.DeploymentService
}

func NewWebhookHandler(svc *service.DeploymentService, v *validator.Validate) *WebhookHandler {
	return &WebhookHandler{Base: NewBase(v), deployments: svc}
}

func (h *WebhookHandler) Register(r fiber.Router) {
	r.Post("/webhooks/github", h.github)
	r.Post("/webhooks/argocd", h.argocd)
}

func (h *WebhookHandler) github(c *fiber.Ctx) error {
	var req dto.GitHubWebhookRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.deployments.HandleGitHubWebhook(c.UserContext(), req); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *WebhookHandler) argocd(c *fiber.Ctx) error {
	var req dto.ArgoCDWebhookRequest
	if err := h.parseAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.deployments.HandleArgoCDWebhook(c.UserContext(), req); err != nil {
		return fail(c, 400, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
