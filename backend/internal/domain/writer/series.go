package writer

import "strings"

// Series is a collection of a translator's novels — ชุดหนังสือ.
type Series struct {
	ID          int64
	OwnerUserID *int64
	Slug        string
	Title       string
	Description string
	CoverURL    string
	// BookCount is filled by list reads so the work tree can label a series
	// without a second query per row.
	BookCount int
}

// SeriesBook is one novel's placement in a series' reading order.
//
// The order lives on novels (series_position, series_note) rather than in a
// join table because a novel belongs to at most one series — the design's
// series picker is single-select — so a join table would carry a uniqueness
// constraint and no extra information.
type SeriesBook struct {
	NovelID  int64
	Position int
	Note     string

	// Display fields, filled on read.
	Slug                string
	TitleTH             string
	CoverURL            string
	CoverStyle          string
	CoverColor          string
	CoverText           string
	Status              string
	ChaptersCount       int
	SourceChaptersCount int
}

// Relation kinds, mirroring the CHECK on novel_relations.kind.
const (
	RelationSequel    = "sequel"     // ภาคต่อโดยตรง
	RelationPrequel   = "prequel"    // ปฐมบท
	RelationSpinoff   = "spinoff"    // ภาคแยก
	RelationSideStory = "side_story" // ภาคพิเศษ
	RelationSameWorld = "same_world" // เกิดในโลกเดียวกัน
)

// Relation is a directed link from one novel to another — เรื่องเกี่ยวเนื่อง.
//
// It is stored directional: the kind reads from NovelID's point of view, so
// "A sequel B" means A is the sequel. RelationSameWorld is the one kind whose
// inverse is itself, which is why reads mirror it and the others they do not.
type Relation struct {
	NovelID        int64
	RelatedNovelID int64
	Kind           string
	Note           string
	SortNo         int

	// Mirrored marks a relation that is stored on the *other* novel and shown
	// here with its inverse kind. The editor hides its unlink control, because
	// the note and sort order belong to the novel that declared the link.
	Mirrored bool

	// Display fields for the related novel, filled on read.
	RelatedSlug       string
	RelatedTitleTH    string
	RelatedCoverURL   string
	RelatedCoverStyle string
	RelatedCoverColor string
	RelatedCoverText  string
	RelatedStatus     string
}

// ValidRelationKind reports whether kind is one of the five stored kinds.
func ValidRelationKind(kind string) bool {
	switch kind {
	case RelationSequel, RelationPrequel, RelationSpinoff, RelationSideStory, RelationSameWorld:
		return true
	}
	return false
}

// InverseRelationKind is how the *other* novel would describe the link.
//
// Only same_world is symmetric. The rest invert (a sequel's target is a
// prequel), and spinoff/side_story have no natural inverse, so they surface on
// the other novel as same_world rather than claiming a relationship the
// translator did not assert.
func InverseRelationKind(kind string) string {
	switch kind {
	case RelationSequel:
		return RelationPrequel
	case RelationPrequel:
		return RelationSequel
	case RelationSameWorld:
		return RelationSameWorld
	default:
		return RelationSameWorld
	}
}

// RelationKindLabelTH is the Thai label the design groups related works under.
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

// ValidateSeries checks a series before it is written.
func ValidateSeries(s Series) error {
	if strings.TrimSpace(s.Title) == "" {
		return ErrInvalidInput
	}
	// Count runes: a Thai title is far shorter in characters than in bytes, and
	// a byte limit would reject perfectly ordinary names.
	if len([]rune(s.Title)) > 200 {
		return ErrInvalidInput
	}
	if len([]rune(s.Description)) > 2000 {
		return ErrInvalidInput
	}
	return nil
}

// ValidateRelation checks a relation before it is written.
func ValidateRelation(r Relation) error {
	if !ValidRelationKind(r.Kind) {
		return ErrInvalidInput
	}
	// A novel related to itself would render as its own sequel and, worse,
	// recurse when the detail page walks relations.
	if r.NovelID == r.RelatedNovelID {
		return ErrInvalidInput
	}
	if r.NovelID == 0 || r.RelatedNovelID == 0 {
		return ErrInvalidInput
	}
	if len([]rune(r.Note)) > 500 {
		return ErrInvalidInput
	}
	return nil
}

// ReorderPositions renumbers ids to 1..n in the given order.
//
// Positions are rewritten wholesale rather than swapped pairwise: the design's
// drag handle can move an item any distance, and a full rewrite is one
// statement that cannot leave a gap or a duplicate behind.
func ReorderPositions(ids []int64) map[int64]int {
	out := make(map[int64]int, len(ids))
	next := 1
	for _, id := range ids {
		if _, seen := out[id]; seen {
			continue
		}
		out[id] = next
		next++
	}
	return out
}
