package library

import "errors"

var (
	ErrNotFound      = errors.New("library: not found")
	ErrForbidden     = errors.New("library: not the owner")
	ErrInvalidStatus = errors.New("library: invalid shelf status")
	ErrInvalidInput  = errors.New("library: invalid input")
)
