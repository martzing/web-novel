// Package writer is the application layer for the translator workspace.
package writer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
	domain "github.com/mokchan/webnovel-backend/internal/domain/writer"
)

// MaxCoverBytes caps an uploaded cover image.
const MaxCoverBytes = 2 << 20 // 2 MiB

// Publisher is notified when a chapter goes live, so followers can be told.
// A one-method port keeps writer independent of the notification context.
type Publisher interface {
	NotifyNewChapter(ctx context.Context, novelID, chapterID int64) error
}

// Service orchestrates the writer use cases.
type Service struct {
	repo      domain.Repository
	storage   domain.Storage
	publisher Publisher
	now       func() time.Time
}

// New wires the service. storage and publisher may be nil.
func New(repo domain.Repository, storage domain.Storage, publisher Publisher, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, storage: storage, publisher: publisher, now: now}
}

// ListNovels returns the novels this translator owns.
func (s *Service) ListNovels(ctx context.Context, ownerID int64, p page.Page) ([]domain.NovelDraft, string, error) {
	return s.repo.ListNovels(ctx, ownerID, p.Normalize(20, 100))
}

// CreateNovel registers a new work, generating a URL-safe slug.
func (s *Service) CreateNovel(ctx context.Context, n domain.NovelDraft, ownerID int64) (*domain.NovelDraft, error) {
	n.TitleTH = strings.TrimSpace(n.TitleTH)
	if n.TitleTH == "" {
		return nil, domain.ErrInvalidInput
	}
	if n.Status == "" {
		n.Status = "ongoing"
	}
	if err := domain.ValidateSettings(n); err != nil {
		return nil, err
	}

	if strings.TrimSpace(n.Slug) == "" {
		seq, err := s.repo.NextSlugSeq(ctx)
		if err != nil {
			return nil, err
		}
		n.Slug = domain.SlugFromTitle(n.TitleTH, seq)
	} else {
		// A caller-supplied slug still has to survive the id-or-slug route.
		n.Slug = domain.SlugFromTitle(n.Slug, 0)
	}
	return s.repo.CreateNovel(ctx, n, ownerID)
}

// UpdateNovel patches a novel the caller owns.
func (s *Service) UpdateNovel(ctx context.Context, userID, novelID int64, n domain.NovelDraft) (*domain.NovelDraft, error) {
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return nil, err
	}
	if err := domain.ValidateSettings(n); err != nil {
		return nil, err
	}
	// Joining a series is only legal into one the caller owns; without this a
	// translator could file their work under someone else's collection.
	if n.SeriesIDSet && n.SeriesID != nil {
		if err := s.assertOwnsSeries(ctx, userID, *n.SeriesID); err != nil {
			return nil, err
		}
	}
	return s.repo.UpdateNovel(ctx, novelID, n)
}

// UploadCover stores a cover image and points the novel at it.
//
// The content type is sniffed from the bytes rather than trusted from the
// client, so a renamed executable cannot be served back as an image.
func (s *Service) UploadCover(ctx context.Context, userID, novelID int64, data []byte) (string, error) {
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", domain.ErrInvalidInput
	}
	if len(data) > MaxCoverBytes {
		return "", domain.ErrFileTooLarge
	}
	if s.storage == nil {
		return "", domain.ErrInvalidInput
	}

	contentType := http.DetectContentType(data)
	ext, ok := imageExtension(contentType)
	if !ok {
		return "", domain.ErrUnsupportedFile
	}

	name, err := randomName(ext)
	if err != nil {
		return "", err
	}
	url, err := s.storage.Save(ctx, name, contentType, data)
	if err != nil {
		return "", err
	}
	if err := s.repo.SetCoverURL(ctx, novelID, url); err != nil {
		return "", err
	}
	return url, nil
}

// ListArcs returns a novel's arcs.
func (s *Service) ListArcs(ctx context.Context, userID, novelID int64) ([]domain.Arc, error) {
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return nil, err
	}
	return s.repo.ListArcs(ctx, novelID)
}

