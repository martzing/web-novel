package social

import (
	"strings"
	"unicode/utf8"
)

// MaxCommentRunes matches the database CHECK, which counts characters:
//
//	CHECK (char_length(body) BETWEEN 1 AND 5000)
//
// Validation must therefore count runes too. A byte-length check would reject a
// perfectly legal 1,700-character Thai comment, since Thai is three bytes per
// character in UTF-8.
const MaxCommentRunes = 5000

// MaxReplyDepth allows one level of replies; a reply to a reply is rejected.
const MaxReplyDepth = 1

// ValidateComment checks a comment body and its nesting depth.
func ValidateComment(body string, parentDepth int) error {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ErrCommentEmpty
	}
	if utf8.RuneCountInString(trimmed) > MaxCommentRunes {
		return ErrCommentTooLong
	}
	if parentDepth >= MaxReplyDepth {
		return ErrReplyTooDeep
	}
	return nil
}

// ValidateRating checks a review score.
func ValidateRating(rating int) error {
	if rating < MinRating || rating > MaxRating {
		return ErrInvalidRating
	}
	return nil
}

// NormalizeSort falls back to the default ordering for an unknown value.
func NormalizeSort(sort string) string {
	switch sort {
	case SortLatest, SortWithReplies:
		return sort
	default:
		return SortPopular
	}
}
