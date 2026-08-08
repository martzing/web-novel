// Package social is the application layer for comments, likes and reviews.
package social

import (
	"context"
	"strings"
	"time"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
	domain "github.com/mokchan/webnovel-backend/internal/domain/social"
)

// Notifier is fired when a comment gets a reply. It is a one-method port so
// social never depends on the notification context directly.
type Notifier interface {
	NotifyReply(ctx context.Context, recipientID, chapterID, commentID int64) error
}

// Service orchestrates the social use cases.
type Service struct {
	repo     domain.Repository
	notifier Notifier
	now      func() time.Time
}

// New wires the service. notifier may be nil.
func New(repo domain.Repository, notifier Notifier, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, notifier: notifier, now: now}
}

// ListComments returns a chapter's comment thread.
func (s *Service) ListComments(ctx context.Context, chapterID int64, sort string, viewerID int64, p page.Page) ([]domain.Comment, string, error) {
	return s.repo.ListComments(ctx, chapterID, domain.NormalizeSort(sort), viewerID, p.Normalize(20, 100))
}

// CreateComment posts a comment or a one-level reply.
func (s *Service) CreateComment(ctx context.Context, c domain.Comment) (*domain.Comment, error) {
	// Depth of the comment being replied to: 0 for a new top-level comment,
	// 1 when the parent is itself a reply — which ValidateComment rejects.
	parentDepth := 0
	if c.ParentID != nil {
		parent, err := s.repo.GetComment(ctx, *c.ParentID)
		if err != nil {
			return nil, err
		}
		if parent.ChapterID != c.ChapterID {
			return nil, domain.ErrNotFound
		}
		if parent.ParentID != nil {
			parentDepth = domain.MaxReplyDepth
		}
	}
	if err := domain.ValidateComment(c.Body, parentDepth); err != nil {
		return nil, err
	}

	c.Body = strings.TrimSpace(c.Body)
	c.CreatedAt = s.now()

	created, err := s.repo.CreateComment(ctx, c)
	if err != nil {
		return nil, err
	}

	// Notify the parent's author, unless they are replying to themselves.
	if created.ParentID != nil && s.notifier != nil {
		if parent, err := s.repo.GetComment(ctx, *created.ParentID); err == nil &&
			parent.UserID != created.UserID {
			_ = s.notifier.NotifyReply(ctx, parent.UserID, created.ChapterID, created.ID)
		}
	}
	return created, nil
}

// DeleteComment soft-deletes a comment the viewer is allowed to remove.
func (s *Service) DeleteComment(ctx context.Context, id int64, v domain.Viewer) error {
	comment, err := s.repo.GetComment(ctx, id)
	if err != nil {
		return err
	}
	translatorID, err := s.repo.ChapterTranslatorID(ctx, comment.ChapterID)
	if err != nil {
		return err
	}
	if !domain.CanDelete(*comment, v, translatorID) {
		return domain.ErrForbidden
	}
	return s.repo.SoftDeleteComment(ctx, id)
}

// Like registers a like; liking twice leaves the count at one.
func (s *Service) Like(ctx context.Context, userID, commentID int64) (int, error) {
	return s.repo.Like(ctx, userID, commentID)
}

// Unlike removes the caller's like.
func (s *Service) Unlike(ctx context.Context, userID, commentID int64) (int, error) {
	return s.repo.Unlike(ctx, userID, commentID)
}

// UpsertReview writes the caller's one review for a novel.
func (s *Service) UpsertReview(ctx context.Context, r domain.Review) (*domain.Review, error) {
	if err := domain.ValidateRating(r.Rating); err != nil {
		return nil, err
	}
	r.Body = strings.TrimSpace(r.Body)
	r.CreatedAt = s.now()
	return s.repo.UpsertReview(ctx, r)
}

// ListReviews returns a novel's reviews.
func (s *Service) ListReviews(ctx context.Context, novelID int64, p page.Page) ([]domain.Review, string, error) {
	return s.repo.ListReviews(ctx, novelID, p.Normalize(20, 100))
}

// GetUserReview returns the caller's own review, or nil when they have none.
func (s *Service) GetUserReview(ctx context.Context, novelID, userID int64) (*domain.Review, error) {
	return s.repo.GetUserReview(ctx, novelID, userID)
}
