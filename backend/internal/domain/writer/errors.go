package writer

import "errors"

var (
	ErrNotFound        = errors.New("writer: not found")
	ErrForbidden       = errors.New("writer: not the owner")
	ErrInvalidInput    = errors.New("writer: invalid input")
	ErrInvalidPrice    = errors.New("writer: price must be zero or more")
	ErrChapterNoTaken  = errors.New("writer: chapter number already used")
	ErrSlugTaken       = errors.New("writer: slug already used")
	ErrUnsupportedFile = errors.New("writer: unsupported file type")
	ErrFileTooLarge    = errors.New("writer: file is too large")
	ErrGroupNotEmpty   = errors.New("writer: glossary group still holds terms")
)