// CreateArc adds an arc covering a chapter range.
func (s *Service) CreateArc(ctx context.Context, userID int64, a domain.Arc) (*domain.Arc, error) {
	if err := s.assertOwnsNovel(ctx, userID, a.NovelID); err != nil {
		return nil, err
	}
	if a.Name == "" || a.FromChapterNo <= 0 || a.ToChapterNo < a.FromChapterNo {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.CreateArc(ctx, a)
}

// UpdateArc patches an arc the caller owns.
func (s *Service) UpdateArc(ctx context.Context, userID, arcID int64, a domain.Arc) (*domain.Arc, error) {
	novelID, err := s.repo.ArcNovelID(ctx, arcID)
	if err != nil {
		return nil, err
	}
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return nil, err
	}
	return s.repo.UpdateArc(ctx, arcID, a)
}

// ListChapters returns the chapter rail, drafts included.
func (s *Service) ListChapters(ctx context.Context, userID, novelID int64, p page.Page) ([]domain.Chapter, string, error) {
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return nil, "", err
	}
	return s.repo.ListChapters(ctx, novelID, p.Normalize(100, 500))
}

// GetChapter loads a chapter for editing.
func (s *Service) GetChapter(ctx context.Context, userID, chapterID int64) (*domain.Chapter, error) {
	if err := s.assertOwnsChapter(ctx, userID, chapterID); err != nil {
		return nil, err
	}
	return s.repo.GetChapter(ctx, chapterID)
}

// CreateChapter adds a new draft, resolving its arc from the chapter number.
func (s *Service) CreateChapter(ctx context.Context, userID int64, c domain.Chapter) (*domain.Chapter, error) {
	if err := s.assertOwnsNovel(ctx, userID, c.NovelID); err != nil {
		return nil, err
	}
	if c.PriceCoins < 0 {
		return nil, domain.ErrInvalidPrice
	}
	if c.ChapterNo <= 0 {
		return nil, domain.ErrInvalidInput
	}

	// The novel's pricing settings decide the default, so a translator sets the
	// price once on the work rather than on every chapter. An explicit price on
	// the request still wins — except below free_until_chapter, where the promise
	// to readers is that those chapters are free, and a stray value in the form
	// must not quietly break it.
	novel, err := s.repo.GetNovel(ctx, c.NovelID)
	if err != nil {
		return nil, err
	}
	if c.PriceCoins == 0 && novel.PricePerChapter != nil {
		c.PriceCoins = *novel.PricePerChapter
	}
	if novel.FreeUntilChapter != nil && c.ChapterNo <= *novel.FreeUntilChapter {
		c.PriceCoins = 0
	}

	arcs, err := s.repo.ListArcs(ctx, c.NovelID)
	if err != nil {
		return nil, err
	}
	c.ArcID = domain.ResolveArcID(arcs, c.ChapterNo)
	c.Status = domain.StatusDraft
	c.WordCount = wordCount(c.BodySource)

	return s.repo.CreateChapter(ctx, c)
}

// SaveChapter autosaves an edit and trims the revision history.
func (s *Service) SaveChapter(ctx context.Context, userID, chapterID int64, c domain.Chapter) (*domain.Chapter, error) {
	if err := s.assertOwnsChapter(ctx, userID, chapterID); err != nil {
		return nil, err
	}
	if c.PriceCoins < 0 {
		return nil, domain.ErrInvalidPrice
	}

	existing, err := s.repo.GetChapter(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	c.ID = chapterID
	c.NovelID = existing.NovelID
	c.Status = existing.Status
	c.WordCount = wordCount(c.BodySource)

	if c.ChapterNo != existing.ChapterNo && c.ChapterNo > 0 {
		arcs, err := s.repo.ListArcs(ctx, existing.NovelID)
		if err != nil {
			return nil, err
		}
		c.ArcID = domain.ResolveArcID(arcs, c.ChapterNo)
	} else {
		c.ChapterNo = existing.ChapterNo
		c.ArcID = existing.ArcID
	}

	return s.repo.SaveChapter(ctx, c, userID, domain.KeepRevisions)
}

// PublishChapter publishes now, or schedules for later.
func (s *Service) PublishChapter(ctx context.Context, userID, chapterID int64, scheduledAt *time.Time) (*domain.Chapter, error) {
	if err := s.assertOwnsChapter(ctx, userID, chapterID); err != nil {
		return nil, err
	}

	status, publishedAt := domain.PublishDecision(scheduledAt, s.now())

	chapter, err := s.repo.PublishChapter(ctx, chapterID, status, publishedAt, scheduledAt)
	if err != nil {
		return nil, err
	}

	// Followers hear about it only once it is actually readable.
	if status == domain.StatusPublished && s.publisher != nil {
		_ = s.publisher.NotifyNewChapter(ctx, chapter.NovelID, chapter.ID)
	}
	return chapter, nil
}

// UnpublishChapter returns a chapter to draft.
func (s *Service) UnpublishChapter(ctx context.Context, userID, chapterID int64) (*domain.Chapter, error) {
	if err := s.assertOwnsChapter(ctx, userID, chapterID); err != nil {
		return nil, err
	}
	return s.repo.UnpublishChapter(ctx, chapterID)
}

// ListGlossary returns the novel's glossary for the editor.
func (s *Service) ListGlossary(ctx context.Context, userID, novelID int64) ([]domain.GlossaryGroup, error) {
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return nil, err
	}
	return s.repo.ListGlossary(ctx, novelID)
}

