package ratelimit

import (
	"strconv"
	"testing"
	"time"
)

// fakeClock makes refill behaviour deterministic; no test sleeps.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 8, 8, 3, 0, 0, 0, time.UTC)}
}

func TestTokenBucket_AllowsUpToTheRateThenRejects(t *testing.T) {
	clock := newClock()
	limiter := NewTokenBucket(3, time.Minute, clock.now)

	for i := range 3 {
		if ok, _ := limiter.Allow("1.2.3.4"); !ok {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}
	ok, retryAfter := limiter.Allow("1.2.3.4")
	if ok {
		t.Fatal("the fourth request should have been rejected")
	}
	if retryAfter <= 0 {
		t.Fatalf("retryAfter = %v, want a positive duration", retryAfter)
	}
}

func TestTokenBucket_RefillsWithFakeClock(t *testing.T) {
	clock := newClock()
	limiter := NewTokenBucket(60, time.Minute, clock.now) // one token per second

	for range 60 {
		limiter.Allow("ip")
	}
	if ok, _ := limiter.Allow("ip"); ok {
		t.Fatal("bucket should be empty")
	}

	clock.advance(time.Second)
	if ok, _ := limiter.Allow("ip"); !ok {
		t.Fatal("one second should refill exactly one token")
	}
	if ok, _ := limiter.Allow("ip"); ok {
		t.Fatal("only one token should have been refilled")
	}

	clock.advance(time.Hour)
	for i := range 60 {
		if ok, _ := limiter.Allow("ip"); !ok {
			t.Fatalf("a long idle period should refill to burst, failed at %d", i+1)
		}
	}
}

func TestTokenBucket_KeysAreIndependent(t *testing.T) {
	clock := newClock()
	limiter := NewTokenBucket(1, time.Minute, clock.now)

	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("first key should be allowed")
	}
	if ok, _ := limiter.Allow("a"); ok {
		t.Fatal("first key should now be exhausted")
	}
	if ok, _ := limiter.Allow("b"); !ok {
		t.Fatal("a different key must have its own budget")
	}
}

// I-SEC-05 is about breadth, not volume: re-reading the same chapter must stay
// allowed no matter how often, while sweeping new chapters is capped.
func TestDistinctCounter_AllowsRepeatsOfSameValue(t *testing.T) {
	clock := newClock()
	counter := NewDistinctCounter(20, time.Minute, clock.now)

	for i := range 100 {
		if ok, _ := counter.Observe("user:7", "chapter-1"); !ok {
			t.Fatalf("repeat read %d of the same chapter must be allowed", i+1)
		}
		clock.advance(time.Second)
	}
}

func TestDistinctCounter_CapsDistinctValues(t *testing.T) {
	clock := newClock()
	counter := NewDistinctCounter(20, time.Minute, clock.now)

	for i := range 20 {
		if ok, _ := counter.Observe("user:7", "chapter-"+strconv.Itoa(i)); !ok {
			t.Fatalf("distinct value %d should be allowed", i+1)
		}
	}
	ok, retryAfter := counter.Observe("user:7", "chapter-21")
	if ok {
		t.Fatal("the 21st distinct value should be rejected")
	}
	if retryAfter <= 0 {
		t.Fatalf("retryAfter = %v, want a positive duration", retryAfter)
	}

	// A value already inside the window still passes even at the cap.
	if ok, _ := counter.Observe("user:7", "chapter-0"); !ok {
		t.Fatal("an already-seen value must remain allowed at the cap")
	}
}

func TestDistinctCounter_WindowExpiry(t *testing.T) {
	clock := newClock()
	counter := NewDistinctCounter(2, time.Minute, clock.now)

	counter.Observe("user:7", "a")
	counter.Observe("user:7", "b")
	if ok, _ := counter.Observe("user:7", "c"); ok {
		t.Fatal("third distinct value should be rejected inside the window")
	}

	clock.advance(2 * time.Minute)
	if ok, _ := counter.Observe("user:7", "c"); !ok {
		t.Fatal("the window should have rolled over")
	}
}

func TestDistinctCounter_KeysAreIndependent(t *testing.T) {
	clock := newClock()
	counter := NewDistinctCounter(1, time.Minute, clock.now)

	counter.Observe("user:1", "a")
	if ok, _ := counter.Observe("user:1", "b"); ok {
		t.Fatal("first user should be at the cap")
	}
	if ok, _ := counter.Observe("user:2", "b"); !ok {
		t.Fatal("a different user must have its own budget")
	}
}
