package archive

import (
	"os"
	"path/filepath"
	"strings"
	"vms/api/dto"
)

func GetArchiveFiles(cameraId string, date string) ([]dto.ArchiveFileDTO, error) {
	dirPath := "./recordings/" + cameraId + "/" + date
	files, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []dto.ArchiveFileDTO{}, nil
		}
	}

	archiveFiles := make([]dto.ArchiveFileDTO, 0)

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".mp4" {
			var af dto.ArchiveFileDTO

			filename := file.Name()
			path := "/" + cameraId + "/" + "date" + filename
			time := strings.TrimSuffix(filename, ".mp4")

			af = dto.ArchiveFileDTO{
				Path: path,
				Time: time,
			}

			archiveFiles = append(archiveFiles, af)
		}
	}

	return archiveFiles, nil
}
