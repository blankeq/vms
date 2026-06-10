package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"vms/api/archive"
	"vms/api/cameras"
	"vms/api/dto"
	"vms/api/record"
	"vms/api/stream"

	"github.com/gorilla/mux"
)

type HTTPHandlers struct {
}

func NewHTTPHandlers() *HTTPHandlers {
	return &HTTPHandlers{}
}

func (h *HTTPHandlers) HandleCreateCamera(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST method", http.StatusMethodNotAllowed)
		return
	}

	var cameraDTO dto.CameraDTO

	if err := json.NewDecoder(r.Body).Decode(&cameraDTO); err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusBadRequest)
		return
	}

	if err := cameraDTO.ValidateCameraDTO(); err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusBadRequest)
		return
	}

	newCamera := cameras.NewCamera(cameraDTO.Name, cameraDTO.RTSPLink)
	newCamera, err := cameras.CreateCamera(newCamera)
	if err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
		return
	}

	b, err := json.MarshalIndent(&newCamera, "", "    ")
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(b); err != nil {
		fmt.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleDeleteCamera(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Use DELETE method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]

	cameraId, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()

		errDTO := dto.NewErrorDTO(errStr, time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusBadRequest)
		return
	}

	if err := cameras.DeleteCamera(cameraId); err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
		return
	}

	if err := record.Manager.StopRecording(cameraId); err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())

		w.Header().Set("Content-Type", "application/json")

		if !errors.Is(err, record.ErrCameraNotActive) {
			http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
			return
		} else {
			http.Error(w, errDTO.ToString(), http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandlers) HandleGetCameras(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cams, err := cameras.GetCameras()
	if err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())

		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
		return
	}

	b, err := json.MarshalIndent(&cams, "", "    ")
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(b); err != nil {
		fmt.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleStartRecording(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]

	cameraID, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()

		errDTO := dto.NewErrorDTO(errStr, time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusBadRequest)
		return
	}

	camera, err := cameras.GetCamera(cameraID)
	if err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
		return
	}

	if err := record.Manager.StartRecording(*camera.Id, camera.RTSPLink); err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	responseStr := []byte("Камера " + camera.Name + ": начала запись")
	if _, err := w.Write(responseStr); err != nil {
		fmt.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleStopRecording(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]

	cameraID, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()

		errDTO := dto.NewErrorDTO(errStr, time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusBadRequest)
		return
	}

	if err := record.Manager.StopRecording(cameraID); err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())

		w.Header().Set("Content-Type", "application/json")

		if errors.Is(err, record.ErrCameraNotActive) {
			http.Error(w, errDTO.ToString(), http.StatusBadRequest)
			return
		} else {
			http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	responseStr := []byte("Камера " + cameraIdQuery + ": запись остановлена")
	if _, err := w.Write(responseStr); err != nil {
		fmt.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleStartStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]

	cameraID, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()

		errDTO := dto.NewErrorDTO(errStr, time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusBadRequest)
		return
	}

	camera, err := cameras.GetCamera(cameraID)
	if err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
		return
	}

	if err := stream.Manager.StartStream(*camera.Id, camera.RTSPLink); err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	responseStr := []byte("Камера " + camera.Name + ": начала трансляцию")
	if _, err := w.Write(responseStr); err != nil {
		fmt.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleStopStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]

	cameraID, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()

		errDTO := dto.NewErrorDTO(errStr, time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusBadRequest)
		return
	}

	if err := stream.Manager.StopStream(cameraID); err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())

		w.Header().Set("Content-Type", "application/json")

		if errors.Is(err, stream.ErrStreamNotExist) {
			http.Error(w, errDTO.ToString(), http.StatusBadRequest)
			return
		} else {
			http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	responseStr := []byte("Камера " + cameraIdQuery + ": трансляция остановлена")
	if _, err := w.Write(responseStr); err != nil {
		fmt.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleGetStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]

	cameraID, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()

		errDTO := dto.NewErrorDTO(errStr, time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusBadRequest)
		return
	}

	stream, err := stream.Manager.GetStream(cameraID)
	if err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusBadRequest)
		return
	}

	stream.ServeHTTP(w, r)
}

func (h *HTTPHandlers) HandleGetArchiveFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]
	if _, err := strconv.Atoi(cameraIdQuery); err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()

		errDTO := dto.NewErrorDTO(errStr, time.Now())
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, errDTO.ToString(), http.StatusBadRequest)
		return
	}

	dateQuery := mux.Vars(r)["date"]

	files, err := archive.GetArchiveFiles(cameraIdQuery, dateQuery)
	if err != nil {
		errDTO := dto.NewErrorDTO(err.Error(), time.Now())

		w.Header().Set("Content-Type", "application/json")

		if errors.Is(err, archive.ErrFilesNotExist) {
			http.Error(w, errDTO.ToString(), http.StatusNotFound)
			return
		} else {
			http.Error(w, errDTO.ToString(), http.StatusInternalServerError)
			return
		}
	}

	b, err := json.MarshalIndent(&files, "", "    ")
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(b); err != nil {
		fmt.Println("Failed to write HTTP response:", err)
		return
	}
}
