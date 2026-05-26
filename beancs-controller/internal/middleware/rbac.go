package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/service"
)

func RequireScope(scope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if HasScope(c, scope) {
			return c.Next()
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient scope"})
	}
}

func HasScope(c *fiber.Ctx, scope string) bool {
	scopes, _ := c.Locals("scopes").([]string)
	for _, s := range scopes {
		if scopeMatches(s, scope) {
			return true
		}
	}
	return false
}

func RequireAPIScope(scope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if AuthMethod(c) != "api_key" || HasScope(c, scope) {
			return c.Next()
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "api key scope required", "scope": scope})
	}
}

func scopeMatches(granted, required string) bool {
	granted = strings.TrimSpace(granted)
	required = strings.TrimSpace(required)
	if granted == "" || required == "" {
		return false
	}
	if granted == required || granted == service.ScopeAdmin {
		return true
	}
	if granted == service.ScopeLegacyAPI && required != service.ScopeAdmin {
		return true
	}
	if strings.HasSuffix(granted, ":*") {
		return strings.HasPrefix(required, strings.TrimSuffix(granted, "*"))
	}
	return false
}
