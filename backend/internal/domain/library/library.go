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
