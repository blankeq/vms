package detection

import "errors"

var ErrFailedToLoadModel error = errors.New("Ошибка при загрузке модели обнаружения")
var ErrInputFrameIsEmpty error = errors.New("Входящий кадр пуст")
var ErrFailedToGetPointer error = errors.New("Ошибка при получении указателя")
