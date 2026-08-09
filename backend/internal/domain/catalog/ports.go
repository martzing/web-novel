package catalog

import "context"

// Repository is the driven port for the catalog domain. Adapters (GORM, fakes)
// implement it.
type Repository interface {
	ListGenres(ctx context.Context) ([]Genre, error)

	// ListNovels returns at most filter.Limit novels plus a flag reporting
	// whether more rows exist beyond the page.
	ListNovels(ctx context.Context, filter NovelFilter) ([]Novel, bool, error)

	GetNovelBySlug(ctx context.Context, slug string) (*NovelDetail, error)
	GetNovelByID(ctx context.Context, id int64) (*NovelDetail, error)

	ListArcs(ctx context.Context, novelID int64) ([]Arc, error)
	ListChapters(ctx context.Context, novelID int64, limit int) ([]Chapter, error)
	GetGlossary(ctx context.Context, novelID int64) ([]GlossaryGroup, error)

	// WeeklyRanking reads the most recent ranking snapshot, falling back to
	// live popularity when no snapshot has been taken yet.
	WeeklyRanking(ctx context.Context, limit int) ([]RankedNovel, error)

	// GetSeries resolves ชุดหนังสือ by numeric id or slug, with its books in
	// reading order. Hidden novels are already excluded.
	GetSeries(ctx context.Context, idOrSlug string) (*SeriesDetail, error)
	// RelatedNovels returns เรื่องเกี่ยวเนื่อง for a novel, in both stored
	// directions, with the mirrored side carrying its inverse kind.
	RelatedNovels(ctx context.Context, novelID int64) ([]RelatedNovel, error)
}

// Entitlements reports which chapters a reader already owns. It is satisfied by
// the wallet repository; catalog never imports the wallet package.
type Entitlements interface {
	ListUnlockedChapterIDs(ctx context.Context, userID int64, chapterIDs []int64) (map[int64]bool, error)
}
