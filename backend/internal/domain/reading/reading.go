// Package reading is the domain layer for serving chapter bodies, tracking
// reading position, and recording read events. It owns the entitlement
// decision — the single place `locked` is computed.
package reading

import (
	"slices"
	"time"
)

// Viewer is the caller identity. The zero value is anonymous.
type Viewer struct {
	UserID int64
	Roles  []string
}

// IsAnonymous reports whether no user is attached.
func (v Viewer) IsAnonymous() bool { return v.UserID == 0 }

// HasRole reports whether the viewer carries the given role.
func (v Viewer) HasRole(role string) bool { return slices.Contains(v.Roles, role) }

// Chapter is chapter metadata as the reader path needs it, including the
// fields that drive the entitlement decision.
type Chapter struct {
	ID           int64
	NovelID      int64
	NovelSlug    string
	NovelTitleTH string
	ArcID        *int64
	ArcNo        int
	ArcName      string
	ChapterNo    int
	Title        string
	PriceCoins   int
	WordCount    int
	Status       string
	TranslatorID *int64
	PublishedAt  string

	// PublishedAtTime and PublicAtTime carry the raw instants the visibility
	// rule needs; PublishedAt above is the pre-formatted wire value.
	PublishedAtTime *time.Time
	PublicAtTime    *time.Time

	// TipsEnabled comes from the novel and rides along here so the reader can
	// decide whether to show the tip control without a second request.
	TipsEnabled bool
}

// ChapterView is what a reader receives: metadata plus the body when entitled.
type ChapterView struct {
	Chapter
	Locked bool
	// LockedReason distinguishes a paywall from an early-access window, so the
	// UI can offer the right call to action: buy, or subscribe.
	LockedReason string
	BodyHTML     string
	PrevID       *int64
	NextID       *int64
}

// Locked reasons.
const (
	LockedReasonPaywall     = "paywall"
	LockedReasonEarlyAccess = "early_access"
)

// Progress is a reader's position in one novel.
type Progress struct {
	UserID        int64
	NovelID       int64
	LastChapterID *int64
	LastChapterNo *int
	ParaAnchor    int
	Pct           float64
	UpdatedAt     time.Time
}

// ReadEvent is a fire-and-forget record that a chapter was opened.
type ReadEvent struct {
	ChapterID int64
	UserID    *int64
	SessionID *string
	// Completed marks a read that reached the end of the chapter. It feeds the
	// อ่านจบต่อบท writer KPI, and lives on the event rather than being derived
	// from reading_progress, which is per novel and mutable and so cannot say
	// how many of a given day's reads finished.
	Completed  bool
	OccurredAt time.Time
}
