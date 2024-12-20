package main

import (
	"fmt"
	"mercuria-backend/internal/server"
	"os"
	"strconv"

	_ "github.com/joho/godotenv/autoload"
)

func main() {

	server := server.New()

	server.RegisterFiberRoutes()
	host, _ := strconv.Atoi(os.Getenv("HOST"))
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	err := server.Listen(fmt.Sprintf("%d:%d", host, port))
	if err != nil {
		panic(fmt.Sprintf("cannot start server: %s", err))
	}
}
