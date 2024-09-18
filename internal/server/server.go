package server

import (
	"github.com/gofiber/fiber/v2"

	"mercuria-backend/internal/database"
)

type FiberServer struct {
	*fiber.App

	db database.Service
}

func New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "mercuria-backend",
			AppName:      "mercuria-backend",
		}),

		db: database.New(),
	}

	return server
}
