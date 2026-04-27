package middleware

import "github.com/gofiber/fiber/v2"

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
		if s == scope {
			return true
		}
	}
	return false
}
