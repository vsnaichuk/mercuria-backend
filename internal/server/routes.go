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

	s.App.Post("/auth/google/login", s.GoogleLoginHandler)

	s.App.Post("/auth/apple/login", s.AppleLoginHandler)

	s.App.Get("events/:id", s.GetEvent)

	s.App.Get("events/user/:id", s.GetUserEvents)

	s.App.Post("events/create", s.CreateEvent)

	s.App.Post("events/like", s.LikeEvent)

	s.App.Delete("events/dislike", s.DislikeEvent)

	// s.App.Post("events/create-invite", s.CreateEventInvite)
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

func (s *FiberServer) GoogleLoginHandler(c *fiber.Ctx) error {
	var body struct {
		Token  string `json:"token"`
		Invite string `json:"invite"`
	}

	if err := c.BodyParser(&body); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}
	if body.Token == "" {
		return ErrResp(c, 400, "Token is required")
	}

	payload, err := idtoken.Validate(context.Background(), body.Token, "")
	if err != nil {
		return ErrResp(c, 401, "Validation Failed")
	}

	user := s.db.GetOrCreateUser(database.User{
		OAuthId:   payload.Claims["sub"].(string),
		Name:      payload.Claims["name"].(string),
		AvatarUrl: payload.Claims["picture"].(string),
		Email:     payload.Claims["email"].(string),
	})

	// TODO: What if user already logged in??
	if body.Invite != "" {
		userId := user["id"]
		s.db.AddEventMember(userId, body.Invite)
	}

	return c.JSON(fiber.Map{
		"code": 200,
		"data": user,
	})
}

// TODO: Test
func (s *FiberServer) AppleLoginHandler(c *fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}
	if body.Token == "" {
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
		Code:         body.Token,
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
	event, _ := s.db.GetEvent(id)
	return c.JSON(fiber.Map{
		"code": 200,
		"data": event,
	})
}

func (s *FiberServer) GetUserEvents(c *fiber.Ctx) error {
	userId := c.Params("id")
	return c.JSON(fiber.Map{
		"code": 200,
		"data": s.db.GetUserEvents(userId),
	})
}

func (s *FiberServer) CreateEvent(c *fiber.Ctx) error {
	var body struct {
		Name    string `json:"name"`
		OwnerID string `json:"owner"`
	}

	if err := c.BodyParser(&body); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}

	requiredFields := map[string]string{
		"Name":    body.Name,
		"OwnerID": body.OwnerID,
	}

	for field, value := range requiredFields {
		if value == "" {
			return ErrResp(c, 400, "Require "+field)
		}
	}

	id := s.db.CreateEvent(database.Event{
		Name:    body.Name,
		OwnerID: body.OwnerID,
	})

	event, _ := s.db.GetEvent(id)

	return c.JSON(fiber.Map{
		"code": 200,
		"data": event,
	})
}

func (s *FiberServer) LikeEvent(c *fiber.Ctx) error {
	var body struct {
		UserId  string `json:"user_id"`
		EventId string `json:"event_id"`
	}

	if err := c.BodyParser(&body); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}
	if body.UserId == "" || body.EventId == "" {
		return ErrResp(c, 400, "Required UserId and EventId")
	}

	return c.JSON(fiber.Map{
		"code":    200,
		"message": s.db.LikeEvent(body.UserId, body.EventId),
	})
}

func (s *FiberServer) DislikeEvent(c *fiber.Ctx) error {
	var body struct {
		UserId  string `json:"user_id"`
		EventId string `json:"event_id"`
	}

	if err := c.BodyParser(&body); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}
	if body.UserId == "" || body.EventId == "" {
		return ErrResp(c, 400, "Required UserId and EventId")
	}

	return c.JSON(fiber.Map{
		"code":    200,
		"message": s.db.DislikeEvent(body.UserId, body.EventId),
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

// Look at better ways to Create Invite
// var secret = []byte(os.Getenv("JWT_SECRET"))

// func generateInviteToken(eventID string, createdBy string) (string, error) {
// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
// 		"event_id":   eventID,
// 		"created_by": createdBy,
// 	})
// 	fmt.Println(token)
// 	return token.SignedString(secret)
// }

// func parseInviteToken(tokenString string) (string, string) {
// 	claims := jwt.MapClaims{}
// 	token, _ := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
// 		return secret, nil
// 	})
// 	println(token)
// 	return claims["event_id"].(string), claims["created_by"].(string)
// }
//
// // func (s *FiberServer) CreateEventInvite(c *fiber.Ctx) error {
// 	var body struct {
// 		EventId   string `json:"event_id"`
// 		CreatedBy string `json:"created_by"`
// 	}

// 	if err := c.BodyParser(&body); err != nil {
// 		return ErrResp(c, 400, "Body parse error")
// 	}
// 	if body.CreatedBy == "" || body.EventId == "" {
// 		return ErrResp(c, 400, "Required EventId and CreatedBy")
// 	}

// 	// token, err := generateInviteToken(body.EventId, body.CreatedBy)
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }

// 	return c.JSON(fiber.Map{
// 		"code": 200,
// 		"data": fiber.Map{
// 			"invite": body.EventId,
// 		},
// 	})
// }
