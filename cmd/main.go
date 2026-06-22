package main

import (
	"fmt"
	"os"
	"strconv"
	"vms/api/database"
	"vms/api/detection"
	"vms/api/handlers"
	"vms/api/notification"
	"vms/api/record"
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
	scoreThreshold, err := strconv.ParseFloat(os.Getenv("YOLO_SCORE_THRESHOLD"), 32)
	if err != nil {
		scoreThreshold = 0.45
	}

	nmsThreshold, err := strconv.ParseFloat(os.Getenv("YOLO_NMS_THRESHOLD"), 64)
	if err != nil {
		nmsThreshold = 0.5
	}
	detection.MainDetector = detection.NewSharedDetector(modelPath, scoreThreshold, nmsThreshold)

	var timeoutCapture = os.Getenv("TIMEOUT_CAPTURE")
	os.Setenv("OPENCV_FFMPEG_CAPTURE_OPTIONS", fmt.Sprintf("rtsp_transport;udp|stimeout;%s|timeout;%s", timeoutCapture, timeoutCapture))

	record.RecordingsDir = os.Getenv("RECORDINGS_DIR")
	if record.RecordingsDir == "" {
		record.RecordingsDir = "../recordings"
	}

	var smptServer = os.Getenv("SMTP_SERVER")
	var smtpUsername = os.Getenv("SMTP_USERNAME")
	var smtpPassword = os.Getenv("SMTP_PASSWORD")

	notification.MailClient, err = notification.NewMailClient(smptServer, smtpUsername, smtpPassword)
	if err != nil {
		panic(err)
	}

	httpHandlers := handlers.NewHTTPHandlers()
	httpServer := server.NewHTTPServer(httpHandlers)

	httpServer.StartServer()
}
