package catalog

import "context"

// Repository is the driven port for the catalog domain. Adapters (GORM, mocks) implement it.
type Repository interface {
	ListGenres(ctx context.Context) ([]Genre, error)
	ListNovels(ctx context.Context, filter NovelFilter) ([]Novel, error)
	GetNovelBySlug(ctx context.Context, slug string) (*NovelDetail, error)
	ListChapters(ctx context.Context, novelID int64, limit int) ([]Chapter, error)
	GetChapter(ctx context.Context, id int64) (*Chapter, error)
	GetChapterBody(ctx context.Context, chapterID int64) (string, error)
}
