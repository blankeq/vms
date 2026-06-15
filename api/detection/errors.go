package detection

import "errors"

var ErrFailedToLoadModel error = errors.New("Ошибка при загрузке модели обнаружения")
var ErrNoOutputLayers error = errors.New("Ошибка при загрузке модели обнаружения")
