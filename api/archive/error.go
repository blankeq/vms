package archive

import "errors"

var ErrFilesNotExist error = errors.New("Файлы за эту дату отсутствуют")
