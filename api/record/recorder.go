package record

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

type RecordManager struct {
	mtx           sync.Mutex
	activeCameras map[int]context.CancelFunc
}

func NewRecordManager() *RecordManager {
	return &RecordManager{
		activeCameras: make(map[int]context.CancelFunc),
	}
}

var Manager = NewRecordManager()

func (r *RecordManager) StartRecording(cameraId int, rtspLink string) error {
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

	ctx, cancel := context.WithCancel(context.Background())

	r.activeCameras[cameraId] = cancel

	go r.recordingLoop(ctx, video, dir)

	fmt.Println("Camera", cameraId, "started recording")

	return nil
}

func (r *RecordManager) recordingLoop(ctx context.Context, video *gocv.VideoCapture, dir string) {
	defer video.Close()

	img := gocv.NewMat()
	defer img.Close()

	fps := video.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 25
	}
	width := int(video.Get(gocv.VideoCaptureFrameWidth))
	height := int(video.Get(gocv.VideoCaptureFrameHeight))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			now := time.Now()
			dirTime := now.Format("02-01-2006")
			dirPath := dir + "/" + dirTime
			os.MkdirAll(dirPath, 0755)
			fileTime := now.Format("15-04-05") + ".mp4"
			filePath := dirPath + "/" + fileTime

			writer, err := gocv.VideoWriterFile(filePath, "avc1", fps, width, height, true)
			if err != nil {
				fmt.Println("Failed to write image to file: ", err)
				time.Sleep(5 * time.Second)
				continue
			}

			segmentEnd := time.Now().Add(60 * time.Second)

			for time.Now().Before(segmentEnd) {
				select {
				case <-ctx.Done():
					writer.Close()
					return
				default:
					if ok := video.Read(&img); !ok {
						time.Sleep(10 * time.Millisecond)
						break
					}

					if img.Empty() {
						continue
					}

					writer.Write(img)
				}
			}

			writer.Close()
		}
	}
}

func (r *RecordManager) StopRecording(cameraId int) error { // need to add delay for writer to finish the file
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
