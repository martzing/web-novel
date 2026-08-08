package reading

import "errors"

var (
	// ErrNotFound covers a missing chapter and an unpublished one alike, so a
	// draft never leaks its existence to a reader.
	ErrNotFound = errors.New("reading: not found")
	// ErrInvalidProgress is returned when a progress update is out of range.
	ErrInvalidProgress = errors.New("reading: invalid progress")
)
