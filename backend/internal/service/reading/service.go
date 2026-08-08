// Package reading is the application layer for the chapter reader.
package reading

import (
	"context"
	"time"

	domain "github.com/mokchan/webnovel-backend/internal/domain/reading"
	"github.com/mokchan/webnovel-backend/internal/domain/roles"
)

// Service orchestrates chapter reads, entitlement and progress.
type Service struct {
	repo         domain.Repository
	entitlements domain.Entitlements
	now          func() time.Time
}

// New wires the service. entitlements may be nil, in which case no paid
// chapter is ever considered owned.
func New(repo domain.Repository, entitlements domain.Entitlements, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, entitlements: entitlements, now: now}
}

// GetChapter returns a chapter with its body attached when the viewer is
// entitled to it, and with Locked set otherwise.
func (s *Service) GetChapter(ctx context.Context, id int64, v domain.Viewer) (*domain.ChapterView, error) {
	// Translators and admins may preview their own unpublished work; readers
	// must not learn that a draft exists.
	includeUnpublished := v.HasRole(roles.Translator) || v.HasRole(roles.Admin)

	chapter, err := s.repo.GetChapter(ctx, id, includeUnpublished)
	if err != nil {
		return nil, err
	}

	// A non-published chapter is visible only to its own translator or an admin.
	if chapter.Status != "published" &&
		!domain.IsTranslatorOf(*chapter, v) &&
		!v.HasRole(roles.Admin) {
		return nil, domain.ErrNotFound
	}

	unlocked := false
	if chapter.PriceCoins > 0 && !v.IsAnonymous() && s.entitlements != nil {
		unlocked, err = s.entitlements.IsChapterUnlocked(ctx, v.UserID, chapter.ID)
		if err != nil {
			return nil, err
		}
	}

	view := &domain.ChapterView{Chapter: *chapter}
	view.Locked = domain.Decide(chapter.PriceCoins, v, unlocked, domain.IsTranslatorOf(*chapter, v))

	prev, next, err := s.repo.NeighbourIDs(ctx, chapter.NovelID, chapter.ChapterNo)
	if err != nil {
		return nil, err
	}
	view.PrevID, view.NextID = prev, next

	if view.Locked {
		return view, nil
	}
	body, err := s.repo.GetBody(ctx, chapter.ID)
	if err != nil {
		return nil, err
	}
	view.BodyHTML = body
	return view, nil
}

// Neighbour returns the id of the chapter before or after the given one.
func (s *Service) Neighbour(ctx context.Context, id int64, direction string, v domain.Viewer) (*int64, error) {
	chapter, err := s.repo.GetChapter(ctx, id, false)
	if err != nil {
		return nil, err
	}
	prev, next, err := s.repo.NeighbourIDs(ctx, chapter.NovelID, chapter.ChapterNo)
	if err != nil {
		return nil, err
	}
	if direction == "prev" {
		return prev, nil
	}
	return next, nil
}

// GetProgress returns the reader's saved position in a novel, or nil when none
// has been recorded.
func (s *Service) GetProgress(ctx context.Context, userID, novelID int64) (*domain.Progress, error) {
	return s.repo.GetProgress(ctx, userID, novelID)
}

// SaveProgress records the reader's position, clamping the percentage.
func (s *Service) SaveProgress(ctx context.Context, p domain.Progress) (*domain.Progress, error) {
	if p.ParaAnchor < 0 {
		return nil, domain.ErrInvalidProgress
	}
	p.Pct = min(max(p.Pct, 0), 100)
	p.UpdatedAt = s.now()
	return s.repo.UpsertProgress(ctx, p)
}

// RecordRead stores a fire-and-forget read event used by the stats rollup.
func (s *Service) RecordRead(ctx context.Context, chapterID int64, userID int64, sessionID string) error {
	event := domain.ReadEvent{ChapterID: chapterID, OccurredAt: s.now()}
	if userID != 0 {
		event.UserID = &userID
	}
	if sessionID != "" {
		event.SessionID = &sessionID
	}
	return s.repo.InsertReadEvent(ctx, event)
}
