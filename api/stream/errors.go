package stream

import "errors"

var ErrStreamAlreadyExist error = errors.New("Эта трансляция уже активна")
var ErrStreamNotExist error = errors.New("Эта трансляция не активна")
var ErrTryingToOpenVideoCapture error = errors.New("Ошибка при попытке захвата видео. Проверьте, что камера активна или в ссылке нет опечаток")
