// Package jobs holds the background work: bonus expiry, glossary re-rendering,
// scheduled publishing, stats rollups, ranking snapshots and housekeeping.
//
// Every job takes `now` as a parameter rather than calling time.Now, so each
// one is deterministic under test.
package jobs

import (
	"context"
	"log/slog"
	"time"
)

// Report summarises what a run did.
type Report struct {
	Name      string
	Processed int
	Skipped   bool
}

// Runner is one unit of background work.
type Runner interface {
	Name() string
	Run(ctx context.Context, now time.Time) (Report, error)
}

// Schedule pairs a job with the rule that decides when it next runs.
type Schedule struct {
	Job Runner
	// Next returns the first run strictly after `after`.
	Next func(after time.Time) time.Time
}

// Scheduler runs jobs on their schedules in a single goroutine.
type Scheduler struct {
	schedules []Schedule
	now       func() time.Time
	logger    *slog.Logger
}

// NewScheduler builds a scheduler.
func NewScheduler(now func() time.Time, logger *slog.Logger) *Scheduler {
	if now == nil {
		now = time.Now
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{now: now, logger: logger}
}

// Add registers a job.
func (s *Scheduler) Add(job Runner, next func(after time.Time) time.Time) {
	s.schedules = append(s.schedules, Schedule{Job: job, Next: next})
}

// Run blocks until the context is cancelled, running each job at its due time.
func (s *Scheduler) Run(ctx context.Context) {
	if len(s.schedules) == 0 {
		<-ctx.Done()
		return
	}

	due := make([]time.Time, len(s.schedules))
	for i, sched := range s.schedules {
		due[i] = sched.Next(s.now())
	}

	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	for {
		soonest, idx := earliest(due)
		wait := max(soonest.Sub(s.now()), 0)

		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(wait)

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.runOne(ctx, s.schedules[idx].Job)
			due[idx] = s.schedules[idx].Next(s.now())
		}
	}
}

// RunNow executes every registered job once, in order. Used by the worker's
// one-shot mode and by tests.
func (s *Scheduler) RunNow(ctx context.Context) {
	for _, sched := range s.schedules {
		s.runOne(ctx, sched.Job)
	}
}

func (s *Scheduler) runOne(ctx context.Context, job Runner) {
	start := s.now()
	report, err := job.Run(ctx, start)
	if err != nil {
		// One failing job must never stop the scheduler.
		s.logger.Error("job failed", "job", job.Name(), "err", err)
		return
	}
	if report.Skipped {
		s.logger.Debug("job skipped", "job", job.Name())
		return
	}
	if report.Processed > 0 {
		s.logger.Info("job finished", "job", job.Name(), "processed", report.Processed)
	}
}

func earliest(times []time.Time) (time.Time, int) {
	best, idx := times[0], 0
	for i, t := range times[1:] {
		if t.Before(best) {
			best, idx = t, i+1
		}
	}
	return best, idx
}

// Every returns a schedule firing at a fixed interval.
func Every(d time.Duration) func(after time.Time) time.Time {
	return func(after time.Time) time.Time { return after.Add(d) }
}

// NextDailyAt returns the next occurrence of hour:min in loc, strictly after
// `after`.
//
// The comparison happens in loc, which is what makes "03:00 Asia/Bangkok"
// correct regardless of the server's own timezone.
func NextDailyAt(after time.Time, loc *time.Location, hour, minute int) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := after.In(loc)
	candidate := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, loc)
	if !candidate.After(local) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate
}

// NextWeeklyAt returns the next occurrence of weekday at hour:min in loc,
// strictly after `after`.
func NextWeeklyAt(after time.Time, loc *time.Location, weekday time.Weekday, hour, minute int) time.Time {
	candidate := NextDailyAt(after, loc, hour, minute)
	for candidate.Weekday() != weekday {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate
}

// NextMonthlyAt returns the next occurrence of the given day-of-month at
// hour:min in loc. A day beyond the month's length lands on its last day.
func NextMonthlyAt(after time.Time, loc *time.Location, day, hour, minute int) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := after.In(loc)

	candidate := monthlyCandidate(local.Year(), local.Month(), day, hour, minute, loc)
	if !candidate.After(local) {
		year, month := local.Year(), local.Month()+1
		if month > time.December {
			year, month = year+1, time.January
		}
		candidate = monthlyCandidate(year, month, day, hour, minute, loc)
	}
	return candidate
}

func monthlyCandidate(year int, month time.Month, day, hour, minute int, loc *time.Location) time.Time {
	// Day 0 of the following month is the last day of this one.
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
	return time.Date(year, month, min(day, lastDay), hour, minute, 0, 0, loc)
}

// Bangkok is the platform's operational timezone; the PRD pins the nightly
// bonus expiry to 03:00 there.
func Bangkok() *time.Location {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		// Fall back to the fixed offset rather than failing to schedule at all;
		// Thailand does not observe daylight saving.
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}
