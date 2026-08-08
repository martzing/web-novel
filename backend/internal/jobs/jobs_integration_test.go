package jobs_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/jobs"
	walletrepo "github.com/mokchan/webnovel-backend/internal/repository/wallet"
	walletsvc "github.com/mokchan/webnovel-backend/internal/service/wallet"
	"github.com/mokchan/webnovel-backend/test/makeme"
)

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// I-COIN-09 — the nightly job writes a bonus_expire ledger row and zeroes the
// bonus balance.
func TestBonusExpiryJob_WritesBonusExpireLedgerAndZeroesBalance(t *testing.T) {
	m := makeme.New(t)
	m.Reset()

	user := m.ANewUser().Please()
	yesterday := time.Now().Add(-24 * time.Hour)
	m.ANewWalletBalance().With(func(w *entities.WalletBalance) {
		w.UserID = user.ID
		w.Balance = 120
		w.BonusBalance = 50
		w.BonusExpiresAt = &yesterday
	}).Please()

	// A second wallet whose bonus is still live must be left alone.
	other := m.ANewUser().Please()
	tomorrow := time.Now().Add(24 * time.Hour)
	m.ANewWalletBalance().With(func(w *entities.WalletBalance) {
		w.UserID = other.ID
		w.Balance = 10
		w.BonusBalance = 30
		w.BonusExpiresAt = &tomorrow
	}).Please()

	wallet := walletsvc.New(walletrepo.New(m.DB), 720*time.Hour, 30, time.Now)
	job := &jobs.BonusExpiryJob{DB: m.DB, Wallet: wallet}

	now := time.Now()
	report, err := job.Run(testContext(t), now)
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if report.Processed != 1 {
		t.Fatalf("processed = %d, want exactly the one lapsed wallet", report.Processed)
	}

	var expired entities.WalletBalance
	if err := m.DB.Where("user_id = ?", user.ID).Take(&expired).Error; err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if expired.BonusBalance != 0 {
		t.Fatalf("bonus_balance = %d, want 0", expired.BonusBalance)
	}
	if expired.Balance != 120 {
		t.Fatalf("balance = %d, want the paid balance untouched at 120", expired.Balance)
	}
	if expired.BonusExpiresAt != nil {
		t.Fatal("a zeroed bonus must clear its expiry timestamp")
	}

	var entry entities.CoinLedgerEntry
	err = m.DB.Where("user_id = ? AND kind = ?", user.ID, entities.LedgerBonusExpire).
		Take(&entry).Error
	if err != nil {
		t.Fatalf("load bonus_expire ledger row: %v", err)
	}
	if entry.BonusDelta != -50 || entry.Delta != 0 {
		t.Fatalf("entry = delta %d bonus %d, want 0/-50", entry.Delta, entry.BonusDelta)
	}
	if entry.BonusBalanceAfter != 0 || entry.BalanceAfter != 120 {
		t.Fatalf("entry after-state = %d/%d, want 120/0", entry.BalanceAfter, entry.BonusBalanceAfter)
	}

	// The live bonus is untouched.
	var untouched entities.WalletBalance
	if err := m.DB.Where("user_id = ?", other.ID).Take(&untouched).Error; err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if untouched.BonusBalance != 30 {
		t.Fatalf("a live bonus was expired: %d", untouched.BonusBalance)
	}

	// Re-running the same day is a no-op: the idempotency key is date-stamped.
	if _, err := job.Run(testContext(t), now); err != nil {
		t.Fatalf("second run: %v", err)
	}
	var expireRows int64
	err = m.DB.Model(&entities.CoinLedgerEntry{}).
		Where("user_id = ? AND kind = ?", user.ID, entities.LedgerBonusExpire).
		Count(&expireRows).Error
	if err != nil {
		t.Fatalf("count ledger rows: %v", err)
	}
	if expireRows != 1 {
		t.Fatalf("bonus_expire rows = %d, want exactly 1 after a re-run", expireRows)
	}
}

