package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func WebhookVerify(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sig := c.Get("X-Webhook-Secret")
		if sig == "" {
			sig = c.Get("X-Hub-Signature-256")
		}
		if !VerifyHMACSHA256(c.Body(), sig, secret) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "invalid signature"})
		}
		return c.Next()
	}
}

func VerifyHMACSHA256(body []byte, signature, secret string) bool {
	signature = strings.TrimPrefix(strings.TrimSpace(signature), "sha256=")
	if signature == "" || secret == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	got, err := hex.DecodeString(signature)
	if err == nil {
		return hmac.Equal(got, expected)
	}
	return hmac.Equal([]byte(signature), []byte(hex.EncodeToString(expected)))
}
