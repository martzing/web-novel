package catalog
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
}

// NovelDetail is a Novel with the arc list attached, used on the detail page.
type NovelDetail struct {
	Novel
	Arcs []Arc
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
	PublishedAt string
}

// ChapterView is what a reader receives: chapter metadata plus body when unlocked.
type ChapterView struct {
	Chapter
	Locked   bool
	BodyHTML string
}

// NovelFilter carries the query parameters used to list novels.
type NovelFilter struct {
	Query     string
	GenreSlug string
	Sort      string // "popular" | "latest"
	Limit     int
}
