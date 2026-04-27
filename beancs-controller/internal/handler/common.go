package handler

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type Base struct {
	validator *validator.Validate
}

func NewBase(v *validator.Validate) Base { return Base{validator: v} }

func (b Base) parseAndValidate(c *fiber.Ctx, req any) error {
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if err := b.validator.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return nil
}

func idParam(c *fiber.Ctx, name string) (uint, error) {
	v, err := strconv.ParseUint(c.Params(name), 10, 64)
	return uint(v), err
}

func fail(c *fiber.Ctx, status int, err error) error {
	msg := "request failed"
	if err != nil && err.Error() != "" {
		msg = err.Error()
	}
	return c.Status(status).JSON(fiber.Map{"error": msg})
}
