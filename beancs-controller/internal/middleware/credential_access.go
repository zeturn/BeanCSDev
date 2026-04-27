package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

func CredentialAccess(db *gorm.DB, credentialType string, ownerOnly bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.ParseUint(c.Params("id"), 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid credential id"})
		}
		var uc model.UserCredential
		err = db.Where("user_id = ? AND credential_type = ? AND credential_id = ?", UserID(c), credentialType, uint(id)).First(&uc).Error
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "credential access denied"})
		}
		if ownerOnly && uc.Role != model.CredentialRoleOwner {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "credential owner required"})
		}
		return c.Next()
	}
}
