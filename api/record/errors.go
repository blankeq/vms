package record

import "errors"

var ErrAlreadyRecording error = errors.New("Эта камера уже записывает")
var ErrCameraNotActive error = errors.New("Эта камера ничего не записывает")
var ErrTryingToGetVideoInfo error = errors.New("Ошибка при попытке получить данные о потоке")
