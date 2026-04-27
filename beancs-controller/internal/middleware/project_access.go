package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

func ProjectAccess(db *gorm.DB) fiber.Handler {
	return projectGuard(db, false)
}

func ProjectOwner(db *gorm.DB) fiber.Handler {
	return projectGuard(db, true)
}

func projectGuard(db *gorm.DB, ownerOnly bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := strconv.ParseUint(c.Params("id"), 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid project id"})
		}
		var project model.Project
		if err := db.First(&project, uint(id)).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
		}
		userID := UserID(c)
		if project.OwnerID == userID || HasScope(c, "beancs.admin") {
			c.Locals("project", &project)
			return c.Next()
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "project access denied"})
	}
}
