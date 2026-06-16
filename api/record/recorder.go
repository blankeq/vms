package record

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"os"
	"sync"
	"time"
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
	activeCameras map[int]context.CancelFunc
}

func NewRecordManager() *RecordManager {
	return &RecordManager{
		activeCameras: make(map[int]context.CancelFunc),
	}
}

var Manager = NewRecordManager()

func (r *RecordManager) StartRecording(cameraId int, rtspLink string, withDetection bool) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if _, ok := r.activeCameras[cameraId]; ok {
		return ErrAlreadyRecording
	}

	video, err := gocv.OpenVideoCapture(rtspLink)
	if err != nil {
		video.Close()
		return ErrTryingToOpenVideoCapture
	}

	dir := fmt.Sprintf("./recordings/%d", cameraId)
	if err := os.MkdirAll(dir, 0755); err != nil {
		video.Close()
		return err
	}

	var detector *detection.SharedDetector
	if withDetection {
		detector = detection.MainDetector
	} else {
		detector = nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	r.activeCameras[cameraId] = cancel
	r.wg.Add(1)

	go r.recordingLoop(ctx, video, detector, dir)

	log.Println("Camera", cameraId, "started recording")

	return nil
}

func (r *RecordManager) recordingLoop(ctx context.Context, video *gocv.VideoCapture, detector *detection.SharedDetector, dir string) {
	defer r.wg.Done()
	defer video.Close()

	img := gocv.NewMat()
	defer img.Close()

	fps := video.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 25
	}
	width := int(video.Get(gocv.VideoCaptureFrameWidth))
	height := int(video.Get(gocv.VideoCaptureFrameHeight))

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
				now := time.Now()
				dirTime := now.Format("02-01-2006")
				dirPath := dir + "/" + dirTime
				_ = os.MkdirAll(dirPath, 0755)
				fileTime := now.Format("15-04-05") + ".mp4"
				filePath := dirPath + "/" + fileTime

				writer, err := gocv.VideoWriterFile(filePath, "avc1", fps, width, height, true)
				if err != nil {
					log.Println("Failed to write image to file: ", err)
					time.Sleep(5 * time.Second)
					continue
				}

				segmentEnd := time.Now().Add(60 * time.Second)

				for time.Now().Before(segmentEnd) {
					select {
					case <-ctx.Done():
						writer.Close()
						return
					case img, ok := <-frameChan:
						if !ok {
							writer.Close()
							return
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

						writer.Write(img)
						img.Close()
					}
				}
				writer.Close()
			}
		}
	}()

	recWg.Wait()
	detectWg.Wait()
}

// func (r *RecordManager) recordingLoop(ctx context.Context, video *gocv.VideoCapture, detector *detection.Detector, dir string) {
// 	defer r.wg.Done()
// 	defer video.Close()

// 	img := gocv.NewMat()
// 	defer img.Close()

// 	if detector != nil {
// 		defer detector.Close()
// 	}

// 	fps := video.Get(gocv.VideoCaptureFPS)
// 	if fps <= 0 {
// 		fps = 25
// 	}
// 	width := int(video.Get(gocv.VideoCaptureFrameWidth))
// 	height := int(video.Get(gocv.VideoCaptureFrameHeight))

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
// 			now := time.Now()
// 			dirTime := now.Format("02-01-2006")
// 			dirPath := dir + "/" + dirTime
// 			os.MkdirAll(dirPath, 0755)
// 			fileTime := now.Format("15-04-05") + ".mp4"
// 			filePath := dirPath + "/" + fileTime

// 			writer, err := gocv.VideoWriterFile(filePath, "avc1", fps, width, height, true)
// 			if err != nil {
// 				log.Println("Failed to write image to file: ", err)
// 				time.Sleep(5 * time.Second)
// 				continue
// 			}

// 			segmentEnd := time.Now().Add(60 * time.Second)

// 			for time.Now().Before(segmentEnd) {
// 				select {
// 				case <-ctx.Done():
// 					writer.Close()
// 					return
// 				default:
// 					if ok := video.Read(&img); !ok {
// 						time.Sleep(10 * time.Millisecond)
// 						break
// 					}

// 					if img.Empty() {
// 						continue
// 					}

// 					if detector != nil {
// 						// detections, err := detector.Detect(&img)
// 						// if err == nil {
// 						// 	detection.DrawOverlay(&img, detections,
// 						// 		struct{ R, G, B, A uint8 }{0, 255, 0, 255})
// 						// } else {
// 						// 	log.Printf("Camera: Detection error: %v", err)
// 						// }

// 						detMtx.Lock()

// 						if !isProcessing {
// 							isProcessing = true

// 							detectImg := img.Clone()

// 							go func(mat gocv.Mat) {
// 								defer mat.Close()

// 								detections, err := detector.Detect(&mat)

// 								detMtx.Lock()
// 								if err == nil {
// 									lastDetections = detections
// 								}
// 								isProcessing = false
// 								detMtx.Unlock()
// 							}(detectImg)
// 						}

// 						detMtx.Unlock()

// 						detMtx.Lock()
// 						detection.DrawOverlay(&img, lastDetections, green)
// 						detMtx.Unlock()
// 					}

// 					writer.Write(img)
// 				}
// 			}

// 			writer.Close()
// 		}
// 	}
// }

func (r *RecordManager) StopRecording(cameraId int) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	cancel, ok := r.activeCameras[cameraId]
	if !ok {
		return ErrCameraNotActive
	}

	cancel()
	delete(r.activeCameras, cameraId)

	log.Println("Camera", cameraId, "stopped recording")

	return nil
}

func (r *RecordManager) StopAll() {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	for cameraId, cancel := range r.activeCameras {
		cancel()
		delete(r.activeCameras, cameraId)
	}

	log.Println("Waiting for recordings to save on disk...")
	r.wg.Wait()
	log.Println("Stopped all recordings...")
}