// I-GLO-02 / I-GLO-03 — the worker re-renders stale bodies, and readers keep
// getting valid HTML throughout.
func TestRerenderJob_UpdatesBodyHTMLAndLiftsGlossaryRev(t *testing.T) {
	m := makeme.New(t)
	m.Reset()

	novel := m.ANewNovel().Please()
	group := m.ANewGlossaryGroup().With(func(g *entities.GlossaryGroup) {
		g.NovelID = novel.ID
	}).Please()
	entry := m.ANewGlossaryEntry().With(func(e *entities.GlossaryEntry) {
		e.GroupID = group.ID
		e.TermKey = "qi"
		e.TitleTH = "ชี่"
	}).Please()

	chapter := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = novel.ID
		c.ChapterNo = 1
		c.Status = entities.ChapterPublished
	}).Please()

	// A body rendered against the old glossary.
	m.ANewChapterBody().With(func(b *entities.ChapterBody) {
		b.ChapterID = chapter.ID
		b.BodySource = "<p>{{qi}} ไหลเวียน</p>"
		b.BodyHTML = `<p><span data-k="qi">ชี่</span> ไหลเวียน</p>`
		b.GlossaryRev = 0
	}).Please()

	// Creating the entry already bumped glossary_rev via the trigger; align the
	// body so the starting state is genuinely fresh.
	var novelRow entities.Novel
	if err := m.DB.Where("id = ?", novel.ID).Take(&novelRow).Error; err != nil {
		t.Fatalf("load novel: %v", err)
	}
	err := m.DB.Model(&entities.ChapterBody{}).
		Where("chapter_id = ?", chapter.ID).
		Update("glossary_rev", novelRow.GlossaryRev).Error
	if err != nil {
		t.Fatalf("align glossary_rev: %v", err)
	}

	job := &jobs.GlossaryRerenderJob{DB: m.DB}

	t.Run("nothing to do when the body is current", func(t *testing.T) {
		report, err := job.Run(testContext(t), time.Now())
		if err != nil {
			t.Fatalf("run job: %v", err)
		}
		if report.Processed != 0 {
			t.Fatalf("processed = %d, want 0", report.Processed)
		}
	})

	// Editing the term bumps novels.glossary_rev through the database trigger.
	err = m.DB.Model(&entities.GlossaryEntry{}).
		Where("id = ?", entry.ID).
		Update("title_th", "ชี่ (ปรับคำแปล)").Error
	if err != nil {
		t.Fatalf("edit glossary entry: %v", err)
	}

	// I-GLO-03: before the worker runs, the stale body is still valid HTML the
	// reader can be served — the read path never gates on glossary_rev.
	t.Run("a stale body still serves the old HTML", func(t *testing.T) {
		var body entities.ChapterBody
		if err := m.DB.Where("chapter_id = ?", chapter.ID).Take(&body).Error; err != nil {
			t.Fatalf("load body: %v", err)
		}
		if !strings.Contains(body.BodyHTML, `<span data-k="qi">ชี่</span>`) {
			t.Fatalf("stale body should still hold the old rendering:\n%s", body.BodyHTML)
		}
	})

	t.Run("the worker re-renders and lifts the revision", func(t *testing.T) {
		report, err := job.Run(testContext(t), time.Now())
		if err != nil {
			t.Fatalf("run job: %v", err)
		}
		if report.Processed != 1 {
			t.Fatalf("processed = %d, want 1", report.Processed)
		}

		var body entities.ChapterBody
		if err := m.DB.Where("chapter_id = ?", chapter.ID).Take(&body).Error; err != nil {
			t.Fatalf("load body: %v", err)
		}
		if !strings.Contains(body.BodyHTML, "ชี่ (ปรับคำแปล)") {
			t.Fatalf("body was not re-rendered with the new title:\n%s", body.BodyHTML)
		}

		var novelAfter entities.Novel
		if err := m.DB.Where("id = ?", novel.ID).Take(&novelAfter).Error; err != nil {
			t.Fatalf("load novel: %v", err)
		}
		if body.GlossaryRev != novelAfter.GlossaryRev {
			t.Fatalf("body glossary_rev = %d, want the novel's %d",
				body.GlossaryRev, novelAfter.GlossaryRev)
		}
	})

	t.Run("a second run has nothing left to do", func(t *testing.T) {
		report, err := job.Run(testContext(t), time.Now())
		if err != nil {
			t.Fatalf("run job: %v", err)
		}
		if report.Processed != 0 {
			t.Fatalf("processed = %d, want 0", report.Processed)
		}
	})
}

// W-06 — the scheduler flips a chapter live once its time passes.
func TestPublishScheduledJob_PublishesDueChapters(t *testing.T) {
	m := makeme.New(t)
	m.Reset()

	novel := m.ANewNovel().Please()
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	due := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = novel.ID
		c.ChapterNo = 1
		c.Status = entities.ChapterScheduled
		c.ScheduledAt = &past
		c.PublishedAt = nil
	}).Please()
	notYet := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = novel.ID
		c.ChapterNo = 2
		c.Status = entities.ChapterScheduled
		c.ScheduledAt = &future
		c.PublishedAt = nil
	}).Please()

	job := &jobs.PublishScheduledJob{DB: m.DB}
	report, err := job.Run(testContext(t), time.Now())
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if report.Processed != 1 {
		t.Fatalf("processed = %d, want 1", report.Processed)
	}

	var published entities.Chapter
	if err := m.DB.Where("id = ?", due.ID).Take(&published).Error; err != nil {
		t.Fatalf("load chapter: %v", err)
	}
	if published.Status != entities.ChapterPublished || published.PublishedAt == nil {
		t.Fatalf("chapter = %+v, want it published and timestamped", published)
	}

	var waiting entities.Chapter
	if err := m.DB.Where("id = ?", notYet.ID).Take(&waiting).Error; err != nil {
		t.Fatalf("load chapter: %v", err)
	}
	if waiting.Status != entities.ChapterScheduled {
		t.Fatalf("a future chapter was published early: %+v", waiting)
	}

	var novelAfter entities.Novel
	if err := m.DB.Where("id = ?", novel.ID).Take(&novelAfter).Error; err != nil {
		t.Fatalf("load novel: %v", err)
	}
	if novelAfter.ChaptersCount != 1 {
		t.Fatalf("chapters_count = %d, want 1", novelAfter.ChaptersCount)
	}
}

