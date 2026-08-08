package writer

import (
	"context"
	"time"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
)

// Repository is the driven port for the translator workspace.
type Repository interface {
	// OwnsNovel and OwnsChapter gate every mutation (I-WR-03).
	OwnsNovel(ctx context.Context, userID, novelID int64) (bool, error)
	OwnsChapter(ctx context.Context, userID, chapterID int64) (bool, error)

	CreateNovel(ctx context.Context, n NovelDraft, ownerID int64) (*NovelDraft, error)
	UpdateNovel(ctx context.Context, id int64, n NovelDraft) (*NovelDraft, error)
	GetNovel(ctx context.Context, id int64) (*NovelDraft, error)
	ListNovels(ctx context.Context, ownerID int64, p page.Page) ([]NovelDraft, string, error)
	SetCoverURL(ctx context.Context, novelID int64, url string) error

	ListArcs(ctx context.Context, novelID int64) ([]Arc, error)
	CreateArc(ctx context.Context, a Arc) (*Arc, error)
	UpdateArc(ctx context.Context, id int64, a Arc) (*Arc, error)
	ArcNovelID(ctx context.Context, arcID int64) (int64, error)

	ListChapters(ctx context.Context, novelID int64, p page.Page) ([]Chapter, string, error)
	GetChapter(ctx context.Context, id int64) (*Chapter, error)
	CreateChapter(ctx context.Context, c Chapter) (*Chapter, error)
	// SaveChapter persists an edit, appends an autosave revision and prunes
	// older ones, all in one transaction.
	SaveChapter(ctx context.Context, c Chapter, authorID int64, keepRevisions int) (*Chapter, error)
	// PublishChapter renders the body against the novel's glossary, records the
	// bound entries, and stamps the status in one transaction.
	PublishChapter(ctx context.Context, chapterID int64, status string, publishedAt, scheduledAt *time.Time) (*Chapter, error)
	UnpublishChapter(ctx context.Context, chapterID int64) (*Chapter, error)
	CountRevisions(ctx context.Context, chapterID int64) (int, error)

	ListGlossary(ctx context.Context, novelID int64) ([]GlossaryGroup, error)
	CreateGlossaryGroup(ctx context.Context, g GlossaryGroup) (*GlossaryGroup, error)
	CreateGlossaryEntry(ctx context.Context, e GlossaryEntry) (*GlossaryEntry, error)
	UpdateGlossaryEntry(ctx context.Context, id int64, e GlossaryEntry) (*GlossaryEntry, error)
	GlossaryEntryNovelID(ctx context.Context, entryID int64) (int64, error)

	DailyStats(ctx context.Context, novelID int64, from, to time.Time) ([]DailyPoint, error)
	TopChapters(ctx context.Context, novelID int64, from, to time.Time, limit int) ([]ChapterPerformance, error)

	// NextSlugSeq yields a monotonically increasing number used to keep
	// generated slugs unique.
	NextSlugSeq(ctx context.Context) (int64, error)
}

// Storage persists uploaded cover images behind a port, so object storage can
// replace the local directory without touching the service.
type Storage interface {
	Save(ctx context.Context, name string, contentType string, data []byte) (url string, err error)
}
