// Package ratelimit provides in-process abuse controls.
//
// Deliberately in-process rather than backed by a shared store: the system has
// no other cross-process cache dependency, and both target scenarios are
// single-process. With N API replicas the effective limit is N times the
// configured rate; production should also rate-limit at the edge (nginx
// `limit_req`).
package ratelimit

import (
	"sync"
	"time"
)

// Limiter decides whether an action keyed by a string may proceed.
type Limiter interface {
	Allow(key string) (allowed bool, retryAfter time.Duration)
}

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// TokenBucket allows `rate` events per `per` window, with burst capacity.
type TokenBucket struct {
	mu      sync.Mutex
	buckets map[string]*bucket

	rate  float64 // tokens per second
	burst float64
	per   time.Duration
	now   func() time.Time
}

// NewTokenBucket builds a limiter allowing `rate` events per `per` duration.
// A nil clock defaults to time.Now.
func NewTokenBucket(rate int, per time.Duration, now func() time.Time) *TokenBucket {
	if rate <= 0 {
		rate = 1
	}
	if per <= 0 {
		per = time.Minute
	}
	if now == nil {
		now = time.Now
	}
	return &TokenBucket{
		buckets: make(map[string]*bucket),
		rate:    float64(rate) / per.Seconds(),
		burst:   float64(rate),
		per:     per,
		now:     now,
	}
}

// Allow consumes one token for key.
func (t *TokenBucket) Allow(key string) (bool, time.Duration) {
	now := t.now()

	t.mu.Lock()
	defer t.mu.Unlock()

	t.sweepLocked(now)

	b, ok := t.buckets[key]
	if !ok {
		b = &bucket{tokens: t.burst, lastSeen: now}
		t.buckets[key] = b
	} else {
		b.tokens = min(t.burst, b.tokens+now.Sub(b.lastSeen).Seconds()*t.rate)
		b.lastSeen = now
	}

	if b.tokens < 1 {
		deficit := 1 - b.tokens
		return false, time.Duration(deficit / t.rate * float64(time.Second))
	}
	b.tokens--
	return true, 0
}

// sweepLocked drops buckets that have been idle long enough to have fully
// refilled, so memory stays bounded by the active key set.
func (t *TokenBucket) sweepLocked(now time.Time) {
	if len(t.buckets) < 1024 {
		return
	}
	for k, b := range t.buckets {
		if now.Sub(b.lastSeen) > 2*t.per {
			delete(t.buckets, k)
		}
	}
}

// DistinctCounter limits how many *distinct* values a key may touch inside a
// rolling window.
//
// This is what "no more than 20 distinct chapter bodies per user per minute"
// requires: re-reading one chapter a hundred times stays allowed, while
// sweeping the catalogue does not.
type DistinctCounter struct {
	mu   sync.Mutex
	seen map[string]map[string]time.Time

	limit  int
	window time.Duration
	now    func() time.Time
}

// NewDistinctCounter builds a distinct-value limiter. A nil clock defaults to
// time.Now.
func NewDistinctCounter(limit int, window time.Duration, now func() time.Time) *DistinctCounter {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	if now == nil {
		now = time.Now
	}
	return &DistinctCounter{
		seen:   make(map[string]map[string]time.Time),
		limit:  limit,
		window: window,
		now:    now,
	}
}

// Observe records that key touched value and reports whether it may proceed.
func (d *DistinctCounter) Observe(key, value string) (bool, time.Duration) {
	now := d.now()
	cutoff := now.Add(-d.window)

	d.mu.Lock()
	defer d.mu.Unlock()

	values, ok := d.seen[key]
	if !ok {
		values = make(map[string]time.Time)
		d.seen[key] = values
	}

	oldest := now
	for v, at := range values {
		if at.Before(cutoff) {
			delete(values, v)
			continue
		}
		if at.Before(oldest) {
			oldest = at
		}
	}

	// A value already inside the window is free to repeat.
	if _, repeat := values[value]; repeat {
		values[value] = now
		return true, 0
	}

	if len(values) >= d.limit {
		return false, oldest.Add(d.window).Sub(now)
	}
	values[value] = now
	d.sweepLocked(cutoff)
	return true, 0
}

func (d *DistinctCounter) sweepLocked(cutoff time.Time) {
	if len(d.seen) < 1024 {
		return
	}
	for k, values := range d.seen {
		for v, at := range values {
			if at.Before(cutoff) {
				delete(values, v)
			}
		}
		if len(values) == 0 {
			delete(d.seen, k)
		}
	}
}
