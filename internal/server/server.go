package server

import (
	"mercuria-backend/internal/database"
	"mercuria-backend/internal/redis"
	"mercuria-backend/internal/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type FiberServer struct {
	*fiber.App

	db database.Service

	redis redis.Service

	storage storage.Service
}

func New() *FiberServer {
	App := fiber.New(fiber.Config{
		ServerHeader: "mercuria-backend",
		AppName:      "mercuria-backend",
	})

	App.Use(cors.New(cors.Config{
		AllowOrigins: "*", //@TODO For security set:
		// os.Getenv("CLIENT_URL") and AllowCredentials: true
		AllowCredentials: false,
		AllowHeaders:     "Content-Type, Content-Length, Accept-Encoding, Authorization, accept, origin",
		AllowMethods:     "POST, OPTIONS, GET, PUT, DELETE",
		ExposeHeaders:    "Set-Cookie",
	}))

	// Init postgres database
	DB := database.New()

	// Init Redis - it's used to store the JWT
	Redis := redis.New()

	// Init S3 bucket to store images
	Storage := storage.New()

	return &FiberServer{
		App:     App,
		db:      DB,
		redis:   Redis,
		storage: Storage,
	}
}
