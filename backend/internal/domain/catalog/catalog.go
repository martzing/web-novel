// Package catalog is the domain layer for novels, chapters and glossaries.
// It holds business types and the repository port; it must not import any
// storage or transport package.
package catalog

// Genre is a discovery tag attached to novels.
type Genre struct {
	ID     int64
	Slug   string
	NameTH string
}

// Novel is the aggregate root of a translated work.
type Novel struct {
	ID             int64
	Slug           string
	TitleTH        string
	TitleCN        string
	AuthorName     string
	Description    string
	CoverURL       string
	Status         string
	RatingAvg      float64
	RatingCount    int
	FollowersCount int
	ChaptersCount  int
	Genres         []Genre
	// UpdatedAt is RFC3339 and doubles as the keyset cursor value for the
	// "latest" ordering.
	UpdatedAt string
}

// NovelDetail is a Novel with arcs and glossary size attached, used on the
// detail page.
type NovelDetail struct {
	Novel
	Arcs          []Arc
	GlossaryCount int
	CommentsCount int
}

// Arc groups a range of chapters under a story section (ภาค).
type Arc struct {
	ID            int64
	NovelID       int64
	ArcNo         int
	Name          string
	FromChapterNo int
	ToChapterNo   int
}

// Chapter is a single translated chapter's metadata.
type Chapter struct {
	ID          int64
	NovelID     int64
	ArcID       *int64
	ChapterNo   int
	Title       string
	PriceCoins  int
	WordCount   int
	PublishedAt string

	// Unlocked is filled in for authenticated table-of-contents requests so the
	// UI can show an "อ่านต่อ" affordance instead of a lock on owned chapters.
	Unlocked bool
}

// GlossaryGroup is one category of glossary entries within a novel.
type GlossaryGroup struct {
	ID      int64
	NovelID int64
	Name    string
	SortNo  int
	Entries []GlossaryEntry
}

// GlossaryEntry is a single term, keyed by TermKey which is what the inline
// <span data-k="..."> in a rendered chapter body refers to.
type GlossaryEntry struct {
	ID      int64
	GroupID int64
	TermKey string
	TitleTH string
	TitleCN string
	Body    string
	Kind    string
}

// RankedNovel is one row of the weekly leaderboard.
type RankedNovel struct {
	Novel
	Rank  int
	Score float64
}

// NovelFilter carries the query parameters used to list novels.
type NovelFilter struct {
	Query     string
	GenreSlug string
	Sort      string // "popular" | "latest"
	Limit     int
	// AfterID is the keyset cursor position: only novels ordered strictly after
	// this id are returned.
	AfterID int64
	// AfterValue is the sort-key of the cursor row (updated_at for "latest",
	// followers_count for "popular").
	AfterValue string
}

// Sort orders accepted by NovelFilter.
const (
	SortPopular = "popular"
	SortLatest  = "latest"
)
