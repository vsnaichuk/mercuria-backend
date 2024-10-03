package server

import (
	"context"
	"os"

	"mercuria-backend/internal/database"

	"github.com/Timothylock/go-signin-with-apple/apple"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/api/idtoken"
)

func (s *FiberServer) RegisterFiberRoutes() {
	s.App.Get("/", s.HelloWorldHandler)

	s.App.Get("/health", s.HealthHandler)

	s.App.Post("/auth/google/verify-id-token", s.GoogleIDTokenHandler)

	s.App.Post("/auth/apple/verify-id-token", s.AppleIDTokenHandler)

	s.App.Get("events/:id", s.GetEvent)

	s.App.Get("events", s.GetEvents)

	s.App.Post("events/create", s.CreateEvent)

}

// -- Auth Handlers

func (s *FiberServer) HelloWorldHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Hello World",
	})
}

func (s *FiberServer) HealthHandler(c *fiber.Ctx) error {
	return c.JSON(s.db.Health())
}

func (s *FiberServer) GoogleIDTokenHandler(c *fiber.Ctx) error {
	var requestBody struct {
		Token string `json:"token"`
	}

	if err := c.BodyParser(&requestBody); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}

	token := requestBody.Token
	if token == "" {
		return ErrResp(c, 400, "Token is required")
	}

	payload, err := idtoken.Validate(context.Background(), token, "")
	if err != nil {
		return ErrResp(c, 401, "Validation Failed")
	}

	return c.JSON(fiber.Map{
		"code": 200,
		"data": s.db.GetOrCreateUser(database.User{
			OAuthId:   payload.Claims["sub"].(string),
			Name:      payload.Claims["name"].(string),
			AvatarUrl: payload.Claims["picture"].(string),
			Email:     payload.Claims["email"].(string),
		}),
	})
}

// TODO: Test
func (s *FiberServer) AppleIDTokenHandler(c *fiber.Ctx) error {
	var requestBody struct {
		Token string `json:"token"`
	}

	if err := c.BodyParser(&requestBody); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}

	token := requestBody.Token
	if token == "" {
		return ErrResp(c, 400, "Token is required")
	}

	key := os.Getenv("APPLE_KEY")
	teamID := os.Getenv("APPLE_TEAM_ID")
	clientID := os.Getenv("APPLE_CLIENT_ID")
	keyID := os.Getenv("APPLE_KEY_ID")

	secret, _ := apple.GenerateClientSecret(key, teamID, clientID, keyID)

	// Generate a new validation client
	client := apple.New()

	var res apple.ValidationResponse
	err := client.VerifyAppToken(context.Background(), apple.AppValidationTokenRequest{
		ClientID:     clientID,
		ClientSecret: secret,
		Code:         token,
	}, &res)

	if err != nil {
		return ErrResp(c, 401, "Validation failed")
	}

	claims, _ := apple.GetClaims(res.IDToken)

	return c.JSON(fiber.Map{
		"code": 200,
		"data": claims,
	})
}

// -- Events Handlers

func (s *FiberServer) GetEvent(c *fiber.Ctx) error {
	id := c.Params("id")

	return c.JSON(fiber.Map{
		"code": 200,
		"data": s.db.GetEvent(id),
	})
}

func (s *FiberServer) GetEvents(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"code": 200,
		"data": s.db.GetEvents(),
	})
}

func (s *FiberServer) CreateEvent(c *fiber.Ctx) error {
	var requestBody struct {
		Name  string `json:"name"`
		Owner string `json:"owner"`
	}

	if err := c.BodyParser(&requestBody); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}

	requiredFields := map[string]string{
		"Name":  requestBody.Name,
		"Owner": requestBody.Owner,
	}

	for field, value := range requiredFields {
		if value == "" {
			return ErrResp(c, 400, "Require "+field)
		}
	}

	id := s.db.CreateEvent(database.Event{
		Name:  requestBody.Name,
		Owner: requestBody.Owner,
	})

	return c.JSON(fiber.Map{
		"code": 200,
		"data": s.db.GetEvent(id),
	})
}



// -- Utils

func ErrResp(c *fiber.Ctx, status int, args ...string) error {
	message := ""
	if len(args) > 0 {
		message = args[0]
	}
	return c.Status(status).JSON(fiber.Map{
		"code":    status,
		"message": message,
	})
}
