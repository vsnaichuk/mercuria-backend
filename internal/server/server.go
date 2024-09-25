package server

import (
	"mercuria-backend/internal/database"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type FiberServer struct {
	*fiber.App

	db database.Service
}

func New() *FiberServer {
	App := fiber.New(fiber.Config{
		ServerHeader: "mercuria-backend",
		AppName:      "mercuria-backend",
	})

	App.Use(cors.New(cors.Config{
		AllowOrigins:     "*", //@TODO For security set: 
													 // os.Getenv("CLIENT_URL") and AllowCredentials: true
		AllowCredentials: false,
		AllowHeaders:     "Content-Type, Content-Length, Accept-Encoding, Authorization, accept, origin",
		AllowMethods:     "POST, OPTIONS, GET, PUT",
		ExposeHeaders:    "Set-Cookie",
	}))

	return &FiberServer{
		App: App,
		db:  database.New(),
	}
}
