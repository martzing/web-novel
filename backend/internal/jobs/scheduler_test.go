package jobs

import (
	"testing"
	"time"
)

func bangkok(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		t.Skipf("Asia/Bangkok tzdata unavailable: %v", err)
	}
	return loc
}

// The PRD pins bonus expiry to 03:00 Asia/Bangkok, so the calculation has to be
// correct regardless of the server's own timezone.
func TestNextDailyAt_BangkokMidnightRollover(t *testing.T) {
	loc := bangkok(t)

	tests := []struct {
		name string
		from time.Time
		want time.Time
	}{
		{
			name: "before the hour, same day",
			from: time.Date(2026, 8, 8, 1, 0, 0, 0, loc),
			want: time.Date(2026, 8, 8, 3, 0, 0, 0, loc),
		},
		{
			name: "after the hour, next day",
			from: time.Date(2026, 8, 8, 4, 0, 0, 0, loc),
			want: time.Date(2026, 8, 9, 3, 0, 0, 0, loc),
		},
		{
			// Exactly at the boundary must move on, or the job would fire in a
			// tight loop.
			name: "exactly at the hour, next day",
			from: time.Date(2026, 8, 8, 3, 0, 0, 0, loc),
			want: time.Date(2026, 8, 9, 3, 0, 0, 0, loc),
		},
		{
			name: "crosses a month boundary",
			from: time.Date(2026, 8, 31, 23, 0, 0, 0, loc),
			want: time.Date(2026, 9, 1, 3, 0, 0, 0, loc),
		},
		{
			name: "crosses a year boundary",
			from: time.Date(2026, 12, 31, 23, 0, 0, 0, loc),
			want: time.Date(2027, 1, 1, 3, 0, 0, 0, loc),
		},
		{
			// 20:30 UTC is already 03:30 the next day in Bangkok, so the next
			// 03:00 there is the day after.
			name: "a UTC instant is compared in Bangkok time",
			from: time.Date(2026, 8, 8, 20, 30, 0, 0, time.UTC),
			want: time.Date(2026, 8, 10, 3, 0, 0, 0, loc),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NextDailyAt(tc.from, loc, 3, 0)
			if !got.Equal(tc.want) {
				t.Fatalf("NextDailyAt(%v) = %v, want %v", tc.from, got, tc.want)
			}
			if !got.After(tc.from) {
				t.Fatalf("the next run must be strictly after the reference time")
			}
		})
	}
}

func TestNextDailyAt_DefaultsToUTC(t *testing.T) {
	from := time.Date(2026, 8, 8, 1, 0, 0, 0, time.UTC)
	got := NextDailyAt(from, nil, 3, 0)
	want := time.Date(2026, 8, 8, 3, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("NextDailyAt = %v, want %v", got, want)
	}
}

func TestNextWeeklyAt(t *testing.T) {
	loc := bangkok(t)

	// 2026-08-08 is a Saturday; the next Monday is the 10th.
	from := time.Date(2026, 8, 8, 12, 0, 0, 0, loc)
	got := NextWeeklyAt(from, loc, time.Monday, 4, 0)
	want := time.Date(2026, 8, 10, 4, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("NextWeeklyAt = %v, want %v", got, want)
	}
	if got.Weekday() != time.Monday {
		t.Fatalf("weekday = %v, want Monday", got.Weekday())
	}

	// From Monday morning before the hour, it fires the same day.
	sameDay := NextWeeklyAt(time.Date(2026, 8, 10, 1, 0, 0, 0, loc), loc, time.Monday, 4, 0)
	if !sameDay.Equal(want) {
		t.Fatalf("NextWeeklyAt = %v, want the same Monday %v", sameDay, want)
	}

	// From Monday afternoon, it waits a whole week.
	nextWeek := NextWeeklyAt(time.Date(2026, 8, 10, 9, 0, 0, 0, loc), loc, time.Monday, 4, 0)
	if !nextWeek.Equal(want.AddDate(0, 0, 7)) {
		t.Fatalf("NextWeeklyAt = %v, want %v", nextWeek, want.AddDate(0, 0, 7))
	}
}

func TestNextMonthlyAt(t *testing.T) {
	loc := time.UTC

	t.Run("later this month", func(t *testing.T) {
		from := time.Date(2026, 8, 8, 12, 0, 0, 0, loc)
		got := NextMonthlyAt(from, loc, 25, 2, 0)
		want := time.Date(2026, 8, 25, 2, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("NextMonthlyAt = %v, want %v", got, want)
		}
	})

	t.Run("rolls into next month", func(t *testing.T) {
		from := time.Date(2026, 8, 26, 12, 0, 0, 0, loc)
		got := NextMonthlyAt(from, loc, 25, 2, 0)
		want := time.Date(2026, 9, 25, 2, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("NextMonthlyAt = %v, want %v", got, want)
		}
	})

	// A day-of-month beyond the month's length must clamp, not overflow into
	// the following month.
	t.Run("day 31 clamps in a 30-day month", func(t *testing.T) {
		from := time.Date(2026, 9, 1, 0, 0, 0, 0, loc)
		got := NextMonthlyAt(from, loc, 31, 2, 0)
		want := time.Date(2026, 9, 30, 2, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("NextMonthlyAt = %v, want %v", got, want)
		}
	})

	t.Run("day 31 clamps in February", func(t *testing.T) {
		from := time.Date(2026, 2, 1, 0, 0, 0, 0, loc)
		got := NextMonthlyAt(from, loc, 31, 2, 0)
		want := time.Date(2026, 2, 28, 2, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("NextMonthlyAt = %v, want %v", got, want)
		}
	})

	t.Run("crosses the year boundary", func(t *testing.T) {
		from := time.Date(2026, 12, 26, 0, 0, 0, 0, loc)
		got := NextMonthlyAt(from, loc, 25, 2, 0)
		want := time.Date(2027, 1, 25, 2, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("NextMonthlyAt = %v, want %v", got, want)
		}
	})
}

func TestEvery(t *testing.T) {
	from := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	got := Every(30 * time.Second)(from)
	if !got.Equal(from.Add(30 * time.Second)) {
		t.Fatalf("Every = %v, want %v", got, from.Add(30*time.Second))
	}
}

func TestEarliest(t *testing.T) {
	base := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	times := []time.Time{
		base.Add(time.Hour),
		base.Add(time.Minute),
		base.Add(time.Hour * 2),
	}

	got, idx := earliest(times)
	if idx != 1 || !got.Equal(base.Add(time.Minute)) {
		t.Fatalf("earliest = %v at %d, want the one-minute entry at index 1", got, idx)
	}
}
