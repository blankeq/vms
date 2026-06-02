package record

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

type Recorder struct {
	mtx           sync.Mutex
	activeCameras map[int]context.CancelFunc
}

func NewRecorder() *Recorder {
	return &Recorder{
		mtx:           sync.Mutex{},
		activeCameras: make(map[int]context.CancelFunc),
	}
}

func (r *Recorder) StartRecording(cameraId int, rtspLink string) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if _, ok := r.activeCameras[cameraId]; ok {
		return ErrAlreadyRecording
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.activeCameras[cameraId] = cancel

	go func(ctx context.Context) {
		dir := fmt.Sprintf("./recordings/%d", cameraId)
		fileFormat := dir + "/%d-%m-%Y/%H-%M-%S.mp4"
		os.MkdirAll(dir, os.ModePerm)

		cmd := exec.CommandContext(
			ctx, "ffmpeg", "-rtcp_transport", "tcp", "-i", rtspLink, "-f", "segment",
			"-segment_time", "60", "-strftime", "1", "-reset_timestamps", "1", "-c", "copy",
			fileFormat,
		)

		fmt.Println("Camera", cameraId, "started recording")

		if err := cmd.Run(); err != nil {
			fmt.Println("Camera", cameraId, "is stopped or an error occured:", err)
		}
	}(ctx)

	return nil
}

func (r *Recorder) StopRecording(cameraId int) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	cancel, ok := r.activeCameras[cameraId]
	if !ok {
		return ErrCameraNotActive
	}

	cancel()
	delete(r.activeCameras, cameraId)
	fmt.Println("Camera", cameraId, "stopped recording")

	return nil
}