// CreateGlossaryGroup adds a glossary category.
func (s *Service) CreateGlossaryGroup(ctx context.Context, userID int64, g domain.GlossaryGroup) (*domain.GlossaryGroup, error) {
	if err := s.assertOwnsNovel(ctx, userID, g.NovelID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(g.Name) == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.CreateGlossaryGroup(ctx, g)
}

// CreateGlossaryEntry adds a term to a group.
func (s *Service) CreateGlossaryEntry(ctx context.Context, userID, novelID int64, e domain.GlossaryEntry) (*domain.GlossaryEntry, error) {
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(e.TermKey) == "" || strings.TrimSpace(e.TitleTH) == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.CreateGlossaryEntry(ctx, e)
}

// UpdateGlossaryEntry edits a term. The database trigger bumps the novel's
// glossary_rev, which the re-render worker picks up.
func (s *Service) UpdateGlossaryEntry(ctx context.Context, userID, entryID int64, e domain.GlossaryEntry) (*domain.GlossaryEntry, error) {
	novelID, err := s.repo.GlossaryEntryNovelID(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return nil, err
	}
	return s.repo.UpdateGlossaryEntry(ctx, entryID, e)
}

// Stats returns the KPI tiles for a period, comparing against the window
// immediately before it.
func (s *Service) Stats(ctx context.Context, userID, novelID int64, period string) (*domain.NovelStats, error) {
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return nil, err
	}

	days := periodDays(period)
	to := s.now()
	from := to.AddDate(0, 0, -days)

	current, err := s.repo.DailyStats(ctx, novelID, from, to)
	if err != nil {
		return nil, err
	}
	previous, err := s.repo.DailyStats(ctx, novelID, from.AddDate(0, 0, -days), from)
	if err != nil {
		return nil, err
	}

	stats := domain.Aggregate(current, previous, from, to)

	top, err := s.repo.TopChapters(ctx, novelID, from, to, 5)
	if err != nil {
		return nil, err
	}
	stats.TopChapters = top
	return &stats, nil
}

func (s *Service) assertOwnsNovel(ctx context.Context, userID, novelID int64) error {
	owns, err := s.repo.OwnsNovel(ctx, userID, novelID)
	if err != nil {
		return err
	}
	if !owns {
		return domain.ErrForbidden
	}
	return nil
}

func (s *Service) assertOwnsChapter(ctx context.Context, userID, chapterID int64) error {
	owns, err := s.repo.OwnsChapter(ctx, userID, chapterID)
	if err != nil {
		return err
	}
	if !owns {
		return domain.ErrForbidden
	}
	return nil
}

// periodDays maps the documented period values to a window length.
func periodDays(period string) int {
	switch period {
	case "30d":
		return 30
	case "all":
		return 3650
	default:
		return 14
	}
}

// wordCount is a cheap whitespace count. Thai does not separate words with
// spaces, so this under-counts Thai prose; it exists to populate the editor's
// indicator, not to be linguistically exact.
func wordCount(source string) int {
	return len(strings.Fields(source))
}

func imageExtension(contentType string) (string, bool) {
	switch {
	case strings.HasPrefix(contentType, "image/jpeg"):
		return ".jpg", true
	case strings.HasPrefix(contentType, "image/png"):
		return ".png", true
	case strings.HasPrefix(contentType, "image/webp"):
		return ".webp", true
	case strings.HasPrefix(contentType, "image/gif"):
		return ".gif", true
	default:
		return "", false
	}
}

// randomName keeps the stored filename server-generated, so a client cannot
// choose a path or an extension.
func randomName(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("writer: read random: %w", err)
	}
	return hex.EncodeToString(buf) + ext, nil
}
