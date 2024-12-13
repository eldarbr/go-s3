package myerrors

import "errors"

var (
	ErrServiceNullPtr = errors.New("nullptr exception")
)
