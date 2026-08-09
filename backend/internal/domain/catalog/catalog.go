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
	// ChaptersCount is บทที่แปลแล้ว and SourceChaptersCount is บทในต้นฉบับ. The
	// product shows the pair everywhere, so both travel together.
	ChaptersCount       int
	SourceChaptersCount int
	Genres              []Genre
	// UpdatedAt is RFC3339 and doubles as the keyset cursor value for the
	// "latest" ordering.
	UpdatedAt string

	// Cover template, for novels with no uploaded artwork.
	CoverStyle string
	CoverColor string
	CoverText  string

	SeriesID *int64

	// PrimaryTranslatorID is who may still see this novel when it is hidden.
	PrimaryTranslatorID *int64
}

// IsHidden reports ซ่อนจากหน้าร้าน.
func (n Novel) IsHidden() bool { return n.Status == StatusHidden }

// VisibleTo reports whether a viewer may see the novel at all. Hidden novels
// stay reachable for their own translator, so the works screen can still open
// one; for everyone else they do not exist.
func (n Novel) VisibleTo(viewerID int64) bool {
	if !n.IsHidden() {
		return true
	}
	return viewerID != 0 && n.PrimaryTranslatorID != nil && *n.PrimaryTranslatorID == viewerID
}

// Novel publication statuses, mirroring the CHECK on novels.status.
const (
	StatusOngoing  = "ongoing"
	StatusComplete = "complete"
	StatusHiatus   = "hiatus"
	StatusHidden   = "hidden"
)

// NovelDetail is a Novel with arcs and glossary size attached, used on the
// detail page.
type NovelDetail struct {
	Novel
	Arcs          []Arc
	GlossaryCount int
	CommentsCount int

	// SellByArc and TipsEnabled drive the detail page's buy-arc and tip
	// controls. They are detail-only: the browse cards have no use for them.
	SellByArc   bool
	TipsEnabled bool
	// ReleaseSchedule is รอบปล่อยบทใหม่, shown so a reader knows when to
	// return. It is metadata, not a schedule the platform acts on.
	ReleaseSchedule  string
	EarlyAccessHours int
}

// SeriesDetail is the public ชุดหนังสือ page: a collection with its books in
// the translator's reading order.
type SeriesDetail struct {
	ID          int64
	Slug        string
	Title       string
	Description string
	CoverURL    string
	Books       []SeriesEntry
}

// SeriesEntry is one novel's slot in a series' reading order.
type SeriesEntry struct {
	Novel
	Position int
	// Note is the translator's line about where this book sits — "อ่านเล่มนี้
	// ก่อนได้ ไม่สปอยล์" and the like.
	Note string
}

// TranslatedOf reports the pair the product shows on nearly every surface:
// how many chapters exist in Thai against how many exist at the source. A
// source count of zero means the translator has not entered one, and callers
// should show the translated figure alone rather than "N จาก 0".
func (s SeriesDetail) TranslatedOf() (translated, source int) {
	for _, b := range s.Books {
		translated += b.ChaptersCount
		source += b.SourceChaptersCount
	}
	return translated, source
}

// RelatedNovel is one เรื่องเกี่ยวเนื่อง card on the detail or series page.
type RelatedNovel struct {
	Novel
	Kind      string
	KindLabel string
	Note      string
	SortNo    int
}

// Relation kinds, mirroring the CHECK on novel_relations.kind.
const (
	RelationSequel    = "sequel"
	RelationPrequel   = "prequel"
	RelationSpinoff   = "spinoff"
	RelationSideStory = "side_story"
	RelationSameWorld = "same_world"
)

// InverseRelationKind is how the far novel would describe the same link. Only
// same_world is symmetric; sequel and prequel invert; spinoff and side_story
// have no natural inverse and surface as same_world rather than asserting a
// relationship the translator never declared.
func InverseRelationKind(kind string) string {
	switch kind {
	case RelationSequel:
		return RelationPrequel
	case RelationPrequel:
		return RelationSequel
	default:
		return RelationSameWorld
	}
}

// RelationKindLabelTH is the Thai heading related works are grouped under.
func RelationKindLabelTH(kind string) string {
	switch kind {
	case RelationSequel:
		return "ภาคต่อโดยตรง"
	case RelationPrequel:
		return "ปฐมบท"
	case RelationSpinoff:
		return "ภาคแยก"
	case RelationSideStory:
		return "ภาคพิเศษ"
	case RelationSameWorld:
		return "เกิดในโลกเดียวกัน"
	}
	return kind
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
