package helper

import "errors"

var (
	// ErrInvalidCSVFormat when the number of columns is different than two
	ErrInvalidCSVFormat = errors.New("Invalid csv format, expecting: key, value")
)
