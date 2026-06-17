package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"vms/api/archive"
	"vms/api/cameras"
	"vms/api/dto"
	"vms/api/record"
	"vms/api/stream"
	"vms/api/utils"

	"github.com/gorilla/mux"
)

type HTTPHandlers struct {
}

func NewHTTPHandlers() *HTTPHandlers {
	return &HTTPHandlers{}
}

func (h *HTTPHandlers) HandleCreateCamera(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondWithErrorString(w, "Use POST method", http.StatusMethodNotAllowed)
		return
	}

	var cameraDTO dto.CameraDTO

	if err := json.NewDecoder(r.Body).Decode(&cameraDTO); err != nil {
		utils.RespondWithErrorJson(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := cameraDTO.ValidateCameraDTO(); err != nil {
		utils.RespondWithErrorJson(w, err.Error(), http.StatusBadRequest)
		return
	}

	newCamera := cameras.NewCamera(cameraDTO.Name, cameraDTO.RTSPLink)
	newCamera, err := cameras.CreateCamera(newCamera)
	if err != nil {
		utils.RespondWithErrorJson(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := utils.RespondWithSuccessJson(w, newCamera, http.StatusCreated); err != nil {
		log.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleDeleteCamera(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.RespondWithErrorString(w, "Use DELETE method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]

	cameraId, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()
		utils.RespondWithErrorString(w, errStr, http.StatusBadRequest)
		return
	}

	if err := cameras.DeleteCamera(cameraId); err != nil {
		utils.RespondWithErrorString(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := record.Manager.StopRecording(cameraId); err != nil {
		if !errors.Is(err, record.ErrCameraNotActive) {
			utils.RespondWithErrorJson(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			utils.RespondWithErrorJson(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandlers) HandleGetCameras(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondWithErrorString(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cams, err := cameras.GetCameras()
	if err != nil {
		utils.RespondWithErrorJson(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := utils.RespondWithSuccessJson(w, cams, http.StatusOK); err != nil {
		log.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleStartRecording(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondWithErrorString(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]
	withDetectionQuery := r.URL.Query().Get("detection")

	cameraID, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()
		utils.RespondWithErrorString(w, errStr, http.StatusBadRequest)
		return
	}

	withDetection, err := strconv.ParseBool(withDetectionQuery)
	if err != nil {
		errStr := "Failed to convert detection param to bool: " + err.Error()
		utils.RespondWithErrorString(w, errStr, http.StatusBadRequest)
		return
	}

	camera, err := cameras.GetCamera(cameraID)
	if err != nil {
		utils.RespondWithErrorJson(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := record.Manager.StartRecording(*camera.Id, camera.RTSPLink, withDetection); err != nil {
		utils.RespondWithErrorJson(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseStr := "Камера " + cameraIdQuery + ": начала запись"
	if err := utils.RespondWithSuccessString(w, responseStr, http.StatusOK); err != nil {
		log.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleStopRecording(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondWithErrorString(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]

	cameraID, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()
		utils.RespondWithErrorString(w, errStr, http.StatusBadRequest)
		return
	}

	if err := record.Manager.StopRecording(cameraID); err != nil {
		if errors.Is(err, record.ErrCameraNotActive) {
			utils.RespondWithErrorJson(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			utils.RespondWithErrorJson(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	responseStr := "Камера " + cameraIdQuery + ": запись остановлена"
	if err := utils.RespondWithSuccessString(w, responseStr, http.StatusOK); err != nil {
		log.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleStartStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondWithErrorString(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]
	withDetectionQuery := r.URL.Query().Get("detection")

	cameraID, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()
		utils.RespondWithErrorString(w, errStr, http.StatusBadRequest)
		return
	}

	withDetection, err := strconv.ParseBool(withDetectionQuery)
	if err != nil {
		errStr := "Failed to convert detection param to bool: " + err.Error()
		utils.RespondWithErrorString(w, errStr, http.StatusBadRequest)
		return
	}

	camera, err := cameras.GetCamera(cameraID)
	if err != nil {
		utils.RespondWithErrorJson(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := stream.Manager.StartStream(*camera.Id, camera.RTSPLink, withDetection); err != nil {
		utils.RespondWithErrorJson(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseStr := "Камера " + cameraIdQuery + ": начала трансляцию"
	if err := utils.RespondWithSuccessString(w, responseStr, http.StatusOK); err != nil {
		log.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleStopStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondWithErrorString(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]

	cameraID, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()
		utils.RespondWithErrorString(w, errStr, http.StatusBadRequest)
		return
	}

	if err := stream.Manager.StopStream(cameraID); err != nil {
		if errors.Is(err, stream.ErrStreamNotExist) {
			utils.RespondWithErrorJson(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			utils.RespondWithErrorJson(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	responseStr := "Камера " + cameraIdQuery + ": трансляция остановлена"
	if err := utils.RespondWithSuccessString(w, responseStr, http.StatusOK); err != nil {
		log.Println("Failed to write HTTP response:", err)
		return
	}
}

func (h *HTTPHandlers) HandleGetStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondWithErrorString(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]

	cameraID, err := strconv.Atoi(cameraIdQuery)
	if err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()
		utils.RespondWithErrorString(w, errStr, http.StatusBadRequest)
		return
	}

	stream, err := stream.Manager.GetStream(cameraID)
	if err != nil {
		utils.RespondWithErrorJson(w, err.Error(), http.StatusBadRequest)
		return
	}

	stream.ServeHTTP(w, r)
}

func (h *HTTPHandlers) HandleGetArchiveFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.RespondWithErrorString(w, "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	cameraIdQuery := mux.Vars(r)["id"]
	if _, err := strconv.Atoi(cameraIdQuery); err != nil {
		errStr := "Failed to convert camera ID to int: " + err.Error()
		utils.RespondWithErrorString(w, errStr, http.StatusBadRequest)
		return
	}

	dateQuery := mux.Vars(r)["date"]

	files, err := archive.GetArchiveFiles(cameraIdQuery, dateQuery)
	if err != nil {
		if errors.Is(err, archive.ErrFilesNotExist) {
			utils.RespondWithErrorJson(w, err.Error(), http.StatusNotFound)
			return
		} else {
			utils.RespondWithErrorJson(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := utils.RespondWithSuccessJson(w, files, http.StatusOK); err != nil {
		log.Println("Failed to write HTTP response:", err)
		return
	}
}
