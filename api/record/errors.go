package record

import "errors"

var ErrAlreadyRecording error = errors.New("This camera is already recording")
var ErrCameraNotActive error = errors.New("This camera is not active")
