package capture

import "errors"

var ErrInstanceNotExist error = errors.New("Этот поток не был открыт")
var ErrTryingToOpenVideoCapture error = errors.New("Ошибка при попытке захвата видео. Проверьте, что камера активна или в ссылке нет опечаток")
