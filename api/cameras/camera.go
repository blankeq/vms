package cameras

import (
	"vms/api/database"
	"vms/api/dto"
)

func GetCameras() ([]dto.CameraDTO, error) {
	cameras := make([]dto.CameraDTO, 0)

	rows, err := database.DB.Query("SELECT id, name, rtsplink FROM cameras")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var c dto.CameraDTO

		if err := rows.Scan(&c.Id, &c.Name, &c.RTSPLink); err == nil {
			cameras = append(cameras, c)
		} else {
			return []dto.CameraDTO{}, err
		}
	}

	return cameras, nil
}

func CreateCamera(cameraDTO *dto.CameraDTO) (dto.CameraDTO, error) {
	err := database.DB.QueryRow("INSERT INTO cameras (name, rtsplink) VALUES ($1, $2) returning id", cameraDTO.Name, cameraDTO.RTSPLink).Scan(&cameraDTO.Id)
	if err != nil {
		return dto.CameraDTO{}, err
	}

	return *cameraDTO, nil
}

func DeleteCamera(cameraId int) error {
	if _, err := database.DB.Exec("DELETE FROM cameras WHERE id = $1", cameraId); err != nil {
		return err
	}

	return nil
}
