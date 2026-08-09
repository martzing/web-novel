package jobs_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/jobs"
	walletrepo "github.com/mokchan/webnovel-backend/internal/repository/wallet"
	walletsvc "github.com/mokchan/webnovel-backend/internal/service/wallet"
	"github.com/mokchan/webnovel-backend/test/makeme"
)

// recordingNotifier captures the failure notices the job sends.
type recordingNotifier struct {
	mu    sync.Mutex
	calls [][3]int64
}

func (n *recordingNotifier) NotifyAutoUnlockFailed(_ context.Context, userID, novelID, chapterID int64) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.calls = append(n.calls, [3]int64{userID, novelID, chapterID})
	return nil
}

func (n *recordingNotifier) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.calls)
}

// auFixture is one novel with a paid chapter published an hour ago.
type auFixture struct {
	m        *makeme.MakeMe
	wallet   *walletsvc.Service
	notifier *recordingNotifier
	writer   *entities.User
	novel    *entities.Novel
	chapter  *entities.Chapter
}

func newAutoUnlockFixture(t *testing.T) *auFixture {
	t.Helper()
	m := makeme.New(t)
	m.Reset()

	writer := m.ANewUser().Please()
	novel := m.ANewNovel().With(func(n *entities.Novel) {
		n.PrimaryTranslatorID = &writer.ID
	}).Please()

	hourAgo := time.Now().Add(-time.Hour)
	chapter := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = novel.ID
		c.ChapterNo = 1
		c.PriceCoins = 10
		c.Status = entities.ChapterPublished
		c.TranslatorID = &writer.ID
		c.PublishedAt = &hourAgo
	}).Please()

	return &auFixture{
		m:        m,
		wallet:   walletsvc.New(walletrepo.New(m.DB), 720*time.Hour, 30, time.Now),
		notifier: &recordingNotifier{},
		writer:   writer,
		novel:    novel,
		chapter:  chapter,
	}
}

func (f *auFixture) job() *jobs.AutoUnlockJob {
	return &jobs.AutoUnlockJob{
		DB:       f.m.DB,
		Wallet:   f.wallet,
		Notifier: f.notifier,
	}
}

// subscriber creates a reader subscribed to the novel, funded with `coins`.
// The subscription is backdated so the chapter counts as published after it.
func (f *auFixture) subscriber(t *testing.T, coins, cap int) *entities.User {
	t.Helper()
	user := f.m.ANewUser().Please()

	if coins > 0 {
		f.m.ANewWalletBalance().With(func(w *entities.WalletBalance) {
			w.UserID = user.ID
			w.Balance = coins
		}).Please()
		f.m.ANewCoinLedgerEntry().With(func(e *entities.CoinLedgerEntry) {
			e.UserID = user.ID
			e.Kind = entities.LedgerTopup
			e.Delta = coins
			e.BalanceAfter = coins
		}).Please()
	}

	dayAgo := time.Now().Add(-24 * time.Hour)
	f.m.ANewAutoUnlockSubscription().With(func(s *entities.AutoUnlockSubscription) {
		s.UserID = user.ID
		s.NovelID = f.novel.ID
		s.Active = true
		s.MaxCoinsPerChapter = int16(cap)
		s.CreatedAt = dayAgo
	}).Please()

	return user
}

func (f *auFixture) unlockCount(t *testing.T, userID int64) int64 {
	t.Helper()
	var n int64
	if err := f.m.DB.Model(&entities.ChapterUnlock{}).
		Where("user_id = ? AND chapter_id = ?", userID, f.chapter.ID).
		Count(&n).Error; err != nil {
		t.Fatalf("count unlocks: %v", err)
	}
	return n
}

func (f *auFixture) balance(t *testing.T, userID int64) int {
	t.Helper()
	var w entities.WalletBalance
	if err := f.m.DB.Where("user_id = ?", userID).Take(&w).Error; err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	return w.Balance
}

func (f *auFixture) attempt(t *testing.T, userID int64) entities.AutoUnlockAttempt {
	t.Helper()
	var a entities.AutoUnlockAttempt
	if err := f.m.DB.
		Where("user_id = ? AND chapter_id = ?", userID, f.chapter.ID).
		Take(&a).Error; err != nil {
		t.Fatalf("load attempt: %v", err)
	}
	return a
}

// I-AU-01 — a funded subscriber is debited and gets the unlock row.
func TestAutoUnlockJob_DebitsASubscriberAndWritesTheUnlock(t *testing.T) {
	f := newAutoUnlockFixture(t)
	reader := f.subscriber(t, 100, 50)

	report, err := f.job().Run(testContext(t), time.Now())
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if report.Processed != 1 {
		t.Fatalf("processed = %d, want 1", report.Processed)
	}
	if f.unlockCount(t, reader.ID) != 1 {
		t.Fatal("no unlock row was written")
	}
	if got := f.balance(t, reader.ID); got != 90 {
		t.Fatalf("balance = %d, want 90 after a 10-coin chapter", got)
	}
	if a := f.attempt(t, reader.ID); a.Outcome != entities.AutoUnlockUnlocked || a.LedgerID == nil {
		t.Fatalf("attempt = %+v, want an unlocked outcome carrying its ledger id", a)
	}
}

