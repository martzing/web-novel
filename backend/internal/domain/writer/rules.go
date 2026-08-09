package writer

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ResolveArcID picks the arc whose chapter range contains chapterNo, so a
// writer never has to assign an arc by hand (W-02). Returns nil when no arc
// covers the number.
func ResolveArcID(arcs []Arc, chapterNo int) *int64 {
	for _, arc := range arcs {
		if chapterNo >= arc.FromChapterNo && chapterNo <= arc.ToChapterNo {
			id := arc.ID
			return &id
		}
	}
	return nil
}

// PruneRevisions returns the ids to delete so only the newest `keep` autosave
// snapshots survive (I-WR-01). Input order is irrelevant; the newest are
// decided by SavedAt, with the id as a tiebreaker so the result is stable.
func PruneRevisions(revs []Revision, keep int) []int64 {
	if keep < 0 {
		keep = 0
	}
	if len(revs) <= keep {
		return nil
	}

	sorted := slices.Clone(revs)
	// Newest first, with the id breaking ties so the result is deterministic
	// when several autosaves share a timestamp.
	slices.SortFunc(sorted, func(a, b Revision) int {
		if c := b.SavedAt.Compare(a.SavedAt); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})

	doomed := make([]int64, 0, len(sorted)-keep)
	for _, rev := range sorted[keep:] {
		doomed = append(doomed, rev.ID)
	}
	return doomed
}

// PublishDecision derives the status and published_at for a publish request
// (W-06 / I-WR-02).
//
// A future scheduled_at yields "scheduled", which readers cannot see until the
// scheduler flips it; anything else publishes immediately.
func PublishDecision(scheduledAt *time.Time, now time.Time) (status string, publishedAt *time.Time) {
	if scheduledAt != nil && scheduledAt.After(now) {
		return StatusScheduled, nil
	}
	published := now
	return StatusPublished, &published
}

// PublicAt is when a published chapter becomes visible to readers who are not
// auto-unlock subscribers.
//
// It is snapshotted at publish time rather than derived from the novel's
// setting at read time, so a translator changing the window later cannot
// retroactively un-publish chapters that readers can already see.
func PublicAt(publishedAt time.Time, earlyAccessHours int) time.Time {
	if earlyAccessHours <= 0 {
		return publishedAt
	}
	return publishedAt.Add(time.Duration(earlyAccessHours) * time.Hour)
}

// SlugFromTitle derives a URL-safe slug (W-01).
//
// Thai has no ASCII transliteration here, so a Thai-only title yields no ASCII
// characters at all; the sequence suffix guarantees a usable, unique slug. An
// all-numeric slug is never produced, because `/novels/:id` resolves numeric
// parameters as ids and a numeric slug would be unreachable.
func SlugFromTitle(title string, seq int64) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_':
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return fmt.Sprintf("novel-%d", seq)
	}
	// A numeric slug would be shadowed by the id route.
	if _, err := strconv.ParseInt(slug, 10, 64); err == nil {
		return fmt.Sprintf("novel-%s", slug)
	}
	if seq > 0 {
		return fmt.Sprintf("%s-%d", slug, seq)
	}
	return slug
}

// TrendPct is the percentage change between two windows, used for the "+12.4%"
// indicator on the stats tiles (W-08). Growth from nothing is reported as 100%
// rather than infinity.
func TrendPct(current, previous int) float64 {
	switch {
	case previous == 0 && current == 0:
		return 0
	case previous == 0:
		return 100
	default:
		return float64(current-previous) / float64(previous) * 100
	}
}

// Aggregate folds daily rows into the KPI tiles, comparing the current window
// against the preceding one of the same length (I-WR-04).
func Aggregate(current, previous []DailyPoint, from, to time.Time) NovelStats {
	stats := NovelStats{
		Series:     current,
		PeriodFrom: from,
		PeriodTo:   to,
	}
	if stats.Series == nil {
		stats.Series = []DailyPoint{}
	}

	var prevReads, prevCoins, completions int
	for _, p := range current {
		stats.Reads += p.Reads
		stats.CoinsEarned += p.CoinsEarned
		stats.Followers += p.Followers
		completions += p.Completions
	}
	for _, p := range previous {
		prevReads += p.Reads
		prevCoins += p.CoinsEarned
	}

	stats.ReadsTrendPct = TrendPct(stats.Reads, prevReads)
	stats.CoinsTrendPct = TrendPct(stats.CoinsEarned, prevCoins)
	// อ่านจบต่อบท. Zero reads yields 0 rather than a division by zero — a
	// window with no traffic has no completion rate to report, and 0% is the
	// honest answer for an empty tile.
	if stats.Reads > 0 {
		stats.AvgCompletePct = float64(completions) / float64(stats.Reads) * 100
	}
	return stats
}
