package stream

import (
	"context"
	"image/color"
	"log"
	"os"
	"sync"
	"time"
	"vms/api/detection"

	"github.com/hybridgroup/mjpeg"
	"gocv.io/x/gocv"
)

const frameBufferSize = 1

type StreamManager struct {
	mtx           sync.Mutex
	activeStreams map[int]StreamRegistry
}

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

func NewStreamManager() *StreamManager {
	return &StreamManager{
		activeStreams: make(map[int]StreamRegistry, 0),
	}
}

var Manager = NewStreamManager()

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

	var detector *detection.Detector
	if withDetection {
		var err error
		modelPath := os.Getenv("YOLO_MODEL")
		detector, err = detection.NewDetector(modelPath, 0.45, 0.5)
		if err != nil {
			log.Printf("Camera %d error: Failed to load YOLO model: %v. Recording will continue WITHOUT detection.", cameraId, err)
			defer detector.Close()
		}
	} else {
		detector = nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	stream := mjpeg.NewStreamWithContext(ctx)

	registry := NewStreamRegistry(stream, cancel)
	s.activeStreams[cameraId] = registry

	if detector != nil {
		go s.streamLoop(ctx, video, stream, detector)
	} else {
		go s.streamLoop(ctx, video, stream, nil)
	}

	log.Println("Camera", cameraId, "started streaming")

	return nil
}

func (s *StreamManager) streamLoop(ctx context.Context, video *gocv.VideoCapture, stream *mjpeg.Stream, detector *detection.Detector) {
	defer video.Close()

	img := gocv.NewMat()
	defer img.Close()

	if detector != nil {
		defer detector.Close()
	}

	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	frameChan := make(chan gocv.Mat, frameBufferSize)

	recWg := sync.WaitGroup{}
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
						if !isProcessing {
							isProcessing = true
							detectImg := img.Clone()

							go func(mat gocv.Mat) {
								defer mat.Close()
								detections, _ := detector.Detect(&mat)
								detMtx.Lock()
								if detections != nil {
									lastDetections = detections
								}
								isProcessing = false
								detMtx.Unlock()
							}(detectImg)
						}
						detMtx.Unlock()

						detMtx.Lock()
						detection.DrawOverlay(&img, lastDetections, green)
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