// I-AU-02 — the derived key makes a re-run a replay, not a second charge. The
// candidate predicate is the missing-unlock invariant, so the second run should
// not even see the reader.
func TestAutoUnlockJob_RunningTwiceChargesOnce(t *testing.T) {
	f := newAutoUnlockFixture(t)
	reader := f.subscriber(t, 100, 50)

	for i := range 3 {
		if _, err := f.job().Run(testContext(t), time.Now()); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}

	if f.unlockCount(t, reader.ID) != 1 {
		t.Fatal("repeated runs unlocked the same chapter more than once")
	}
	if got := f.balance(t, reader.ID); got != 90 {
		t.Fatalf("balance = %d, want a single 10-coin debit", got)
	}
	assertWalletReconciles(t, f.m, reader.ID)
}

// I-AU-03 — this is the structural test named in the design: the batch claim
// takes an advisory lock, but each debit runs in its own transaction. If a
// later refactor moves the debits inside withJobLock's transaction, the broke
// subscriber's rollback takes everyone else's unlock with it and this fails.
func TestAutoUnlockJob_OneBrokeSubscriberDoesNotRollBackTheOthers(t *testing.T) {
	f := newAutoUnlockFixture(t)

	broke := f.subscriber(t, 0, 50)
	rich := f.subscriber(t, 100, 50)
	alsoRich := f.subscriber(t, 100, 50)

	report, err := f.job().Run(testContext(t), time.Now())
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if report.Processed != 2 {
		t.Fatalf("processed = %d, want the two funded subscribers", report.Processed)
	}

	for _, u := range []*entities.User{rich, alsoRich} {
		if f.unlockCount(t, u.ID) != 1 {
			t.Fatal("a funded subscriber lost their unlock to another reader's failure")
		}
	}
	if f.unlockCount(t, broke.ID) != 0 {
		t.Fatal("a subscriber with no coins was unlocked anyway")
	}
}

// I-AU-04 — running out of coins notifies once and leaves the subscription on,
// so topping up resumes unlocking without the reader re-subscribing.
func TestAutoUnlockJob_InsufficientCoinsNotifiesAndKeepsTheSubscriptionActive(t *testing.T) {
	f := newAutoUnlockFixture(t)
	broke := f.subscriber(t, 0, 50)

	if _, err := f.job().Run(testContext(t), time.Now()); err != nil {
		t.Fatalf("run job: %v", err)
	}

	if f.notifier.count() != 1 {
		t.Fatalf("notifications = %d, want exactly 1", f.notifier.count())
	}
	if a := f.attempt(t, broke.ID); a.Outcome != entities.AutoUnlockInsufficient || a.Attempts != 1 {
		t.Fatalf("attempt = %+v, want one insufficient attempt", a)
	}

	var sub entities.AutoUnlockSubscription
	if err := f.m.DB.Where("user_id = ? AND novel_id = ?", broke.ID, f.novel.ID).
		Take(&sub).Error; err != nil {
		t.Fatalf("load subscription: %v", err)
	}
	if !sub.Active {
		t.Fatal("a failed debit paused the subscription; it must stay on so a top-up resumes it")
	}
}

// I-AU-05 — a short reader is retried with backoff, and only up to MaxAttempts.
func TestAutoUnlockJob_RetriesWithBackoffThenStopsAtMaxAttempts(t *testing.T) {
	f := newAutoUnlockFixture(t)
	broke := f.subscriber(t, 0, 50)

	job := f.job()
	job.RetryAfter = time.Hour
	job.MaxAttempts = 3

	// A second run immediately after the first is inside the backoff window, so
	// the reader is not a candidate and attempts stay at 1.
	now := time.Now()
	for range 2 {
		if _, err := job.Run(testContext(t), now); err != nil {
			t.Fatalf("run: %v", err)
		}
	}
	if a := f.attempt(t, broke.ID); a.Attempts != 1 {
		t.Fatalf("attempts = %d, want the backoff to suppress the immediate retry", a.Attempts)
	}

	// Past the backoff, they are retried — up to the cap and no further.
	for i := 1; i <= 5; i++ {
		if _, err := job.Run(testContext(t), now.Add(time.Duration(i)*2*time.Hour)); err != nil {
			t.Fatalf("run: %v", err)
		}
	}
	if a := f.attempt(t, broke.ID); a.Attempts != 3 {
		t.Fatalf("attempts = %d, want to stop at MaxAttempts=3", a.Attempts)
	}
}

