package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/basaltpass"
	"github.com/zeturn/beancs-controller/internal/service"
)

type tokenCacheEntry struct {
	info      *basaltpass.IntrospectionResult
	expiresAt time.Time
}

func Auth(registry *basaltpass.ClientRegistry, apiKeys *service.APIKeyService) fiber.Handler {
	var mu sync.RWMutex
	cache := map[string]tokenCacheEntry{}

	return func(c *fiber.Ctx) error {
		token := authToken(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}
		if apiKeys != nil && strings.HasPrefix(token, "bcs_") {
			identity, ok, err := apiKeys.Authenticate(c.UserContext(), token)
			if err != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "api key auth unavailable"})
			}
			if !ok {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid api key"})
			}
			setAPIKeyLocals(c, identity)
			return c.Next()
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

func authToken(c *fiber.Ctx) string {
	if token := bearerToken(c.Get("Authorization")); token != "" {
		return token
	}
	return strings.TrimSpace(c.Get("X-API-Key"))
}

func setAuthLocals(c *fiber.Ctx, info *basaltpass.IntrospectionResult) {
	c.Locals("user_id", info.Sub)
	c.Locals("tenant_id", info.TenantID)
	c.Locals("tenant_code", info.TenantCode)
	c.Locals("scopes", strings.Fields(info.Scope))
	c.Locals("auth_method", "basaltpass")
	if info.Act != nil {
		c.Locals("actor", info.Act)
	}
}

func setAPIKeyLocals(c *fiber.Ctx, identity *service.APIKeyIdentity) {
	c.Locals("user_id", identity.UserID)
	c.Locals("tenant_id", identity.TenantID)
	c.Locals("tenant_code", identity.TenantID)
	c.Locals("scopes", identity.Scopes)
	c.Locals("auth_method", "api_key")
	c.Locals("api_key_id", identity.KeyID)
	c.Locals("api_key_name", identity.KeyName)
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

func TenantCode(c *fiber.Ctx) string {
	if v, ok := c.Locals("tenant_code").(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return TenantID(c)
}

func Scopes(c *fiber.Ctx) []string {
	scopes, _ := c.Locals("scopes").([]string)
	return scopes
}
