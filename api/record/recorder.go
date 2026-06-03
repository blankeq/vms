package record

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
	"vms/api/stream"

	"github.com/hybridgroup/mjpeg"
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
		return ErrTryingToOpenVideoCapture
	}
	defer video.Close()

	cameraStream := mjpeg.NewStream()
	stream.Manager.StartStream(cameraId, cameraStream)
	defer stream.Manager.StopStream(cameraId)

	img := gocv.NewMat()
	defer img.Close()

	ctx, cancel := context.WithCancel(context.Background())
	r.activeCameras[cameraId] = cancel
	// defer r.StopRecording(cameraId) - decide how to delete camera from active pool on error

	fps := video.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 25
	}
	width := int(video.Get(gocv.VideoCaptureFrameWidth))
	height := int(video.Get(gocv.VideoCaptureFrameHeight))

	dir := fmt.Sprintf("./recordings/%d", cameraId)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			now := time.Now()
			dirTime := now.Format("02-01-2006")
			dirPath := dir + "/" + dirTime
			os.MkdirAll(dirPath, os.ModePerm)
			fileTime := now.Format("15-04-05.mp4")
			filePath := dirPath + "/" + fileTime

			writer, err := gocv.VideoWriterFile(filePath, "mp4v", fps, width, height, true)
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
					return nil
				default:
					if ok := video.Read(&img); !ok || img.Empty() {
						time.Sleep(10 * time.Millisecond)
						break
					}

					writer.Write(img)

					buf, err := gocv.IMEncode(".jpg", img)
					if err == nil {
						cameraStream.UpdateJPEG(buf.GetBytes())
					}
				}
			}

			writer.Close()
		}
	}
}

func (r *RecordManager) StopRecording(cameraId int) error {
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
