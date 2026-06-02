package dto

import "errors"

var ErrNameIsBlank error = errors.New("No camera name found")
var ErrLinkIsBlank error = errors.New("No RTSP Link found")
