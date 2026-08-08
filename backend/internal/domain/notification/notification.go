// Package notification is the domain layer for reader notifications.
//
// It is deliberately thin: writer (chapter publish) and social (comment reply)
// depend on a one-method port here rather than on each other.
package notification

import (
	"context"
	"time"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
)

// Notification kinds.
const (
	KindNewChapter    = "new_chapter"
	KindReply         = "reply"
	KindBonusExpiring = "bonus_expiring"
)

// Notification is one inbox row.
type Notification struct {
	ID        int64
	UserID    int64
	Kind      string
	Payload   map[string]any
	ReadAt    *time.Time
	CreatedAt time.Time
}

// Repository is the driven port for the inbox.
type Repository interface {
	// FanOut writes one notification per recipient, skipping duplicates so a
	// republished chapter cannot spam followers. It returns how many rows were
	// actually created.
	FanOut(ctx context.Context, userIDs []int64, kind string, payload map[string]any, now time.Time) (int, error)
	List(ctx context.Context, userID int64, unreadOnly bool, p page.Page) ([]Notification, string, error)
	CountUnread(ctx context.Context, userID int64) (int, error)
	MarkRead(ctx context.Context, userID int64, ids []int64, now time.Time) error
	MarkAllRead(ctx context.Context, userID int64, now time.Time) error
}

// Followers lists who should hear about a novel's new chapter. It is satisfied
// by the library repository.
type Followers interface {
	ListFollowerIDs(ctx context.Context, novelID, afterID int64, limit int) ([]int64, error)
}
