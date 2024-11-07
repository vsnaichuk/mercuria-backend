package server

import (
	"context"
	"fmt"
	"io"
	"os"

	"mercuria-backend/internal/database"

	"github.com/Timothylock/go-signin-with-apple/apple"
	"github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v4"
	"google.golang.org/api/idtoken"
)

func PublicRoutes(s *FiberServer) {
	route := s.App.Group("/api/v1")

	route.Get("/", s.HelloWorldHandler)
	route.Get("/health", s.HealthHandler)
	route.Get("/auth/logout", s.Logout)
	route.Post("/auth/google/login", s.GoogleLoginHandler) // oauth2, return Access & Refresh tokens
	route.Post("/auth/apple/login", s.AppleLoginHandler)   // oauth2, return Access & Refresh tokens
	route.Post("/auth/refresh-token", s.RefreshToken)
}

func PrivateRoutes(s *FiberServer) {
	route := s.App.Group("/api/v1")

	route.Get("events/:id", JWTProtected(), s.GetEvent)
	route.Get("events/user/:id", JWTProtected(), s.GetUserEvents)
	route.Post("events/create", JWTProtected(), s.CreateEvent)
	route.Post("events/like", JWTProtected(), s.LikeEvent)
	route.Post("events/create-invite", JWTProtected(), s.CreateEventInvite)
	route.Post("events/verify-invite", JWTProtected(), s.VerifyEventInvite)
	route.Post("events/upload-photos", JWTProtected(), s.UploadPhotos)
	route.Delete("events/dislike", JWTProtected(), s.DislikeEvent)
}

func (s *FiberServer) RegisterFiberRoutes() {
	PublicRoutes(s)
	PrivateRoutes(s)
}

func (s *FiberServer) HelloWorldHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Hello World",
	})
}

func (s *FiberServer) HealthHandler(c *fiber.Ctx) error {
	return c.JSON(s.db.Health())
}

func (s *FiberServer) RefreshToken(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrResp(c, 400, "Body parse error")
	}

	//is expired
	token, err := jwt.Parse(body.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("REFRESH_SECRET")), nil
	})
	if err != nil {
		return ErrResp(c, 401, "Invalid authorization, please login again")
	}
	//is token valid?
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return ErrResp(c, 401, "Invalid authorization, please login again")
	}
	//the token claims should conform to MapClaims
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		refreshUUID, ok := claims["refresh_uuid"].(string)
		if !ok {
			return ErrResp(c, 401, "Invalid authorization, please login again")
		}
		userID, ok := claims["user_id"].(string)
		if !ok {
			return ErrResp(c, 401, "Invalid authorization, please login again")
		}
		//delete the previous Refresh Token
		deleted, delErr := s.DeleteAuth(refreshUUID)
		if delErr != nil || deleted == 0 { //if any goes wrong
			return ErrResp(c, 401, "Invalid authorization, please login again")
		}
		//create new fresh tokens
		ts, createErr := CreateToken(userID)
		if createErr != nil {
			return ErrResp(c, 403, "Invalid authorization, please login again")
		}
		//save the tokens metadata to redis
		if err := s.CreateAuth(userID, ts); err != nil {
			return ErrResp(c, 403, "Save Token Details error", err)
		}
		return c.JSON(fiber.Map{
			"access_token":  ts.AccessToken,
			"refresh_token": ts.RefreshToken,
		})
	} else {
		return ErrResp(c, 401, "Invalid authorization, please login again")
	}
}

func (s *FiberServer) Logout(c *fiber.Ctx) error {
	au, err := ExtractTokenMetadata(c)
	if err != nil {
		return ErrResp(c, 400, "User not logged in")
	}

	deleted, delErr := s.DeleteAuth(au.AccessUUID)
	if delErr != nil || deleted == 0 { //if any goes wrong
		return ErrResp(c, 401, "Invalid request")
	}

	return c.JSON(fiber.Map{
		"message": "Successfully logged out",
	})
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
		return ErrResp(c, 400, "`id_token` is required")
	}

	payload, err := idtoken.Validate(context.Background(), body.IdToken, "")
	if err != nil {
		return ErrResp(c, 401, "Validation Failed")
	}

	// TODO: handle error and return 500
	user := s.db.GetOrCreateUser(database.User{
		OAuthId:   payload.Claims["sub"].(string),
		Name:      payload.Claims["name"].(string),
		AvatarUrl: payload.Claims["picture"].(string),
		Email:     payload.Claims["email"].(string),
	})
	userId := user["id"]

	// Create JWT and save to Redis
	tokenDetails, err := CreateToken(userId)
	if err != nil {
		return ErrResp(c, 500, "Create Token error", err)
	}
	if err := s.CreateAuth(userId, tokenDetails); err != nil {
		return ErrResp(c, 500, "Save Token Details error", err)
	}

	// TODO: handle error and return 500
	if body.Invite != "" {
		s.db.AddEventMember(userId, body.Invite)
	}

	return c.JSON(fiber.Map{
		"access_token":  tokenDetails.AccessToken,
		"refresh_token": tokenDetails.RefreshToken,
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
		return ErrResp(c, 400, "`id_token` is required")
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
		return ErrResp(c, 400, "Required `user_id` and `event_id`")
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
		return ErrResp(c, 400, "Required `user_id` and `event_id`")
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
		return ErrResp(c, 400, "Required `event_id` and `created_by`")
	}

	// Look at better ways to Create Invite
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
		return ErrResp(c, 400, "Required `user_id` and `invite`")
	}

	// TODO: Add better verification process

	return c.JSON(fiber.Map{
		"message": s.db.AddEventMember(body.UserId, body.Invite),
	})
}

func (s *FiberServer) UploadPhotos(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return ErrResp(c, 400, "Form data parse error")
	}

	createdByValues, creatorOk := form.Value["created_by"]
	eventIdValues, eventOk := form.Value["event_id"]
	if !creatorOk || !eventOk {
		return ErrResp(c, 400, "Required `created_by` and `event_id`")
	}

	files := form.File["photos"]
	for i, file := range files {
		createdBy := createdByValues[i]
		eventId := eventIdValues[i]
		if createdBy == "" || eventId == "" {
			return ErrResp(c, 400, "Required `created_by` and `event_id`")
		}

		fmt.Println(file.Filename, file.Size, file.Header["Content-Type"])
		// => "photo.jpeg" 160037 "image/jpeg"

		src, err := file.Open()
		if err != nil {
			return ErrResp(c, 500, "Open file error", err)
		}
		defer src.Close()

		fileBytes, _ := io.ReadAll(src)
		fileName := file.Filename
		fileType := file.Header.Get("Content-Type")

		photo := &database.Photo{
			ID:        UUID().String(),
			CreatedBy: createdBy,
			FileName:  fileName,
			FileType:  fileType,
			EventID:   eventId,
		}

		output, err := s.storage.UploadFile(fileBytes, photo.ID, fileType)
		if err != nil {
			return ErrResp(c, 500, "Upload file to storage error", err)
		}

		photo.PublicUrl = output.Location

		if err := s.db.CreatePhoto(photo); err != nil {
			return ErrResp(c, 500, err.Error())
		}
	}

	return c.JSON(fiber.Map{
		"message": "success",
	})
}
