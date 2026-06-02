package dto

import "time"

type ErrorDTO struct {
	Message error     `json:"message"`
	Time    time.Time `json:"time"`
}

func NewErrorDTO(message error, time time.Time) ErrorDTO {
	return ErrorDTO{
		Message: message,
		Time:    time,
	}
}

type CameraDTO struct {
	Id       *int   `json:"id"`
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

func NewArchiveFileDTO(path string, time string) *ArchiveFileDTO {
	return &ArchiveFileDTO{
		Path: path,
		Time: time,
	}
}
