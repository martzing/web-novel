// Package library is the application layer for the reader's shelf.
package library

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	domain "github.com/mokchan/webnovel-backend/internal/domain/library"
	"github.com/mokchan/webnovel-backend/internal/domain/page"
)

// Service orchestrates shelf, bookmark and follow use cases.
type Service struct {
	repo   domain.Repository
	series domain.SeriesBooks
	now    func() time.Time
}

// New wires the service. series resolves ชุดหนังสือ to its books for the
// follow-a-whole-series fan-out; it is satisfied by the catalog repository.
func New(repo domain.Repository, series domain.SeriesBooks, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, series: series, now: now}
}

// ListShelf returns one tab of the reader's shelf.
func (s *Service) ListShelf(ctx context.Context, userID int64, tab string, p page.Page) ([]domain.EntryWithNovel, string, error) {
	if tab != "" && !domain.ValidStatus(tab) {
		return nil, "", domain.ErrInvalidStatus
	}
	return s.repo.ListShelf(ctx, userID, tab, p.Normalize(20, 100))
}

// CountShelf returns the tab badge counts.
func (s *Service) CountShelf(ctx context.Context, userID int64) (domain.Counts, error) {
	return s.repo.CountShelf(ctx, userID)
}

// SetShelfStatus adds a novel to the shelf or moves it between tabs.
func (s *Service) SetShelfStatus(ctx context.Context, userID, novelID int64, status string) (*domain.Entry, error) {
	if !domain.ValidStatus(status) {
		return nil, domain.ErrInvalidStatus
	}
	return s.repo.UpsertShelf(ctx, domain.Entry{
		UserID:  userID,
		NovelID: novelID,
		Status:  status,
		AddedAt: s.now(),
	})
}

// RemoveFromShelf takes a novel off the shelf.
func (s *Service) RemoveFromShelf(ctx context.Context, userID, novelID int64) error {
	return s.repo.RemoveShelf(ctx, userID, novelID)
}

// ListBookmarks returns the caller's bookmarks, optionally scoped to a novel.
func (s *Service) ListBookmarks(ctx context.Context, userID int64, novelID *int64, p page.Page) ([]domain.Bookmark, string, error) {
	return s.repo.ListBookmarks(ctx, userID, novelID, p.Normalize(50, 200))
}

// CreateBookmark saves a paragraph position with a short excerpt.
func (s *Service) CreateBookmark(ctx context.Context, b domain.Bookmark) (*domain.Bookmark, error) {
	if b.ChapterID <= 0 || b.NovelID <= 0 || b.ParaAnchor < 0 {
		return nil, domain.ErrInvalidInput
	}

	b.Excerpt = strings.TrimSpace(b.Excerpt)
	if b.Excerpt == "" {
		return nil, domain.ErrInvalidInput
	}
	// Truncate by runes: Thai is three bytes per character, so a byte-based cut
	// would slice a character in half and corrupt the excerpt.
	b.Excerpt = truncateRunes(b.Excerpt, domain.MaxExcerptRunes)
	b.Note = truncateRunes(strings.TrimSpace(b.Note), domain.MaxExcerptRunes)
	b.CreatedAt = s.now()

	return s.repo.CreateBookmark(ctx, b)
}

// DeleteBookmark removes a bookmark the caller owns.
func (s *Service) DeleteBookmark(ctx context.Context, id, userID int64) error {
	return s.repo.DeleteBookmark(ctx, id, userID)
}

// Follow subscribes the reader to a novel's new chapters.
func (s *Service) Follow(ctx context.Context, userID, novelID int64) error {
	return s.repo.Follow(ctx, userID, novelID)
}

// Unfollow removes the subscription.
func (s *Service) Unfollow(ctx context.Context, userID, novelID int64) error {
	return s.repo.Unfollow(ctx, userID, novelID)
}

// IsFollowing reports whether the reader follows the novel.
func (s *Service) IsFollowing(ctx context.Context, userID, novelID int64) (bool, error) {
	return s.repo.IsFollowing(ctx, userID, novelID)
}

// FollowSeries subscribes the reader to every book in a series.
//
// ติดตามทั้งชุด is a fan-out over the existing per-novel follows rather than a
// row of its own. A series row would need its own notification fan-out and
// would have to decide what happens when a book joins later; following the
// books directly reuses the machinery that already works, and a book added
// afterwards simply leaves the series "partial" until the reader acts.
func (s *Service) FollowSeries(ctx context.Context, userID int64, seriesIDOrSlug string) (domain.SeriesFollow, error) {
	ids, err := s.seriesBooks(ctx, seriesIDOrSlug)
	if err != nil {
		return domain.SeriesFollow{}, err
	}
	if _, err := s.repo.FollowMany(ctx, userID, ids); err != nil {
		return domain.SeriesFollow{}, err
	}
	return domain.SummariseSeriesFollow(len(ids), len(ids)), nil
}

// UnfollowSeries drops the reader's follows on every book in the series, and
// only those: a novel that has since left the series keeps its own follow.
func (s *Service) UnfollowSeries(ctx context.Context, userID int64, seriesIDOrSlug string) (domain.SeriesFollow, error) {
	ids, err := s.seriesBooks(ctx, seriesIDOrSlug)
	if err != nil {
		return domain.SeriesFollow{}, err
	}
	if _, err := s.repo.UnfollowMany(ctx, userID, ids); err != nil {
		return domain.SeriesFollow{}, err
	}
	return domain.SummariseSeriesFollow(0, len(ids)), nil
}

// SeriesFollowState summarises how much of a series the reader follows, so the
// button can read ติดตามทั้งชุด, ติดตามอยู่ or ติดตามบางเล่ม.
func (s *Service) SeriesFollowState(ctx context.Context, userID int64, seriesIDOrSlug string) (domain.SeriesFollow, error) {
	ids, err := s.seriesBooks(ctx, seriesIDOrSlug)
	if err != nil {
		return domain.SeriesFollow{}, err
	}
	n, err := s.repo.CountFollowing(ctx, userID, ids)
	if err != nil {
		return domain.SeriesFollow{}, err
	}
	return domain.SummariseSeriesFollow(n, len(ids)), nil
}

func (s *Service) seriesBooks(ctx context.Context, seriesIDOrSlug string) ([]int64, error) {
	if s.series == nil {
		return nil, domain.ErrNotFound
	}
	ids, err := s.series.NovelIDsInSeries(ctx, seriesIDOrSlug)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		// An empty series is not an error — there is simply nothing to follow.
		return []int64{}, nil
	}
	return ids, nil
}

// ListFollowerIDs pages through a novel's followers, for notification fan-out.
func (s *Service) ListFollowerIDs(ctx context.Context, novelID, afterID int64, limit int) ([]int64, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	return s.repo.ListFollowerIDs(ctx, novelID, afterID, limit)
}

func truncateRunes(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	runes := []rune(s)
	return string(runes[:limit])
}
