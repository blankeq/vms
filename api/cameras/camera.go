package cameras

import (
	"vms/api/database"
)

type Camera struct {
	Id       *int   `json:"id"`
	Name     string `json:"name"`
	RTSPLink string `json:"rtsplink"`
}

func NewCamera(name string, rtsplink string) *Camera {
	return &Camera{
		Name:     name,
		RTSPLink: rtsplink,
	}
}

func GetCameras() ([]Camera, error) {
	cameras := make([]Camera, 0)

	rows, err := database.DB.Query("SELECT id, name, rtsp_url FROM cameras")
	if err != nil {
		return []Camera{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var c Camera

		if err := rows.Scan(&c.Id, &c.Name, &c.RTSPLink); err == nil {
			cameras = append(cameras, c)
		} else {
			return []Camera{}, err
		}
	}

	return cameras, nil
}

func GetCamera(cameraId int) (Camera, error) {
	var c Camera

	err := database.DB.QueryRow("SELECT id, name, rtsp_url FROM cameras WHERE id = $1", cameraId).Scan(&c.Id, &c.Name, &c.RTSPLink)
	if err != nil {
		return Camera{}, err
	}

	return c, nil
}

func CreateCamera(camera *Camera) (*Camera, error) {
	err := database.DB.QueryRow("INSERT INTO cameras (name, rtsp_url) VALUES ($1, $2) returning id", camera.Name, camera.RTSPLink).Scan(&camera.Id)
	if err != nil {
		return &Camera{}, err
	}

	return camera, nil
}

func DeleteCamera(cameraId int) error {
	if _, err := database.DB.Exec("DELETE FROM cameras WHERE id = $1", cameraId); err != nil {
		return err
	}

	return nil
}
