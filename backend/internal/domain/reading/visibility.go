package reading

import (
	"time"

	"github.com/mokchan/webnovel-backend/internal/domain/roles"
)

// Visibility is what a viewer may see of a chapter.
//
// It sits beside Decide rather than inside it: timing and entitlement are
// orthogonal axes — when a chapter is available at all, versus whether this
// reader has paid for it — and keeping them apart lets each be table-tested on
// its own.
type Visibility int

const (
	// VisibleHidden means the chapter is not listed and a read is a 404.
	// Drafts and scheduled work.
	VisibleHidden Visibility = iota
	// VisibleTeaser means the chapter is listed with its metadata but the body
	// is always withheld: an early-access chapter seen by someone with no claim
	// to it. It is deliberately listed rather than hidden, because it is the
	// conversion surface for auto-unlock, and because novels.chapters_count is
	// a stored column that cannot be viewer-dependent.
	VisibleTeaser
	// VisibleFull means readable, subject to Decide.
	VisibleFull
)

// String renders the state for the wire, so a client can explain the paywall.
func (v Visibility) String() string {
	switch v {
	case VisibleTeaser:
		return "early_access"
	case VisibleFull:
		return "public"
	default:
		return "hidden"
	}
}

// Availability is the viewer-independent timing of a chapter.
type Availability struct {
	Status      string
	PublishedAt *time.Time
	// PublicAt is when non-subscribers may read it; nil means immediately.
	PublicAt *time.Time
}

// Access is the viewer's claim on a chapter.
type Access struct {
	Viewer Viewer
	// Subscribed reports auto-unlock being on for this novel, which is what
	// grants the early-access window.
	Subscribed bool
	// Owns reports an existing chapter_unlocks row. Ownership survives
	// unsubscribing: a reader who auto-unlocked and then cancelled keeps what
	// they paid for.
	Owns         bool
	IsTranslator bool
}

// See reports what the viewer may see of a chapter at `now`:
//
//	not published                              -> hidden (its translator or an admin: full)
//	published, past PublicAt (or PublicAt nil) -> full
//	published, inside the early window, and
//	  subscriber | owner | translator | admin  -> full
//	published, inside the early window, other  -> teaser
func See(a Availability, ac Access, now time.Time) Visibility {
	privileged := ac.IsTranslator || ac.Viewer.HasRole(roles.Admin)

	if a.Status != StatusPublished {
		if privileged {
			return VisibleFull
		}
		return VisibleHidden
	}

	if a.PublicAt == nil || !a.PublicAt.After(now) {
		return VisibleFull
	}

	if privileged || ac.Subscribed || ac.Owns {
		return VisibleFull
	}
	return VisibleTeaser
}

// Chapter statuses, mirroring the CHECK on chapters.status.
const (
	StatusDraft     = "draft"
	StatusScheduled = "scheduled"
	StatusPublished = "published"
)
