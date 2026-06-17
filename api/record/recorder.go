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

const detectionInterval = 500 * time.Millisecond

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
	cameraId      int
	rtspLink      string
	withDetection bool
	mtx           sync.Mutex
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	frameCh       capture.FrameSubscriber
	cancel        context.CancelFunc
}

func NewRecordRegistry(cameraId int, rtspLink string, withDetection bool, cmd *exec.Cmd, stdin io.WriteCloser, frameCh capture.FrameSubscriber, cancel context.CancelFunc) *RecordRegistry {
	return &RecordRegistry{
		cameraId: cameraId,
		rtspLink: rtspLink,
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

	dir := fmt.Sprintf("../recordings/%d", cameraId)
	if err := os.MkdirAll(dir, 0755); err != nil {
		capture.Manager.Unsubscribe(cameraId, frameCh)
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	registry := &RecordRegistry{
		cameraId:      cameraId,
		rtspLink:      rtspLink,
		withDetection: withDetection,
		frameCh:       frameCh,
		cancel:        cancel,
	}
	r.activeCameras[cameraId] = registry

	go r.recordingLoop(ctx, registry, width, height, fps, dir)

	log.Println("Camera", cameraId, "started recording")

	return nil
}

func (r *RecordManager) recordingLoop(ctx context.Context, registry *RecordRegistry, width, height int, fps float64, dir string) {
	defer func() {
		capture.Manager.Unsubscribe(registry.cameraId, registry.frameCh)
		log.Printf("[Camera %d] Recording stopped...", registry.cameraId)
	}()

	detector := detection.MainDetector
	if !registry.withDetection {
		detector = nil
	}

	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		waitDuration := nextMidnight.Sub(now)
		log.Printf("[Camera %d] Next day restart in %v", registry.cameraId, waitDuration)

		registry.mtx.Lock()
		cmd, stdin, err := startFFmpeg(registry.cameraId, width, height, fps, dir)
		if err != nil {
			registry.mtx.Unlock()
			log.Printf("[Camera %d] FFmpeg start error: %v", registry.cameraId, err)
			return
		}
		registry.cmd = cmd
		registry.stdin = stdin
		registry.mtx.Unlock()

		var (
			lastDetections []detection.Detection
			lastDetectTime time.Time
			detMtx         sync.Mutex
			isProcessing   bool
		)

		dayEnd := now.Add(waitDuration)

	INNERLOOP:
		for {
			select {
			case <-ctx.Done():
				registry.mtx.Lock()
				if registry.stdin != nil {
					registry.stdin.Close()
				}
				registry.mtx.Unlock()
				if registry.cmd != nil {
					registry.cmd.Wait()
				}

				break INNERLOOP
			case img, ok := <-registry.frameCh:
				if !ok {
					log.Printf("[Camera %d] Frame channel closed", registry.cameraId)
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
					if len(lastDetections) > 0 {
						detection.DrawOverlay(&img, lastDetections, green)
					}
					detMtx.Unlock()
				}

				data, err := img.DataPtrUint8()
				if err == nil {
					registry.mtx.Lock()
					if registry.stdin != nil {
						if _, err := registry.stdin.Write(data); err != nil {
							log.Printf("[Camera %d] Write to FFmpeg stdin error: %v", registry.cameraId, err)
							registry.mtx.Unlock()
							img.Close()
							return
						}
					}
					registry.mtx.Unlock()
				} else {
					log.Printf("[Camera %d] Failed to get raw data: %v", registry.cameraId, err)
				}
				img.Close()

			case <-time.After(waitDuration):
				log.Printf("[Camera %d] Midnight reached, restarting FFmpeg", registry.cameraId)

				registry.mtx.Lock()
				if registry.stdin != nil {
					registry.stdin.Close()
				}
				registry.mtx.Unlock()
				if registry.cmd != nil {
					registry.cmd.Wait()
				}

				break INNERLOOP
			}

			if time.Now().After(dayEnd) {
				log.Printf("[Camera %d] Day ended (fallback), restarting", registry.cameraId)
				registry.mtx.Lock()
				if registry.stdin != nil {
					registry.stdin.Close()
				}
				registry.mtx.Unlock()
				if registry.cmd != nil {
					registry.cmd.Wait()
				}

				break INNERLOOP
			}
		}

		select {
		case <-ctx.Done():
			return
		default:
			continue
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

	delete(r.activeCameras, cameraId)

	// log.Println("Camera", cameraId, "stopped recording")

	return nil
}

func (r *RecordManager) StopAll() {
	r.mtx.Lock()
	cameras := make([]int, 0, len(r.activeCameras))
	for id := range r.activeCameras {
		cameras = append(cameras, id)
	}
	r.mtx.Unlock()

	log.Println("Waiting for recordings to save on disk...")

	for _, id := range cameras {
		r.StopRecording(id)
	}

	log.Println("Stopped all recordings...")
}

func startFFmpeg(cameraId int, width, height int, fps float64, dir string) (*exec.Cmd, io.WriteCloser, error) {
	dateDir := dir + "/" + time.Now().Format("02-01-2006")
	os.MkdirAll(dateDir, 0755)
	outputPattern := dateDir + "/" + "%H-%M-%S.mp4"
	args := []string{
		"-loglevel", "error",
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
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
		"-segment_time_delta", "0.05",
		"-force_key_frames", "expr:gte(t,n_forced*60)",
		"-reset_timestamps", "1",
		"-strftime", "1",
		outputPattern,
	}
	cmd := exec.Command("ffmpeg", args...)
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
			log.Printf("[FFmpeg | Camera %d] %s", cameraId, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("[FFmpeg | Camera %d] stderr read error: %v", cameraId, err)
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
		return 0, 0, 0, ErrTryingToGetVideoInfo
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
