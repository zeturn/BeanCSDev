package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/basaltpass"
)

type tokenCacheEntry struct {
	info      *basaltpass.IntrospectionResult
	expiresAt time.Time
}

func Auth(registry *basaltpass.ClientRegistry) fiber.Handler {
	var mu sync.RWMutex
	cache := map[string]tokenCacheEntry{}

	return func(c *fiber.Ctx) error {
		token := bearerToken(c.Get("Authorization"))
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}

		mu.RLock()
		if entry, ok := cache[token]; ok && time.Now().Before(entry.expiresAt) {
			mu.RUnlock()
			setAuthLocals(c, entry.info)
			return c.Next()
		}
		mu.RUnlock()

		client, err := registry.GetManagementClient()
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "auth service unavailable"})
		}
		info, err := client.IntrospectToken(c.UserContext(), token)
		if err != nil || info == nil || !info.Active {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		expiresAt := time.Now().Add(30 * time.Second)
		if info.Exp > 0 {
			tokenExpiry := time.Unix(info.Exp, 0)
			if !tokenExpiry.After(time.Now()) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
			}
			if tokenExpiry.Before(expiresAt) {
				expiresAt = tokenExpiry
			}
		}
		mu.Lock()
		cache[token] = tokenCacheEntry{info: info, expiresAt: expiresAt}
		mu.Unlock()
		setAuthLocals(c, info)
		return c.Next()
	}
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func setAuthLocals(c *fiber.Ctx, info *basaltpass.IntrospectionResult) {
	c.Locals("user_id", info.Sub)
	c.Locals("tenant_id", info.TenantID)
	c.Locals("scopes", strings.Fields(info.Scope))
	if info.Act != nil {
		c.Locals("actor", info.Act)
	}
}

func UserID(c *fiber.Ctx) string {
	if v, ok := c.Locals("user_id").(string); ok {
		return v
	}
	return ""
}

func TenantID(c *fiber.Ctx) string {
	if v, ok := c.Locals("tenant_id").(string); ok {
		return v
	}
	return ""
}
