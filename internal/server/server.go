package server

import (
	"mercuria-backend/internal/database"

	redis "github.com/go-redis/redis/v7"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type FiberServer struct {
	*fiber.App

	redis *redis.Client

	db database.Service
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

	// Init Redis on database 1 - it's used to store the JWT
	Redis := database.NewRedis(1)

	// Init postgres database
	DB := database.New()

	return &FiberServer{
		App:   App,
		redis: Redis,
		db:    DB,
	}
}
