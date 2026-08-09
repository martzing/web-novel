// Package catalog is the application/use-case layer for catalog discovery.
package catalog

import (
	"context"
	"strconv"

	domain "github.com/mokchan/webnovel-backend/internal/domain/catalog"
)

// Service orchestrates catalog use cases over the domain repository port.
type Service struct {
	repo         domain.Repository
	entitlements domain.Entitlements
}

// New wires a service to a repository adapter. entitlements may be nil, in
// which case table-of-contents rows are never marked unlocked.
func New(repo domain.Repository, entitlements domain.Entitlements) *Service {
	return &Service{repo: repo, entitlements: entitlements}
}

// ListGenres returns every available discovery tag.
func (s *Service) ListGenres(ctx context.Context) ([]domain.Genre, error) {
	return s.repo.ListGenres(ctx)
}

// ListNovels applies default paging/sort and reports whether more rows follow.
func (s *Service) ListNovels(ctx context.Context, filter domain.NovelFilter) ([]domain.Novel, bool, error) {
	filter = normalizeFilter(filter)
	return s.repo.ListNovels(ctx, filter)
}

// GetNovel resolves a novel by numeric id or by slug. The route parameter is a
// single wildcard because gin panics when two names occupy the same path
// segment, so the discrimination happens here.
// viewerID is 0 for anonymous callers. A hidden novel is a 404 for everyone but
// its own translator — the filter lives here rather than in SQL because the
// same two repository reads serve the writer workspace, where hidden work must
// still be reachable.
func (s *Service) GetNovel(ctx context.Context, idOrSlug string, viewerID int64) (*domain.NovelDetail, error) {
	var (
		detail *domain.NovelDetail
		err    error
	)
	if id, parseErr := strconv.ParseInt(idOrSlug, 10, 64); parseErr == nil && id > 0 {
		detail, err = s.repo.GetNovelByID(ctx, id)
	} else {
		detail, err = s.repo.GetNovelBySlug(ctx, idOrSlug)
	}
	if err != nil {
		return nil, err
	}
	if !detail.VisibleTo(viewerID) {
		return nil, domain.ErrNotFound
	}
	return detail, nil
}

// GetSeries returns the public ชุดหนังสือ page by id or slug.
func (s *Service) GetSeries(ctx context.Context, idOrSlug string) (*domain.SeriesDetail, error) {
	return s.repo.GetSeries(ctx, idOrSlug)
}

// RelatedNovels returns เรื่องเกี่ยวเนื่อง grouped-ready for the detail page.
func (s *Service) RelatedNovels(ctx context.Context, novelID int64) ([]domain.RelatedNovel, error) {
	return s.repo.RelatedNovels(ctx, novelID)
}

// ListArcs returns the arc breakdown used by the reader's table of contents.
func (s *Service) ListArcs(ctx context.Context, novelID int64) ([]domain.Arc, error) {
	return s.repo.ListArcs(ctx, novelID)
}

// GetGlossary returns the novel's glossary grouped by category.
func (s *Service) GetGlossary(ctx context.Context, novelID int64) ([]domain.GlossaryGroup, error) {
	return s.repo.GetGlossary(ctx, novelID)
}

// ListChapters returns published chapters for the given novel, applying a
// default cap. When viewerID is non-zero the rows carry per-chapter unlock
// state, resolved in one bulk query rather than per row.
func (s *Service) ListChapters(ctx context.Context, novelID int64, limit int, viewerID int64) ([]domain.Chapter, error) {
	limit = clampChapterLimit(limit)
	chapters, err := s.repo.ListChapters(ctx, novelID, limit)
	if err != nil {
		return nil, err
	}
	if viewerID == 0 || s.entitlements == nil {
		return chapters, nil
	}

	paid := make([]int64, 0, len(chapters))
	for _, c := range chapters {
		if c.PriceCoins > 0 {
			paid = append(paid, c.ID)
		}
	}
	if len(paid) == 0 {
		return chapters, nil
	}
	unlocked, err := s.entitlements.ListUnlockedChapterIDs(ctx, viewerID, paid)
	if err != nil {
		return nil, err
	}
	for i := range chapters {
		chapters[i].Unlocked = chapters[i].PriceCoins == 0 || unlocked[chapters[i].ID]
	}
	return chapters, nil
}

// WeeklyRanking returns the weekly Top-N shown on the home page.
func (s *Service) WeeklyRanking(ctx context.Context, limit int) ([]domain.RankedNovel, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return s.repo.WeeklyRanking(ctx, limit)
}

func normalizeFilter(f domain.NovelFilter) domain.NovelFilter {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Sort != domain.SortLatest {
		f.Sort = domain.SortPopular
	}
	return f
}

func clampChapterLimit(limit int) int {
	if limit <= 0 || limit > 500 {
		return 100
	}
	return limit
}
