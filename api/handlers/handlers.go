package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
	"vms/api/cameras"
	"vms/api/dto"
)

type HTTPHandlers struct {
}

func (h *HTTPHandlers) HandleCreateCamera(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST method", http.StatusMethodNotAllowed)
		return
	}

	var cameraDTO dto.CameraDTO

	if err := json.NewDecoder(r.Body).Decode(&cameraDTO); err != nil {
		panic(err)
	}

	if err := cameraDTO.ValidateCameraDTO(); err != nil {
		if errors.Is(err, dto.ErrLinkIsBlank) || errors.Is(err, dto.ErrNameIsBlank) {
			errorDTO := dto.NewErrorDTO(err.Error(), time.Now())

			b, err := json.MarshalIndent(&errorDTO, "", "    ")
			if err != nil {
				panic(err)
			}

			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write(b); err != nil {
				fmt.Println("Failed to write HTTP response: ", err)
				return
			}

			return
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
}

func (h *HTTPHandlers) HandleDeleteCamera(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Use DELETE method", http.StatusMethodNotAllowed)
		return
	}

	var cameraDTO dto.CameraDTO

	if err := json.NewDecoder(r.Body).Decode(&cameraDTO); err != nil {
		panic(err)
	}

	if cameraDTO.Id != nil {
		if err := cameras.DeleteCamera(*cameraDTO.Id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		errorDTO := dto.NewErrorDTO("Camera ID is nil", time.Now())

		b, err := json.MarshalIndent(&errorDTO, "", "    ")
		if err != nil {
			panic(err)
		}

		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write(b); err != nil {
			fmt.Println("Failed to write HTTP response: ", err)
			return
		}
	}

}
