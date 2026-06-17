package stream

import (
	"context"
	"image/color"
	"log"
	"sync"
	"time"
	"vms/api/capture"
	"vms/api/detection"

	"github.com/hybridgroup/mjpeg"
	"gocv.io/x/gocv"
)

const detectionInterval = 500 * time.Millisecond

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
	Stream  *mjpeg.Stream
	FrameCh capture.FrameSubscriber
	Cancel  context.CancelFunc
}

func NewStreamRegistry(stream *mjpeg.Stream, frameCh capture.FrameSubscriber, cancel context.CancelFunc) *StreamRegistry {
	return &StreamRegistry{
		Stream:  stream,
		FrameCh: frameCh,
		Cancel:  cancel,
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
	defer s.StopStream(cameraId)

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
		case img, ok := <-frameCh:
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

			buf, err := gocv.IMEncode(".jpg", img)
			if err == nil {
				stream.UpdateJPEG(buf.GetBytes())
				buf.Close()
			}
			img.Close()
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

	registry.Cancel()
	if err := capture.Manager.Unsubscribe(cameraId, registry.FrameCh); err != nil {
		log.Println(err)
	}
	delete(s.activeStreams, cameraId)

	log.Printf("[Camera %d] Stopped streaming...", cameraId)

	return nil
}

func (s *StreamManager) GetStream(cameraId int) (*mjpeg.Stream, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	registry, ok := s.activeStreams[cameraId]
	if !ok {
		return &mjpeg.Stream{}, ErrStreamNotExist
	}

	return registry.Stream, nil
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
