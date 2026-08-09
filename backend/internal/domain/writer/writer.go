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
//
// The string fields keep the existing "" means "leave alone" convention. The
// settings below cannot: false and 0 are the values a translator most wants to
// set — turning arc sales off, making every chapter free — so they are pointers
// and nil means "not supplied". Mixing the two conventions in one struct is
// unfortunate, but the alternative is a patch that can enable a toggle and
// never disable it.
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

	// SeriesIDSet distinguishes "leave the series alone" from "remove it",
	// since both are a nil SeriesID.
	SeriesIDSet bool

	// SourceChaptersCount is บทในต้นฉบับ: how long the original work is. It is
	// translator-entered because no feed tells us, and it is what every
	// "translated N of M" figure in the product divides by.
	SourceChaptersCount *int

	PricePerChapter  *int
	FreeUntilChapter *int
	SellByArc        *bool
	TipsEnabled      *bool
	EarlyAccessHours *int

	// ReleaseSchedule (รอบปล่อยบทใหม่) is display-only metadata this round: it
	// tells readers when to come back, and deliberately does not drive the
	// publishing scheduler.
	ReleaseSchedule string

	CoverStyle string
	CoverColor string
	CoverText  string

	SeriesPosition *int
	SeriesNote     string

	// Read-only, filled on read.
	ChaptersCount int
	Title         string
}

// Release schedules offered by the settings tab. The column is free text so
// the option list can grow without a migration; the API rejects anything else
// so the reader-facing label set stays closed and translatable.
const (
	ReleaseIrregular = "irregular"
	ReleaseDaily     = "daily"
	ReleaseWeekly    = "weekly"
	ReleaseBiweekly  = "biweekly"
	ReleaseMonthly   = "monthly"
)

// ValidReleaseSchedule reports whether s is one of the offered cadences.
func ValidReleaseSchedule(s string) bool {
	switch s {
	case ReleaseIrregular, ReleaseDaily, ReleaseWeekly, ReleaseBiweekly, ReleaseMonthly:
		return true
	}
	return false
}

// MaxEarlyAccessHours caps the early-access window at a week. Without a cap a
// translator could set it to a year and effectively unpublish their work from
// everyone who has not subscribed.
const MaxEarlyAccessHours = 168

// MaxPricePerChapter mirrors the per-chapter price ceiling.
const MaxPricePerChapter = 999

// ValidateSettings checks the monetisation and presentation fields.
func ValidateSettings(n NovelDraft) error {
	if n.SourceChaptersCount != nil && (*n.SourceChaptersCount < 0 || *n.SourceChaptersCount > 100000) {
		return ErrInvalidInput
	}
	if n.PricePerChapter != nil && (*n.PricePerChapter < 0 || *n.PricePerChapter > MaxPricePerChapter) {
		return ErrInvalidPrice
	}
	if n.FreeUntilChapter != nil && *n.FreeUntilChapter < 0 {
		return ErrInvalidInput
	}
	if n.EarlyAccessHours != nil && (*n.EarlyAccessHours < 0 || *n.EarlyAccessHours > MaxEarlyAccessHours) {
		return ErrInvalidInput
	}
	if n.ReleaseSchedule != "" && !ValidReleaseSchedule(n.ReleaseSchedule) {
		return ErrInvalidInput
	}
	if n.CoverStyle != "" && !ValidCoverStyle(n.CoverStyle) {
		return ErrInvalidInput
	}
	if len([]rune(n.CoverText)) > 40 {
		return ErrInvalidInput
	}
	if n.CoverColor != "" && !ValidCoverColor(n.CoverColor) {
		return ErrInvalidInput
	}
	return nil
}

// Cover styles, mirroring the CHECK on novels.cover_style. CoverImage means
// "render cover_url"; the rest are generated from cover_color and cover_text,
// so a translator with no artwork still gets a distinctive cover.
const (
	CoverImage = "image"
	CoverInk   = "ink"
	CoverSeal  = "seal"
	CoverBrush = "brush"
	CoverPlain = "plain"
)

// ValidCoverStyle reports whether s is one of the offered styles.
func ValidCoverStyle(s string) bool {
	switch s {
	case CoverImage, CoverInk, CoverSeal, CoverBrush, CoverPlain:
		return true
	}
	return false
}

// ValidCoverColor reports whether c is a six-digit hex colour, matching the
// CHECK on novels.cover_color. The editor offers a curated palette; the column
// accepts any hex so the palette can change without a migration.
func ValidCoverColor(c string) bool {
	if len(c) != 7 || c[0] != '#' {
		return false
	}
	for i := 1; i < len(c); i++ {
		switch ch := c[i]; {
		case ch >= '0' && ch <= '9',
			ch >= 'a' && ch <= 'f',
			ch >= 'A' && ch <= 'F':
		default:
			return false
		}
	}
	return true
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
