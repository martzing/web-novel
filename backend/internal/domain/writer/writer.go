// Package writer is the domain layer for the translator workspace: novels,
// arcs, chapter drafts, publishing, glossary management and stats.
package writer

import "time"

// Chapter statuses, mirroring the CHECK constraint on chapters.status.
const (
	StatusDraft     = "draft"
	StatusScheduled = "scheduled"
	StatusPublished = "published"
)

// NovelDraft is the writable view of a novel.
type NovelDraft struct {
	ID          int64
	Slug        string
	TitleTH     string
	TitleCN     string
	AuthorName  string
	Description string
	CoverURL    string
	Status      string
	SeriesID    *int64
	GenreIDs    []int64
}

// Arc is a chapter range within a novel.
type Arc struct {
	ID            int64
	NovelID       int64
	ArcNo         int
	Name          string
	FromChapterNo int
	ToChapterNo   int
}

// Chapter is the writable view of a chapter.
type Chapter struct {
	ID          int64
	NovelID     int64
	ArcID       *int64
	ChapterNo   int
	Title       string
	BodySource  string
	BodyHTML    string
	PriceCoins  int
	WordCount   int
	Status      string
	ScheduledAt *time.Time
	PublishedAt *time.Time
	UpdatedAt   time.Time
}

// Revision is one autosave snapshot.
type Revision struct {
	ID         int64
	ChapterID  int64
	AuthorID   int64
	BodySource string
	SavedAt    time.Time
}

// KeepRevisions is how many autosave snapshots survive per chapter.
const KeepRevisions = 20

// GlossaryGroup is a glossary category within a novel.
type GlossaryGroup struct {
	ID      int64
	NovelID int64
	Name    string
	SortNo  int
	Entries []GlossaryEntry
}

// GlossaryEntry is one term.
type GlossaryEntry struct {
	ID      int64
	GroupID int64
	TermKey string
	TitleTH string
	TitleCN string
	Body    string
	Kind    string
}

// DailyPoint is one day of aggregated statistics.
type DailyPoint struct {
	Day         time.Time
	Reads       int
	CoinsEarned int
	Followers   int
}

// ChapterPerformance is one row of the best-performing-chapters table.
type ChapterPerformance struct {
	ChapterID   int64
	ChapterNo   int
	Title       string
	Reads       int
	CoinsEarned int
}

// NovelStats powers the writer's KPI tiles.
type NovelStats struct {
	Reads          int
	Followers      int
	CoinsEarned    int
	ReadsTrendPct  float64
	CoinsTrendPct  float64
	Series         []DailyPoint
	TopChapters    []ChapterPerformance
	PeriodFrom     time.Time
	PeriodTo       time.Time
	AvgCompletePct float64
}
