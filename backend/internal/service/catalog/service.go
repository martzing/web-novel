// Package catalog is the application/use-case layer for catalog discovery.
package catalog

import (
	"context"

	domain "github.com/mokchan/webnovel-backend/internal/domain/catalog"
)

// Service orchestrates catalog use cases over the domain repository port.
type Service struct {
	repo domain.Repository
}

// New wires a service to a repository adapter.
func New(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// ListGenres returns every available discovery tag.
func (s *Service) ListGenres(ctx context.Context) ([]domain.Genre, error) {
	return s.repo.ListGenres(ctx)
}

// ListNovels applies default paging/sort and forwards the request to the repository.
func (s *Service) ListNovels(ctx context.Context, filter domain.NovelFilter) ([]domain.Novel, error) {
	filter = normalizeFilter(filter)
	return s.repo.ListNovels(ctx, filter)
}

// GetNovelBySlug returns full novel detail or domain.ErrNotFound.
func (s *Service) GetNovelBySlug(ctx context.Context, slug string) (*domain.NovelDetail, error) {
	return s.repo.GetNovelBySlug(ctx, slug)
}

// ListChapters returns published chapters for the given novel, applying a default cap.
func (s *Service) ListChapters(ctx context.Context, novelID int64, limit int) ([]domain.Chapter, error) {
	limit = clampChapterLimit(limit)
	return s.repo.ListChapters(ctx, novelID, limit)
}

// GetChapter loads a chapter and gates the body: free chapters return the body,
// paid chapters are marked Locked. Access control comes in a later phase.
func (s *Service) GetChapter(ctx context.Context, id int64) (*domain.ChapterView, error) {
	chapter, err := s.repo.GetChapter(ctx, id)
	if err != nil {
		return nil, err
	}
	view := &domain.ChapterView{Chapter: *chapter}
	if chapter.PriceCoins > 0 {
		view.Locked = true
		return view, nil
	}
	body, err := s.repo.GetChapterBody(ctx, chapter.ID)
	if err != nil {
		return nil, err
	}
	view.BodyHTML = body
	return view, nil
}

func normalizeFilter(f domain.NovelFilter) domain.NovelFilter {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Sort != "latest" {
		f.Sort = "popular"
	}
	return f
}

func clampChapterLimit(limit int) int {
	if limit <= 0 || limit > 500 {
		return 100
	}
	return limit
}