// The PRD's release gate: the ledger must reconcile with every cached balance.
func TestReconcileJob_ReportsZeroDiscrepancyOnAConsistentLedger(t *testing.T) {
	m := makeme.New(t)
	m.Reset()

	user := m.ANewUser().Please()
	wallet := walletsvc.New(walletrepo.New(m.DB), 720*time.Hour, 30, time.Now)

	// Drive a few movements through the real write path.
	ctx := testContext(t)
	if _, err := wallet.Adjust(ctx, user.ID, 100, 20, "seed", user.ID, "k1"); err != nil {
		t.Fatalf("adjust: %v", err)
	}
	if _, err := wallet.Adjust(ctx, user.ID, -30, 0, "spend", user.ID, "k2"); err != nil {
		t.Fatalf("adjust: %v", err)
	}

	job := &jobs.ReconcileJob{DB: m.DB}
	report, err := job.Run(ctx, time.Now())
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if report.Processed != 0 {
		t.Fatalf("mismatched wallets = %d, want 0", report.Processed)
	}

	// Corrupt the cached balance and confirm the job notices.
	err = m.DB.Model(&entities.WalletBalance{}).
		Where("user_id = ?", user.ID).
		Update("balance", 999).Error
	if err != nil {
		t.Fatalf("corrupt balance: %v", err)
	}

	report, err = job.Run(ctx, time.Now())
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if report.Processed != 1 {
		t.Fatalf("mismatched wallets = %d, want the corrupted one detected", report.Processed)
	}
}

func TestSessionSweepJob_DeletesOnlyLongExpiredTokens(t *testing.T) {
	m := makeme.New(t)
	m.Reset()

	user := m.ANewUser().Please()
	insert := func(expiresAt time.Time, hash string) {
		err := m.DB.Create(&entities.RefreshToken{
			UserID:    user.ID,
			FamilyID:  "00000000-0000-4000-8000-000000000000",
			TokenHash: []byte(hash),
			ExpiresAt: expiresAt,
		}).Error
		if err != nil {
			t.Fatalf("insert token: %v", err)
		}
	}

	insert(time.Now().Add(-30*24*time.Hour), "ancient")
	insert(time.Now().Add(-time.Hour), "recently-expired")
	insert(time.Now().Add(24*time.Hour), "live")

	job := &jobs.SessionSweepJob{DB: m.DB, Retain: 7 * 24 * time.Hour}
	report, err := job.Run(testContext(t), time.Now())
	if err != nil {
		t.Fatalf("run job: %v", err)
	}
	if report.Processed != 1 {
		t.Fatalf("deleted %d tokens, want only the long-expired one", report.Processed)
	}

	var remaining int64
	if err := m.DB.Model(&entities.RefreshToken{}).Count(&remaining).Error; err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if remaining != 2 {
		t.Fatalf("remaining tokens = %d, want 2", remaining)
	}
}

// Read events must land in a real monthly partition, not the catch-all.
func TestPartitionJob_CreatesMonthlyPartitions(t *testing.T) {
	m := makeme.New(t)
	m.Reset()

	job := &jobs.PartitionJob{DB: m.DB}
	now := time.Now()
	if _, err := job.Run(testContext(t), now); err != nil {
		t.Fatalf("run job: %v", err)
	}

	for _, offset := range []int{0, 1} {
		name, _, _ := jobs.PartitionSpecFor(now, offset)
		var exists bool
		err := m.DB.Raw("SELECT to_regclass(?) IS NOT NULL", name).Scan(&exists).Error
		if err != nil {
			t.Fatalf("check partition: %v", err)
		}
		if !exists {
			t.Fatalf("partition %s was not created", name)
		}
	}

	// Running twice must not fail on the already-created partitions.
	if _, err := job.Run(testContext(t), now); err != nil {
		t.Fatalf("second run: %v", err)
	}
}

func TestPartitionSpecFor(t *testing.T) {
	now := time.Date(2026, 12, 15, 10, 0, 0, 0, time.UTC)

	name, from, to := jobs.PartitionSpecFor(now, 0)
	if name != "chapter_read_events_2026_12" {
		t.Fatalf("name = %q", name)
	}
	if !from.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("from = %v", from)
	}
	if !to.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("to = %v", to)
	}

	// The offset must roll the year over correctly.
	name, from, _ = jobs.PartitionSpecFor(now, 1)
	if name != "chapter_read_events_2027_01" {
		t.Fatalf("name = %q", name)
	}
	if !from.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("from = %v", from)
	}
}
