package service

import "errors"

var (
	ErrDeviceNotFound  = errors.New("device not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrInvalidTimestamp = errors.New("invalid timestamp format")
)
