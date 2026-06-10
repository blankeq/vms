package record

import "errors"

var ErrAlreadyRecording error = errors.New("Эта камера уже записывает")
var ErrCameraNotActive error = errors.New("Эта камера ничего не записывает")
var ErrTryingToOpenVideoCapture error = errors.New("Ошибка при попытке захвата видео. Проверьте, что камера активна или в ссылке нет опечаток")
