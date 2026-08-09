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
	repo          domain.Repository
	entitlements  domain.Entitlements
	subscriptions domain.Subscriptions
	now           func() time.Time
}

// New wires the service. entitlements and subscriptions may be nil, in which
// case no paid chapter is ever owned and nobody has early access.
func New(
	repo domain.Repository,
	entitlements domain.Entitlements,
	subscriptions domain.Subscriptions,
	now func() time.Time,
) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, entitlements: entitlements, subscriptions: subscriptions, now: now}
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

	isTranslator := domain.IsTranslatorOf(*chapter, v)

	unlocked := false
	if chapter.PriceCoins > 0 && !v.IsAnonymous() && s.entitlements != nil {
		unlocked, err = s.entitlements.IsChapterUnlocked(ctx, v.UserID, chapter.ID)
		if err != nil {
			return nil, err
		}
	}

	// Only ask about the subscription when it could change the answer: a
	// chapter already past its early-access window does not need the lookup.
	subscribed := false
	if chapter.PublicAtTime != nil && chapter.PublicAtTime.After(s.now()) &&
		!v.IsAnonymous() && s.subscriptions != nil {
		subscribed, err = s.subscriptions.IsSubscribed(ctx, v.UserID, chapter.NovelID)
		if err != nil {
			return nil, err
		}
	}

	// Timing first, then entitlement. See decides whether the chapter exists
	// for this viewer at all; Decide then decides whether they have paid.
	visibility := domain.See(
		domain.Availability{
			Status:      chapter.Status,
			PublishedAt: chapter.PublishedAtTime,
			PublicAt:    chapter.PublicAtTime,
		},
		domain.Access{
			Viewer:       v,
			Subscribed:   subscribed,
			Owns:         unlocked,
			IsTranslator: isTranslator,
		},
		s.now(),
	)
	if visibility == domain.VisibleHidden {
		return nil, domain.ErrNotFound
	}

	view := &domain.ChapterView{Chapter: *chapter}
	switch {
	case visibility == domain.VisibleTeaser:
		view.Locked = true
		view.LockedReason = domain.LockedReasonEarlyAccess
	default:
		view.Locked = domain.Decide(chapter.PriceCoins, v, unlocked, isTranslator)
		if view.Locked {
			view.LockedReason = domain.LockedReasonPaywall
		}
	}

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
