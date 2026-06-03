package stream

import "errors"

var ErrStreamAlreadyExist error = errors.New("This stream already exist")
var ErrStreamNotExist error = errors.New("This stream doesn't exist")
