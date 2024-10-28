package server

import (
	"os"

	"github.com/gofiber/fiber/v2"

	jwtMiddleware "github.com/gofiber/contrib/jwt"
)

// Middleware for routes group with JWT authentication.
// See: https://github.com/gofiber/contrib/jwt
func JWTProtected() func(*fiber.Ctx) error {
	// Create config for JWT authentication middleware.
	config := jwtMiddleware.Config{
		SigningKey:   jwtMiddleware.SigningKey{Key: []byte(os.Getenv("JWT_SECRET_KEY"))},
		ContextKey:   "jwt", // used in private routes
		ErrorHandler: jwtError,
	}

	return jwtMiddleware.New(config)
}

func jwtError(c *fiber.Ctx, err error) error {
	// Return status 400 and failed authentication error.
	if err.Error() == jwtMiddleware.ErrJWTMissingOrMalformed.Error() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": err.Error(),
		})
	}

	// Return status 401 and failed authentication error.
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error":   true,
		"message": "EXPIRED_TOKEN",
	})
}
