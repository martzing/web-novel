// Package library is the domain layer for a reader's personal shelf:
// library entries, bookmarks and follows.
//
// These three live together because they are the same shape — a per-user
// relation to a novel — and share one ownership rule: the row belongs to the
// authenticated caller and nobody else may see or delete it.
package library

import (
	"slices"
	"time"
)

// Shelf tabs, mirroring the CHECK constraint on library_entries.status.
const (
	StatusReading = "reading"
	StatusSaved   = "saved"
	StatusDone    = "done"
)

// Statuses lists every valid shelf tab.
var Statuses = []string{StatusReading, StatusSaved, StatusDone}

// ValidStatus reports whether s is a known shelf tab.
func ValidStatus(s string) bool { return slices.Contains(Statuses, s) }

// Series-follow states. ติดตามทั้งชุด is a fan-out over per-novel follows
// rather than a row of its own, so "following a series" is a summary of its
// books rather than a fact — hence three states instead of a boolean.
const (
	// SeriesFollowNone means the reader follows none of the books.
	SeriesFollowNone = "none"
	// SeriesFollowPartial means some books are followed. A reader reaches this
	// state by following one book directly, or by a new book joining a series
	// they had already followed — joining does not follow it for them.
	SeriesFollowPartial = "partial"
	// SeriesFollowAll means every book in the series is followed.
	SeriesFollowAll = "all"
)

// SeriesFollow summarises a reader's follow state across a series' books.
type SeriesFollow struct {
	State     string
	Total     int
	Following int
}

// SummariseSeriesFollow turns "n of total books followed" into a state. It is
// pure so the three-way rule can be tested without a database.
func SummariseSeriesFollow(following, total int) SeriesFollow {
	s := SeriesFollow{State: SeriesFollowNone, Total: total, Following: following}
	switch {
	case total == 0 || following == 0:
		s.State = SeriesFollowNone
	case following >= total:
		s.State = SeriesFollowAll
	default:
		s.State = SeriesFollowPartial
	}
	return s
}

// Entry is one novel on a reader's shelf.
type Entry struct {
	UserID  int64
	NovelID int64
	Status  string
	AddedAt time.Time
}

// EntryWithNovel is a shelf row joined with the display fields the shelf UI
// needs, plus the reader's progress in that novel.
type EntryWithNovel struct {
	Entry
	Slug          string
	TitleTH       string
	TitleCN       string
	CoverURL      string
	ChaptersCount int
	LastChapterNo *int
	Pct           float64

	// SourceChaptersCount lets the shelf show "บทที่ 87 จาก 88 บทที่แปลแล้ว"
	// against the original's length rather than a bare percentage.
	SourceChaptersCount int

	// LastChapterID is what the continue button opens. Without it the shelf
	// can only link to the novel, which is not "resume where I left off".
	LastChapterID *int64

	CoverStyle string
	CoverColor string
	CoverText  string
}

// Counts drives the shelf tab badges.
type Counts struct {
	Reading int
	Saved   int
	Done    int
}

// Bookmark is a saved paragraph position.
type Bookmark struct {
	ID         int64
	UserID     int64
	NovelID    int64
	ChapterID  int64
	ParaAnchor int
	Excerpt    string
	Note       string
	CreatedAt  time.Time

	ChapterNo int
	Title     string
}

// MaxExcerptRunes bounds the stored excerpt so a bookmark cannot be used to
// smuggle a whole paid chapter out of the reader.
const MaxExcerptRunes = 500

// Follow is a subscription to a novel's new chapters.
type Follow struct {
	UserID  int64
	NovelID int64
	Since   time.Time
}
