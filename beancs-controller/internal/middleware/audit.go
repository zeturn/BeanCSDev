package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zeturn/beancs-controller/internal/model"
	"gorm.io/gorm"
)

func Audit(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			entry := model.AuditLog{
				UserID:    UserID(c),
				Action:    c.Method() + " " + c.Path(),
				Resource:  c.Params("id"),
				Status:    c.Response().StatusCode(),
				IP:        c.IP(),
				UserAgent: string(c.Context().UserAgent()),
				CreatedAt: time.Now().UTC(),
			}
			go func() { _ = db.Create(&entry).Error }()
		}
		return err
	}
}
