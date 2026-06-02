package stream

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"vms/api/database"
)

const Boundary = "vms_mjpeg_boundary"

func GetLiveStreamHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("camera_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Неверный ID")
		return
	}

	var rtspURL string
	err = database.DB.QueryRow("SELECT rtsp_url FROM cameras WHERE id = $1", id).Scan(&rtspURL)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Камера не найдена")
		return
	}

	// Инициализируем видеопоток для трансляции
	video, err := gocv.OpenVideoCapture(rtspURL)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Ошибка подключения к камере")
		return
	}
	defer video.Close()

	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary="+Boundary)
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	img := gocv.NewMat()
	defer img.Close()

	for {
		select {
		case <-r.Context().Done(): // Браузер закрыл вкладку — останавливаем стрим
			return
		default:
			if ok := video.Read(&img); !ok || img.Empty() {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Кодируем матрицу кадра (Mat) в JPEG в памяти Linux-сервера
			buf, err := gocv.IMEncode(".jpg", img)
			if err != nil {
				return
			}

			// Формируем MJPEG multipart-пакет
			_, err = fmt.Fprintf(w, "--%s\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", Boundary, len(buf))
			if err != nil {
				return
			}

			// Отправляем байты кадра
			if _, err = w.Write(buf); err != nil {
				return
			}
			if _, err = w.Write([]byte("\r\n")); err != nil {
				return
			}

			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}
