package writer

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

var base = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

// W-02 — a chapter's arc is derived from its number, not chosen by hand.
func TestResolveArcID(t *testing.T) {
	arcs := []Arc{
		{ID: 1, ArcNo: 1, FromChapterNo: 1, ToChapterNo: 48},
		{ID: 2, ArcNo: 2, FromChapterNo: 49, ToChapterNo: 120},
		{ID: 3, ArcNo: 3, FromChapterNo: 121, ToChapterNo: 186},
	}

	tests := []struct {
		name      string
		chapterNo int
		want      *int64
	}{
		{"first chapter of the first arc", 1, ptr(int64(1))},
		{"last chapter of the first arc", 48, ptr(int64(1))},
		{"first chapter of the second arc", 49, ptr(int64(2))},
		{"middle of the second arc", 87, ptr(int64(2))},
		{"last arc", 186, ptr(int64(3))},
		{"beyond every arc", 999, nil},
		{"before every arc", 0, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveArcID(arcs, tc.chapterNo)
			switch {
			case tc.want == nil && got != nil:
				t.Fatalf("arc = %d, want nil", *got)
			case tc.want != nil && got == nil:
				t.Fatalf("arc = nil, want %d", *tc.want)
			case tc.want != nil && *got != *tc.want:
				t.Fatalf("arc = %d, want %d", *got, *tc.want)
			}
		})
	}

	t.Run("no arcs at all", func(t *testing.T) {
		if got := ResolveArcID(nil, 5); got != nil {
			t.Fatalf("arc = %d, want nil", *got)
		}
	})
}

// I-WR-01 — autosave keeps the newest 20 revisions.
func TestPruneRevisions_Keeps20(t *testing.T) {
	revs := make([]Revision, 0, 25)
	for i := range 25 {
		revs = append(revs, Revision{
			ID:      int64(i + 1),
			SavedAt: base.Add(time.Duration(i) * time.Minute),
		})
	}

	doomed := PruneRevisions(revs, KeepRevisions)
	if len(doomed) != 5 {
		t.Fatalf("deleting %d revisions, want 5", len(doomed))
	}

	// The five oldest go; ids 1..5 by construction.
	want := []int64{5, 4, 3, 2, 1}
	if !reflect.DeepEqual(doomed, want) {
		t.Fatalf("doomed = %v, want the five oldest %v", doomed, want)
	}
}

func TestPruneRevisions_NothingToDoUnderTheLimit(t *testing.T) {
	revs := []Revision{
		{ID: 1, SavedAt: base},
		{ID: 2, SavedAt: base.Add(time.Minute)},
	}
	for _, keep := range []int{2, 3, KeepRevisions} {
		if got := PruneRevisions(revs, keep); got != nil {
			t.Fatalf("keep=%d: doomed = %v, want none", keep, got)
		}
	}
}

// Order of the input must not matter; recency decides.
func TestPruneRevisions_IgnoresInputOrder(t *testing.T) {
	revs := []Revision{
		{ID: 3, SavedAt: base.Add(2 * time.Minute)},
		{ID: 1, SavedAt: base},
		{ID: 4, SavedAt: base.Add(3 * time.Minute)},
		{ID: 2, SavedAt: base.Add(time.Minute)},
	}

	doomed := PruneRevisions(revs, 2)
	want := []int64{2, 1}
	if !reflect.DeepEqual(doomed, want) {
		t.Fatalf("doomed = %v, want the two oldest %v", doomed, want)
	}
}

// Autosaves can share a timestamp; the id keeps the outcome deterministic.
func TestPruneRevisions_TiedTimestampsUseTheID(t *testing.T) {
	revs := []Revision{
		{ID: 1, SavedAt: base},
		{ID: 2, SavedAt: base},
		{ID: 3, SavedAt: base},
	}
	doomed := PruneRevisions(revs, 1)
	want := []int64{2, 1}
	if !reflect.DeepEqual(doomed, want) {
		t.Fatalf("doomed = %v, want %v", doomed, want)
	}
}

// W-06 / I-WR-02 — a future schedule hides the chapter until its time.
func TestPublishDecision(t *testing.T) {
	future := base.Add(time.Hour)
	past := base.Add(-time.Hour)

	tests := []struct {
		name           string
		scheduledAt    *time.Time
		wantStatus     string
		wantPublished  bool
		wantPublishNow bool
	}{
		{"no schedule publishes now", nil, StatusPublished, true, true},
		{"a future schedule waits", &future, StatusScheduled, false, false},
		{"a past schedule publishes now", &past, StatusPublished, true, true},
		{"scheduled for exactly now publishes", &base, StatusPublished, true, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, publishedAt := PublishDecision(tc.scheduledAt, base)
			if status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", status, tc.wantStatus)
			}
			if (publishedAt != nil) != tc.wantPublished {
				t.Fatalf("publishedAt = %v, want set=%v", publishedAt, tc.wantPublished)
			}
			if tc.wantPublishNow && !publishedAt.Equal(base) {
				t.Fatalf("publishedAt = %v, want %v", publishedAt, base)
			}
		})
	}
}

func TestSlugFromTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		seq   int64
		want  string
	}{
		{"ascii title", "Nine Streams Sword Immortal", 0, "nine-streams-sword-immortal"},
		{"collapses separators", "Nine   Streams__Sword", 0, "nine-streams-sword"},
		{"trims leading and trailing dashes", "  -Nine Streams-  ", 0, "nine-streams"},
		{"appends the sequence when given", "Nine Streams", 7, "nine-streams-7"},
		{"drops punctuation", "Nine: Streams! (Vol. 2)", 0, "nine-streams-vol-2"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SlugFromTitle(tc.title, tc.seq); got != tc.want {
				t.Fatalf("SlugFromTitle(%q, %d) = %q, want %q", tc.title, tc.seq, got, tc.want)
			}
		})
	}

	// A Thai-only title has no ASCII to keep, so the sequence carries it.
	t.Run("thai title falls back to the sequence", func(t *testing.T) {
		got := SlugFromTitle("เซียนดาบเก้าสายธาร", 42)
		if got != "novel-42" {
			t.Fatalf("slug = %q, want novel-42", got)
		}
	})

	// `/novels/:id` resolves a numeric parameter as an id, so a numeric slug
	// would be permanently unreachable.
	t.Run("a numeric title never yields a numeric slug", func(t *testing.T) {
		// The invariant is only that the slug must not parse as an integer.
		// "2 0 2 6" becomes "2-0-2-6", which is already unambiguous.
		for _, title := range []string{"12345", "2 0 2 6", "007", "0"} {
			got := SlugFromTitle(title, 0)
			if _, err := strconv.ParseInt(got, 10, 64); err == nil {
				t.Fatalf("slug %q is all-numeric and would be shadowed by the id route", got)
			}
		}

		// An all-digit title has nothing else to disambiguate it, so it gets
		// the explicit prefix.
		if got := SlugFromTitle("12345", 0); !strings.HasPrefix(got, "novel-") {
			t.Fatalf("slug = %q, want a novel- prefix", got)
		}
	})
}

// W-08 — the "+12.4%" indicator.
func TestTrendPct(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		previous int
		want     float64
	}{
		{"growth", 112, 100, 12},
		{"decline", 90, 100, -10},
		{"flat", 100, 100, 0},
		{"both zero", 0, 0, 0},
		// Growth from nothing is capped rather than reported as infinity.
		{"from zero", 50, 0, 100},
		{"to zero", 0, 50, -100},
		{"fractional", 1240, 1103, 12.42066},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TrendPct(tc.current, tc.previous)
			if math.Abs(got-tc.want) > 0.001 {
				t.Fatalf("TrendPct(%d, %d) = %v, want %v", tc.current, tc.previous, got, tc.want)
			}
		})
	}
}

// I-WR-04 — the KPI totals are the sum of the daily rows.
func TestAggregate(t *testing.T) {
	current := []DailyPoint{
		{Day: base, Reads: 100, CoinsEarned: 20, Followers: 3},
		{Day: base.AddDate(0, 0, 1), Reads: 150, CoinsEarned: 30, Followers: 2},
		{Day: base.AddDate(0, 0, 2), Reads: 250, CoinsEarned: 50, Followers: 5},
	}
	previous := []DailyPoint{
		{Day: base.AddDate(0, 0, -3), Reads: 200, CoinsEarned: 50},
		{Day: base.AddDate(0, 0, -2), Reads: 200, CoinsEarned: 50},
	}

	stats := Aggregate(current, previous, base, base.AddDate(0, 0, 2))

	if stats.Reads != 500 {
		t.Fatalf("reads = %d, want 500", stats.Reads)
	}
	if stats.CoinsEarned != 100 {
		t.Fatalf("coins = %d, want 100", stats.CoinsEarned)
	}
	if stats.Followers != 10 {
		t.Fatalf("followers = %d, want 10", stats.Followers)
	}
	// 500 vs 400 = +25%, 100 vs 100 = flat.
	if math.Abs(stats.ReadsTrendPct-25) > 0.001 {
		t.Fatalf("reads trend = %v, want 25", stats.ReadsTrendPct)
	}
	if stats.CoinsTrendPct != 0 {
		t.Fatalf("coins trend = %v, want 0", stats.CoinsTrendPct)
	}
	if len(stats.Series) != 3 {
		t.Fatalf("series has %d points, want 3", len(stats.Series))
	}
}

func TestAggregate_EmptyWindows(t *testing.T) {
	stats := Aggregate(nil, nil, base, base)

	if stats.Reads != 0 || stats.CoinsEarned != 0 {
		t.Fatalf("stats = %+v, want zeroes", stats)
	}
	// A nil series would serialise as JSON null; the UI expects an array.
	if stats.Series == nil {
		t.Fatal("series must be an empty slice, not nil")
	}
}

func ptr[T any](v T) *T { return &v }
