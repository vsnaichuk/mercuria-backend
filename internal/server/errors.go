package server

import "github.com/gofiber/fiber/v2"

func ErrResp(c *fiber.Ctx, status int, msg string, err ...error) error {
	details := ""
	if len(err) > 0 {
		details = err[0].Error()
	}
	return c.Status(status).JSON(fiber.Map{
		"error":   true,
		"message": msg,
		"details": details,
	})
}
