package stream

import "errors"

var ErrStreamAlreadyExist error = errors.New("Эта трансляция уже активна")
var ErrStreamNotExist error = errors.New("Эта трансляция не активна")
