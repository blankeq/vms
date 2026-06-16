package record

import (
	"bufio"
	"context"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
	"vms/api/capture"
	"vms/api/detection"

	"gocv.io/x/gocv"
)

const (
	frameBufferSize   = 20
	detectionInterval = 300 * time.Millisecond
)

type RecordManager struct {
	mtx           sync.Mutex
	wg            sync.WaitGroup
	activeCameras map[int]*RecordRegistry
}

func NewRecordManager() *RecordManager {
	return &RecordManager{
		activeCameras: make(map[int]*RecordRegistry),
	}
}

type RecordRegistry struct {
	cameraId int
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	frameCh  capture.FrameSubscriber
	cancel   context.CancelFunc
}

func NewRecordRegistry(cameraId int, cmd *exec.Cmd, stdin io.WriteCloser, frameCh capture.FrameSubscriber, cancel context.CancelFunc) *RecordRegistry {
	return &RecordRegistry{
		cameraId: cameraId,
		cmd:      cmd,
		stdin:    stdin,
		frameCh:  frameCh,
		cancel:   cancel,
	}
}

var Manager = NewRecordManager()

func (r *RecordManager) StartRecording(cameraId int, rtspLink string, withDetection bool) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if _, ok := r.activeCameras[cameraId]; ok {
		return ErrAlreadyRecording
	}

	width, height, fps, err := getVideoInfo(rtspLink)
	if err != nil {
		return err
	}

	frameCh, err := capture.Manager.Subscribe(cameraId, rtspLink)
	if err != nil {
		return err
	}

	dir := fmt.Sprintf("./recordings/%d", cameraId)
	if err := os.MkdirAll(dir, 0755); err != nil {
		capture.Manager.Unsubscribe(cameraId, frameCh)
		return err
	}

	var detector *detection.SharedDetector
	if withDetection {
		detector = detection.MainDetector
	} else {
		detector = nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	cmd, stdin, err := startFFmpeg(ctx, cameraId, width, height, fps, dir)
	if err != nil {
		capture.Manager.Unsubscribe(cameraId, frameCh)
		cancel()
		return err
	}

	registry := NewRecordRegistry(cameraId, cmd, stdin, frameCh, cancel)
	r.activeCameras[cameraId] = registry

	go r.recordingLoop(ctx, registry, detector)

	log.Println("Camera", cameraId, "started recording")

	return nil
}

func (r *RecordManager) recordingLoop(ctx context.Context, registry *RecordRegistry, detector *detection.SharedDetector) {
	defer func() {
		registry.stdin.Close()
		if err := registry.cmd.Wait(); err != nil {
			log.Println("FFmpeg finished with error:", err)
		}
		capture.Manager.Unsubscribe(registry.cameraId, registry.frameCh)
	}()

	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	var (
		lastDetections []detection.Detection
		lastDetectTime time.Time
		detMtx         sync.Mutex
		isProcessing   bool
	)

	for {
		select {
		case <-ctx.Done():
			return
		case img, ok := <-registry.frameCh:
			if !ok {
				return
			}

			if detector != nil {
				detMtx.Lock()

				if !isProcessing && time.Since(lastDetectTime) >= detectionInterval {
					isProcessing = true
					lastDetectTime = time.Now()
					detectImg := img.Clone()

					go func(img gocv.Mat) {
						defer img.Close()

						detector.Mtx.Lock()
						detections, _ := detector.Detector.Detect(&img)
						detector.Mtx.Unlock()

						detMtx.Lock()
						lastDetections = detections
						isProcessing = false
						detMtx.Unlock()
					}(detectImg)
				}
				detMtx.Unlock()

				detMtx.Lock()
				if len(lastDetections) > 0 {
					detection.DrawOverlay(&img, lastDetections, green)
				}
				detMtx.Unlock()
			}

			data, err := img.DataPtrUint8()
			if err == nil {
				if _, err := registry.stdin.Write(data); err != nil {
					log.Println("FFmpeg stdin write error:", err)
					return
				}
			} else {
				log.Println("Failed to get raw data:", err)
			}
			img.Close()
		}
	}
}

func (r *RecordManager) StopRecording(cameraId int) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	registry, ok := r.activeCameras[cameraId]
	if !ok {
		return ErrCameraNotActive
	}

	registry.cancel()
	capture.Manager.Unsubscribe(cameraId, registry.frameCh)
	delete(r.activeCameras, cameraId)

	log.Println("Camera", cameraId, "stopped recording")

	return nil
}

func (r *RecordManager) StopAll() {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	for cameraId, registry := range r.activeCameras {
		registry.cancel()
		capture.Manager.Unsubscribe(cameraId, registry.frameCh)
		delete(r.activeCameras, cameraId)
	}

	log.Println("Waiting for recordings to save on disk...")
	r.wg.Wait()
	log.Println("Stopped all recordings...")
}

func startFFmpeg(ctx context.Context, cameraId int, width, height int, fps float64, dir string) (*exec.Cmd, io.WriteCloser, error) {
	outputPattern := dir + "/%d-%m-%Y/%H-%M-%S.mp4"
	args := []string{
		"-loglevel", "error",
		"-f", "rawvideo",
		"-vcodec", "rawvideo",
		"-s", fmt.Sprintf("%dx%d", width, height),
		"-pix_fmt", "bgr24",
		"-r", fmt.Sprintf("%.2f", fps),
		"-i", "pipe:0",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "23",
		"-f", "segment",
		"-segment_time", "60",
		"-reset_timestamps", "1",
		"-strftime", "1",
		outputPattern,
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("[FFmpeg camera %d] %s", cameraId, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("[FFmpeg camera %d] stderr read error: %v", cameraId, err)
		}
	}()

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, stdin, nil
}

func getVideoInfo(rtspLink string) (int, int, float64, error) {
	capture, err := gocv.OpenVideoCapture(rtspLink)
	if err != nil {
		return 0, 0, 0, ErrTryingToOpenVideoCapture
	}
	defer capture.Close()

	width := int(capture.Get(gocv.VideoCaptureFrameWidth))
	height := int(capture.Get(gocv.VideoCaptureFrameHeight))
	fps := capture.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 25.0
	}

	return width, height, fps, nil
}
