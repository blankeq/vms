package dto

import "errors"

var ErrNameIsBlank error = errors.New("Отсутствует имя камеры")
var ErrLinkIsBlank error = errors.New("Отстутствует RTSP-ссылка")
