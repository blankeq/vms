package main

import (
	"fmt"
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

	fmt.Println("HTTP Server started...")

	if err := httpServer.StartServer(); err != nil {
		fmt.Println("HTTP Server error:", err)
	}

	fmt.Println("HTTP Server closed...")
}
