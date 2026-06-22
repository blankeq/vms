package stream

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"os"
	"sync"
	"time"
	"vms/api/capture"
	"vms/api/detection"
	"vms/api/notification"
	"vms/api/record"

	"github.com/hybridgroup/mjpeg"
	"gocv.io/x/gocv"
)

const detectionInterval = 500 * time.Millisecond
const timeoutCapture = 5 * time.Second

type StreamManager struct {
	mtx           sync.Mutex
	activeStreams map[int]*StreamRegistry
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		activeStreams: make(map[int]*StreamRegistry, 0),
	}
}

var Manager = NewStreamManager()

type StreamRegistry struct {
	stream  *mjpeg.Stream
	frameCh capture.FrameSubscriber
	cancel  context.CancelFunc
}

func NewStreamRegistry(stream *mjpeg.Stream, frameCh capture.FrameSubscriber, cancel context.CancelFunc) *StreamRegistry {
	return &StreamRegistry{
		stream:  stream,
		frameCh: frameCh,
		cancel:  cancel,
	}
}

func (s *StreamManager) StartStream(cameraId int, rtspLink string, withDetection bool) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, ok := s.activeStreams[cameraId]
	if ok {
		return ErrStreamAlreadyExist
	}

	frameCh, err := capture.Manager.Subscribe(cameraId, rtspLink)
	if err != nil {
		return err
	}

	var detector *detection.SharedDetector
	if withDetection {
		detector = detection.MainDetector
	} else {
		detector = nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	stream := mjpeg.NewStreamWithContext(ctx)

	registry := NewStreamRegistry(stream, frameCh, cancel)
	s.activeStreams[cameraId] = registry

	go s.streamLoop(ctx, cameraId, frameCh, stream, detector)

	log.Printf("[Camera %d] Started streaming...", cameraId)

	return nil
}

func (s *StreamManager) streamLoop(ctx context.Context, cameraId int, frameCh capture.FrameSubscriber, stream *mjpeg.Stream, detector *detection.SharedDetector) {
	defer func() {
		log.Printf("[Camera %d] Stopped streaming...", cameraId)
	}()

	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	var (
		lastDetections    []detection.Detection
		lastDetectTime    time.Time
		detMtx            sync.Mutex
		isProcessing      bool
		notificationTimer = time.Now()
		notifyMtx         sync.Mutex
	)

	alertDir := fmt.Sprintf(record.RecordingsDir+"/%d", cameraId)
	dateDir := alertDir + "/" + time.Now().Format("02-01-2006")
	os.MkdirAll(dateDir, 0755)

	for {
		select {
		case <-ctx.Done():
			return
		case img, ok := <-frameCh:
			if !ok {
				log.Printf("[Camera %d] Frame channel closed...", cameraId)
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

					notifyMtx.Lock()
					if time.Now().After(notificationTimer) {
						notificationTimer = notificationTimer.Add(notification.NotificationCooldown)

						filename := dateDir + "/" + time.Now().Format("15-04-05") + ".jpg"

						mat := img.Clone()
						go func(img gocv.Mat, filename string) {
							defer img.Close()

							if ok := gocv.IMWrite(filename, img); ok {
								notification.SendMessage(notification.MailClient, filename, cameraId)
							}
						}(mat, filename)

					}
					notifyMtx.Unlock()
				}
				detMtx.Unlock()
			}

			buf, err := gocv.IMEncode(".jpg", img)
			if err == nil {
				stream.UpdateJPEG(buf.GetBytes())
				buf.Close()
			}
			img.Close()
		case <-time.After(timeoutCapture):
			log.Printf("[Camera %d] No new frames incoming. Automatically shutting down stream...", cameraId)

			go func(id int) {
				s.StopStream(id)
			}(cameraId)

			return
		}

	}
}

func (s *StreamManager) StopStream(cameraId int) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	registry, ok := s.activeStreams[cameraId]
	if !ok {
		return ErrStreamNotExist
	}

	registry.cancel()
	if err := capture.Manager.Unsubscribe(cameraId, registry.frameCh); err != nil {
		log.Println(err)
	}
	delete(s.activeStreams, cameraId)

	return nil
}

func (s *StreamManager) GetStream(cameraId int) (*mjpeg.Stream, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	registry, ok := s.activeStreams[cameraId]
	if !ok {
		return &mjpeg.Stream{}, ErrStreamNotExist
	}

	return registry.stream, nil
}

func (s *StreamManager) StopAll() {
	s.mtx.Lock()
	cameras := make([]int, 0, len(s.activeStreams))
	for id := range s.activeStreams {
		cameras = append(cameras, id)
	}
	s.mtx.Unlock()

	log.Println("Waiting for streams to stop...")

	for _, id := range cameras {
		s.StopStream(id)
	}

	log.Println("Stopped all streams...")
}
