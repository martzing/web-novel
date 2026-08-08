// Package social is the domain layer for chapter comments, likes and per-novel
// reviews.
package social

import (
	"slices"
	"time"

	"github.com/mokchan/webnovel-backend/internal/domain/roles"
)

// Comment sort orders accepted by the list endpoint.
const (
	SortPopular     = "popular"
	SortLatest      = "latest"
	SortWithReplies = "with_replies"
)

// Author is the public identity attached to a comment or review.
type Author struct {
	ID          int64
	DisplayName string
	AvatarURL   string
	// Role is "translator" for the chapter's own translator, "admin" for a
	// platform administrator, else "reader". It drives the badge in the UI.
	Role string
}

// Comment is one chapter-scoped comment, optionally a reply.
type Comment struct {
	ID              int64
	ChapterID       int64
	UserID          int64
	ParentID        *int64
	Body            string
	IsSpoilerHidden bool
	LikesCount      int
	IsTranslator    bool
	CreatedAt       time.Time
	DeletedAt       *time.Time

	Author  Author
	Liked   bool
	Replies []Comment
}

// Review is a 1–5 star rating with optional prose.
type Review struct {
	ID        int64
	NovelID   int64
	UserID    int64
	Rating    int
	Body      string
	CreatedAt time.Time
	Author    Author
}

// Rating bounds, mirroring the CHECK constraint on reviews.rating.
const (
	MinRating = 1
	MaxRating = 5
)

// Viewer is the caller identity; the zero value is anonymous.
type Viewer struct {
	UserID int64
	Roles  []string
}

// IsAnonymous reports whether no user is attached.
func (v Viewer) IsAnonymous() bool { return v.UserID == 0 }

// HasRole reports whether the viewer carries the given role.
func (v Viewer) HasRole(role string) bool { return slices.Contains(v.Roles, role) }

// CanDelete reports whether the viewer may remove a comment: its author, the
// translator of the chapter it belongs to, or an administrator.
func CanDelete(c Comment, v Viewer, chapterTranslatorID *int64) bool {
	if v.IsAnonymous() {
		return false
	}
	if c.UserID == v.UserID {
		return true
	}
	if v.HasRole(roles.Admin) {
		return true
	}
	return chapterTranslatorID != nil && *chapterTranslatorID == v.UserID
}
