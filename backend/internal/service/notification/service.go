// Package notification is the application layer for the reader inbox.
package notification

import (
	"context"
	"time"

	domain "github.com/mokchan/webnovel-backend/internal/domain/notification"
	"github.com/mokchan/webnovel-backend/internal/domain/page"
)

// fanOutBatch bounds how many followers are read per round trip.
const fanOutBatch = 500

// Service orchestrates notification delivery and reading.
type Service struct {
	repo      domain.Repository
	followers domain.Followers
	now       func() time.Time
}

// New wires the service. followers may be nil, in which case chapter fan-out
// is a no-op.
func New(repo domain.Repository, followers domain.Followers, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, followers: followers, now: now}
}

// NotifyNewChapter tells a novel's followers that a chapter went live.
//
// Followers are paged through so a novel with tens of thousands of them does
// not load every id into memory at once.
func (s *Service) NotifyNewChapter(ctx context.Context, novelID, chapterID int64) error {
	if s.followers == nil {
		return nil
	}

	payload := map[string]any{
		"novel_id":   novelID,
		"chapter_id": chapterID,
	}

	var afterID int64
	for {
		ids, err := s.followers.ListFollowerIDs(ctx, novelID, afterID, fanOutBatch)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		if _, err := s.repo.FanOut(ctx, ids, domain.KindNewChapter, payload, s.now()); err != nil {
			return err
		}
		afterID = ids[len(ids)-1]
		if len(ids) < fanOutBatch {
			return nil
		}
	}
}

// NotifyReply tells a commenter that somebody replied.
func (s *Service) NotifyReply(ctx context.Context, recipientID, chapterID, commentID int64) error {
	_, err := s.repo.FanOut(ctx, []int64{recipientID}, domain.KindReply, map[string]any{
		"chapter_id": chapterID,
		"comment_id": commentID,
	}, s.now())
	return err
}

// List returns the caller's inbox.
func (s *Service) List(ctx context.Context, userID int64, unreadOnly bool, p page.Page) ([]domain.Notification, string, error) {
	return s.repo.List(ctx, userID, unreadOnly, p.Normalize(20, 100))
}

// CountUnread powers the bell badge.
func (s *Service) CountUnread(ctx context.Context, userID int64) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}

// MarkRead marks specific notifications read, or all of them when ids is empty.
func (s *Service) MarkRead(ctx context.Context, userID int64, ids []int64) error {
	if len(ids) == 0 {
		return s.repo.MarkAllRead(ctx, userID, s.now())
	}
	return s.repo.MarkRead(ctx, userID, ids, s.now())
}
