package main

import (
	"time"
	"vms/api/record"
)

func main() {
	go record.Manager.StartRecording(1, "rtsp://admin:123456@192.168.1.2:554/stream0")

	time.Sleep(8 * time.Second)

	record.Manager.StopRecording(1)

	time.Sleep(5 * time.Second)
}
