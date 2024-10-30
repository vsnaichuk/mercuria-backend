package server

import (
	"mercuria-backend/internal/database"
	"mercuria-backend/internal/storage"

	redis "github.com/go-redis/redis/v7"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	s3 "github.com/gofiber/storage/s3/v2"
)

type FiberServer struct {
	*fiber.App

	db database.Service

	redis *redis.Client

	storage *s3.Storage
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

	// Init Redis on database 1 - it's used to store the JWT
	Redis := database.NewRedis(1)

	// Init S3 bucket to store images
	Storage := storage.New()

	return &FiberServer{
		App:     App,
		db:      DB,
		redis:   Redis,
		storage: Storage,
	}
}
