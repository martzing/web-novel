package social

import "errors"

var (
	ErrNotFound       = errors.New("social: not found")
	ErrForbidden      = errors.New("social: not permitted")
	ErrCommentEmpty   = errors.New("social: comment is empty")
	ErrCommentTooLong = errors.New("social: comment is too long")
	ErrReplyTooDeep   = errors.New("social: replies may only be one level deep")
	ErrInvalidRating  = errors.New("social: rating must be between 1 and 5")
)
