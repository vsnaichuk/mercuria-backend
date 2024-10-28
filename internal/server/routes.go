package server

import (
	"context"
	"os"

	"mercuria-backend/internal/database"

	"github.com/Timothylock/go-signin-with-apple/apple"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/api/idtoken"
)

func PublicRoutes(s *FiberServer) {
	route := s.App.Group("/api/v1")

	route.Get("/", s.HelloWorldHandler)
	route.Get("/health", s.HealthHandler)

	route.Post("/auth/google/login", s.GoogleLoginHandler) // oauth2, return Access & Refresh tokens
	route.Post("/auth/apple/login", s.AppleLoginHandler)   // oauth2, return Access & Refresh tokens
}

func PrivateRoutes(s *FiberServer) {
	route := s.App.Group("/api/v1")

	route.Get("events/:id", JWTProtected(), s.GetEvent)
	route.Get("events/user/:id", JWTProtected(), s.GetUserEvents)

	route.Post("events/create", JWTProtected(), s.CreateEvent)
	route.Post("events/like", JWTProtected(), s.LikeEvent)
	route.Post("events/create-invite", JWTProtected(), s.CreateEventInvite)
	route.Post("events/verify-invite", JWTProtected(), s.VerifyEventInvite)

	route.Delete("events/dislike", JWTProtected(), s.DislikeEvent)
}

func (s *FiberServer) RegisterFiberRoutes() {
	PublicRoutes(s)
	PrivateRoutes(s)
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
		IdToken string `json:"id_token"`
		Invite  string `json:"invite"`
	}

	if err := c.BodyParser(&body); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}
	if body.IdToken == "" {
		return ErrResp(c, 400, "Token is required")
	}

	payload, err := idtoken.Validate(context.Background(), body.IdToken, "")
	if err != nil {
		return ErrResp(c, 401, "Validation Failed")
	}

	user := s.db.GetOrCreateUser(database.User{
		OAuthId:   payload.Claims["sub"].(string),
		Name:      payload.Claims["name"].(string),
		AvatarUrl: payload.Claims["picture"].(string),
		Email:     payload.Claims["email"].(string),
	})

	tokens, err := GenerateNewTokens(user["id"])
	if err != nil {
		return ErrResp(c, 500, "Token generation error")
	}

	if body.Invite != "" {
		userId := user["id"]
		s.db.AddEventMember(userId, body.Invite)
	}

	return c.JSON(fiber.Map{
		"access_token":  tokens.Access,
		"refresh_token": tokens.Refresh,
		"user":          user,
	})
}

// TODO: Test
func (s *FiberServer) AppleLoginHandler(c *fiber.Ctx) error {
	var body struct {
		IdToken string `json:"id_token"`
		Invite  string `json:"invite"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}
	if body.IdToken == "" {
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
		Code:         body.IdToken,
	}, &res)

	if err != nil {
		return ErrResp(c, 401, "Validation failed")
	}

	claims, _ := apple.GetClaims(res.IDToken)

	return c.JSON(fiber.Map{
		"data": claims,
	})
}

// -- Events Handlers

func (s *FiberServer) GetEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	event, _ := s.db.GetEvent(id)
	return c.JSON(fiber.Map{
		"data": event,
	})
}

func (s *FiberServer) GetUserEvents(c *fiber.Ctx) error {
	userId := c.Params("id")
	return c.JSON(fiber.Map{
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
		"message": s.db.DislikeEvent(body.UserId, body.EventId),
	})
}

func (s *FiberServer) CreateEventInvite(c *fiber.Ctx) error {
	var body struct {
		EventId   string `json:"event_id"`
		CreatedBy string `json:"created_by"`
	}

	if err := c.BodyParser(&body); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}
	if body.CreatedBy == "" || body.EventId == "" {
		return ErrResp(c, 400, "Required EventId and CreatedBy")
	}

	// token, err := generateInviteToken(body.EventId, body.CreatedBy)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"invite": body.EventId,
		},
	})
}

func (s *FiberServer) VerifyEventInvite(c *fiber.Ctx) error {
	var body struct {
		UserId string `json:"user_id"`
		Invite string `json:"invite"`
	}

	if err := c.BodyParser(&body); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}
	if body.UserId == "" || body.Invite == "" {
		return ErrResp(c, 400, "Required UserId and Invite")
	}

	// TODO: Add better verification process

	return c.JSON(fiber.Map{
		"message": s.db.AddEventMember(body.UserId, body.Invite),
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
