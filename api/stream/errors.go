package stream

import "errors"

var ErrStreamAlreadyExist error = errors.New("This stream already exist")
var ErrStreamNotExist error = errors.New("This stream doesn't exist")
var ErrTryingToOpenVideoCapture error = errors.New("Error occured while trying to open video capture. Check if camera is active")
