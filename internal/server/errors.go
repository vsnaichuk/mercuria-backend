package server

import "github.com/gofiber/fiber/v2"

func ErrResp(c *fiber.Ctx, status int, args ...string) error {
	message := ""
	if len(args) > 0 {
		message = args[0]
	}
	return c.Status(status).JSON(fiber.Map{
		"error":   true,
		"message": message,
	})
}
