package record

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
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
	// r.mtx.Lock()
	// defer r.mtx.Unlock()

	// if _, ok := r.activeCameras[cameraId]; ok {
	// 	return ErrAlreadyRecording
	// }

	// ctx, cancel := context.WithCancel(context.Background())
	// r.activeCameras[cameraId] = cancel

	// go func(ctx context.Context) {
	// 	dir := fmt.Sprintf("./recordings/%d", cameraId)
	// 	fileFormat := dir + "/%d-%m-%Y/%H-%M-%S.mp4"
	// 	os.MkdirAll(dir, os.ModePerm)

	// 	cmd := exec.CommandContext(
	// 		ctx, "ffmpeg", "-rtcp_transport", "tcp", "-i", rtspLink, "-f", "segment",
	// 		"-segment_time", "60", "-strftime", "1", "-reset_timestamps", "1", "-c", "copy",
	// 		fileFormat,
	// 	)

	// 	fmt.Println("Camera", cameraId, "started recording")

	// 	if err := cmd.Run(); err != nil {
	// 		fmt.Println("Camera", cameraId, "is stopped or an error occured:", err)
	// 	}
	// }(ctx)

	video, err := gocv.OpenVideoCapture(rtspLink)
	if err != nil {
		fmt.Println("Не удалось открыть RTSP для камеры %d: %v", cameraId, err)
		return err
	}
	defer video.Close()

	img := gocv.NewMat()
	defer img.Close()

	fps := video.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 25
	}
	width := int(video.Get(gocv.VideoCaptureFrameWidth))
	height := int(video.Get(gocv.VideoCaptureFrameHeight))

	baseDir := filepath.Join(".", "recordings", strconv.Itoa(cameraId))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			now := time.Now()
			dateDir := filepath.Join(baseDir, now.Format("2006-01-02"))
			os.MkdirAll(dateDir, 0755)

			fileName := filepath.Join(dateDir, now.Format("15-04-05.mp4"))

			// Открываем девайс записи видео (кодек mp4v или avc1 в зависимости от сборки OpenCV)
			writer, err := gocv.VideoWriterFile(fileName, "mp4v", fps, width, height, true)
			if err != nil {
				log.Printf("Ошибка создания файла записи %s: %v", fileName, err)
				time.Sleep(5 * time.Second)
				continue
			}

			segmentEnd := time.Now().Add(60 * time.Second) // Нарезка по 60 секунд ровно

			for time.Now().Before(segmentEnd) {
				select {
				case <-ctx.Done():
					writer.Close()
					return
				default:
					if ok := video.Read(&img); !ok || img.Empty() {
						log.Printf("Потерян поток с камеры %d, переподключение...", cameraId)
						time.Sleep(2 * time.Second)
						break
					}

					// Записываем кадр в файл архива
					writer.Write(img)
				}
			}
			writer.Close() // Закрываем сегмент, переходим к следующей минуте
		}
	}

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
