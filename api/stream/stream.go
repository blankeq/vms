package stream

import (
	"context"
	"image/color"
	"log"
	"sync"
	"time"
	"vms/api/detection"

	"github.com/hybridgroup/mjpeg"
	"gocv.io/x/gocv"
)

const (
	frameBufferSize   = 1
	detectionInterval = 300 * time.Millisecond
)

type StreamManager struct {
	mtx           sync.Mutex
	activeStreams map[int]StreamRegistry
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		activeStreams: make(map[int]StreamRegistry, 0),
	}
}

var Manager = NewStreamManager()

type StreamRegistry struct {
	Stream *mjpeg.Stream
	Cancel context.CancelFunc
}

func NewStreamRegistry(stream *mjpeg.Stream, cancel context.CancelFunc) StreamRegistry {
	return StreamRegistry{
		Stream: stream,
		Cancel: cancel,
	}
}

func (s *StreamManager) StartStream(cameraId int, rtspLink string, withDetection bool) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	_, ok := s.activeStreams[cameraId]
	if ok {
		return ErrStreamAlreadyExist
	}

	video, err := gocv.OpenVideoCapture(rtspLink)
	if err != nil {
		video.Close()
		return ErrTryingToOpenVideoCapture
	}

	var detector *detection.SharedDetector
	if withDetection {
		detector = detection.MainDetector
	} else {
		detector = nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	stream := mjpeg.NewStreamWithContext(ctx)

	registry := NewStreamRegistry(stream, cancel)
	s.activeStreams[cameraId] = registry

	go s.streamLoop(ctx, video, stream, detector)

	log.Println("Camera", cameraId, "started streaming")

	return nil
}

func (s *StreamManager) streamLoop(ctx context.Context, video *gocv.VideoCapture, stream *mjpeg.Stream, detector *detection.SharedDetector) {
	defer video.Close()

	img := gocv.NewMat()
	defer img.Close()

	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	frameChan := make(chan gocv.Mat, frameBufferSize)

	recWg := sync.WaitGroup{}
	detectWg := sync.WaitGroup{}
	recWg.Add(2)

	go func() {
		defer recWg.Done()
		defer close(frameChan)

		img := gocv.NewMat()

		for {
			select {
			case <-ctx.Done():
				img.Close()
				return
			default:
				if ok := video.Read(&img); !ok {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				if img.Empty() {
					continue
				}

				newFrame := img.Clone()

				select {
				case frameChan <- newFrame:
				default:
					newFrame.Close()
					log.Println("Buffer overflow, dropping frame...")
				}
			}
		}
	}()

	go func() {
		defer recWg.Done()

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
			default:

				select {
				case <-ctx.Done():
					return
				case img, ok := <-frameChan:
					if !ok {
						time.Sleep(10 * time.Millisecond)
						continue
					}

					if img.Empty() {
						continue
					}

					if detector != nil {
						detMtx.Lock()

						if !isProcessing && time.Since(lastDetectTime) >= detectionInterval {
							isProcessing = true
							lastDetectTime = time.Now()
							detectImg := img.Clone()

							detectWg.Add(1)

							go func(img gocv.Mat) {
								defer detectWg.Done()
								defer img.Close()

								detector.Mtx.Lock()
								detections, _ := detector.Detector.Detect(&img)
								detector.Mtx.Unlock()

								detMtx.Lock()
								// if detections != nil {
								// 	lastDetections = detections
								// }
								lastDetections = detections
								isProcessing = false
								detMtx.Unlock()
							}(detectImg)
						}
						detMtx.Unlock()

						detMtx.Lock()
						// detection.DrawOverlay(&img, lastDetections, green)
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
	}()

	recWg.Wait()
	detectWg.Wait()
}

// func (s *StreamManager) streamLoop(ctx context.Context, video *gocv.VideoCapture, stream *mjpeg.Stream, detector *detection.Detector) {
// 	defer video.Close()

// 	img := gocv.NewMat()
// 	defer img.Close()

// 	if detector != nil {
// 		defer detector.Close()
// 	}

// 	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}

// 	var (
// 		lastDetections []detection.Detection
// 		detMtx         sync.Mutex
// 		isProcessing   bool
// 	)

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		default:
// 			if ok := video.Read(&img); !ok {
// 				time.Sleep(10 * time.Millisecond)
// 				continue
// 			}

// 			if img.Empty() {
// 				continue
// 			}

// 			if detector != nil {
// 				// detections, err := detector.Detect(&img)
// 				// if err == nil {
// 				// 	detection.DrawOverlay(&img, detections,
// 				// 		struct{ R, G, B, A uint8 }{0, 255, 0, 255})
// 				// } else {
// 				// 	log.Printf("Camera: Detection error: %v", err)
// 				// }

// 				detMtx.Lock()

// 				if !isProcessing {
// 					isProcessing = true

// 					detectImg := img.Clone()

// 					go func(mat gocv.Mat) {
// 						defer mat.Close()

// 						detections, err := detector.Detect(&mat)

// 						detMtx.Lock()
// 						if err == nil {
// 							lastDetections = detections
// 						}
// 						isProcessing = false
// 						detMtx.Unlock()
// 					}(detectImg)
// 				}

// 				detMtx.Unlock()

// 				detMtx.Lock()
// 				detection.DrawOverlay(&img, lastDetections, green)
// 				detMtx.Unlock()
// 			}

// 			buf, err := gocv.IMEncode(".jpg", img)
// 			if err == nil {
// 				stream.UpdateJPEG(buf.GetBytes())
// 				buf.Close()
// 			}
// 		}
// 	}
// }

func (s *StreamManager) StopStream(cameraId int) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	registry, ok := s.activeStreams[cameraId]
	if !ok {
		return ErrStreamNotExist
	}

	registry.Cancel()
	delete(s.activeStreams, cameraId)

	log.Println("Camera", cameraId, "stopped streaming")

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
	defer s.mtx.Unlock()

	for cameraId, registry := range s.activeStreams {
		registry.Cancel()
		delete(s.activeStreams, cameraId)
	}

	log.Println("Stopped all streams...")
}
