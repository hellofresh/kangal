package helper

import "errors"

var (
	ErrInvalidCSVFormat = errors.New("Invalid csv format, expecting: key, value")
)