// I-AU-06 — a chapter priced above the reader's cap is skipped, not charged,
// and is not retried.
func TestAutoUnlockJob_SkipsChaptersOverTheReadersCap(t *testing.T) {
	f := newAutoUnlockFixture(t)
	reader := f.subscriber(t, 500, 5) // cap 5, chapter costs 10

	report, err := f.job().Run(testContext(t), time.Now())
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if report.Processed != 0 {
		t.Fatalf("processed = %d, want 0", report.Processed)
	}
	if f.unlockCount(t, reader.ID) != 0 {
		t.Fatal("a chapter over the cap was unlocked")
	}
	if got := f.balance(t, reader.ID); got != 500 {
		t.Fatalf("balance = %d, want it untouched", got)
	}
	if a := f.attempt(t, reader.ID); a.Outcome != entities.AutoUnlockOverCap {
		t.Fatalf("outcome = %q, want over_cap", a.Outcome)
	}
	if f.notifier.count() != 0 {
		t.Fatal("an over-cap skip is not a failure and must not notify")
	}
}

// I-AU-07 — a reader who bought the chapter manually is never charged again,
// because the candidate predicate is "no unlock row exists".
func TestAutoUnlockJob_NeverChargesForAManuallyUnlockedChapter(t *testing.T) {
	f := newAutoUnlockFixture(t)
	reader := f.subscriber(t, 100, 50)

	if _, err := f.wallet.UnlockChapter(testContext(t), reader.ID, f.chapter.ID, "manual"); err != nil {
		t.Fatalf("manual unlock: %v", err)
	}

	report, err := f.job().Run(testContext(t), time.Now())
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if report.Processed != 0 {
		t.Fatalf("processed = %d, want the manual buyer skipped entirely", report.Processed)
	}
	if got := f.balance(t, reader.ID); got != 90 {
		t.Fatalf("balance = %d, want only the manual purchase", got)
	}
	assertWalletReconciles(t, f.m, reader.ID)
}

// I-AU-08 — inactive subscriptions and chapters published before the reader
// opted in are both out of scope: opting in must not backfill a whole history.
func TestAutoUnlockJob_IgnoresInactiveSubscriptionsAndPreSubscriptionChapters(t *testing.T) {
	f := newAutoUnlockFixture(t)

	inactive := f.subscriber(t, 100, 50)
	if err := f.m.DB.Model(&entities.AutoUnlockSubscription{}).
		Where("user_id = ?", inactive.ID).
		Update("active", false).Error; err != nil {
		t.Fatalf("deactivate: %v", err)
	}

	// A reader who subscribed after the chapter went out.
	late := f.m.ANewUser().Please()
	f.m.ANewWalletBalance().With(func(w *entities.WalletBalance) {
		w.UserID = late.ID
		w.Balance = 100
	}).Please()
	f.m.ANewAutoUnlockSubscription().With(func(s *entities.AutoUnlockSubscription) {
		s.UserID = late.ID
		s.NovelID = f.novel.ID
		s.Active = true
		s.MaxCoinsPerChapter = 50
		s.CreatedAt = time.Now()
	}).Please()

	report, err := f.job().Run(testContext(t), time.Now())
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if report.Processed != 0 {
		t.Fatalf("processed = %d, want neither reader charged", report.Processed)
	}
	if f.unlockCount(t, inactive.ID) != 0 {
		t.Fatal("an inactive subscription was still charged")
	}
	if f.unlockCount(t, late.ID) != 0 {
		t.Fatal("subscribing backfilled a chapter published beforehand")
	}
}

// A free chapter is not a sale, so auto-unlock has nothing to do with it.
func TestAutoUnlockJob_IgnoresFreeChapters(t *testing.T) {
	f := newAutoUnlockFixture(t)
	reader := f.subscriber(t, 100, 50)

	if err := f.m.DB.Model(&entities.Chapter{}).
		Where("id = ?", f.chapter.ID).
		Update("price_coins", 0).Error; err != nil {
		t.Fatalf("make it free: %v", err)
	}

	report, err := f.job().Run(testContext(t), time.Now())
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if report.Processed != 0 || f.unlockCount(t, reader.ID) != 0 {
		t.Fatalf("processed = %d and %d unlocks, want a free chapter ignored",
			report.Processed, f.unlockCount(t, reader.ID))
	}
}

// assertWalletReconciles is the release gate: every coin in a balance must be
// explained by a ledger row.
func assertWalletReconciles(t *testing.T, m *makeme.MakeMe, userID int64) {
	t.Helper()

	var sums struct {
		Delta      int
		BonusDelta int
	}
	err := m.DB.Model(&entities.CoinLedgerEntry{}).
		Select("COALESCE(SUM(delta),0) AS delta, COALESCE(SUM(bonus_delta),0) AS bonus_delta").
		Where("user_id = ?", userID).
		Take(&sums).Error
	if err != nil {
		t.Fatalf("sum ledger: %v", err)
	}

	var wallet entities.WalletBalance
	if err := m.DB.Where("user_id = ?", userID).Take(&wallet).Error; err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if sums.Delta != wallet.Balance || sums.BonusDelta != wallet.BonusBalance {
		t.Fatalf("ledger %d/%d does not reconcile with wallet %d/%d",
			sums.Delta, sums.BonusDelta, wallet.Balance, wallet.BonusBalance)
	}
}
