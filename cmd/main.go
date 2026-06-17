package main

import (
	"os"
	"vms/api/database"
	"vms/api/detection"
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

	var modelPath = os.Getenv("YOLO_MODEL")
	detection.MainDetector = detection.NewSharedDetector(modelPath, 0.45, 0.5)

	httpHandlers := handlers.NewHTTPHandlers()
	httpServer := server.NewHTTPServer(httpHandlers)

	httpServer.StartServer()
}

// TODO:
// 1. Start streams on startup (if active == 1 in db).
