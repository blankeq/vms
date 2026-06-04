package dto

import (
	"encoding/json"
	"time"
)

type ErrorDTO struct {
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

func NewErrorDTO(message string, time time.Time) ErrorDTO {
	return ErrorDTO{
		Message: message,
		Time:    time,
	}
}

func (e ErrorDTO) ToString() string {
	b, err := json.MarshalIndent(&e, "", "    ")
	if err != nil {
		panic(err)
	}

	return string(b)
}

type CameraDTO struct {
	Name     string `json:"name"`
	RTSPLink string `json:"rtsplink"`
}

func (c *CameraDTO) ValidateCameraDTO() error {
	if c.Name == "" {
		return ErrNameIsBlank
	}

	if c.RTSPLink == "" {
		return ErrLinkIsBlank
	}

	return nil
}

type ArchiveFileDTO struct {
	Path string `json:"path"`
	Time string `json:"time"`
}

func NewArchiveFileDTO(path string, time string) ArchiveFileDTO {
	return ArchiveFileDTO{
		Path: path,
		Time: time,
	}
}
