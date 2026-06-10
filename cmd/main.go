package main

import (
	"vms/api/database"
	"vms/api/handlers"
	"vms/api/server"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		panic(err)
	}

	if err := database.Init(); err != nil {
		panic(err)
	}

	httpHandlers := handlers.NewHTTPHandlers()
	httpServer := server.NewHTTPServer(httpHandlers)

	httpServer.StartServer()
}

// TODO:
// 1. Fix showSuccess()
// 2. Fix playerContainer size
// 3. Add human detection
// 4. Work on errors, reduce repeatable code segments
