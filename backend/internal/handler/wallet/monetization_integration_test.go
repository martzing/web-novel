package wallet_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type arcBundleResponse struct {
	ArcID           string `json:"arc_id"`
	ChapterCount    int    `json:"chapter_count"`
	Gross           int    `json:"gross"`
	DiscountPercent int    `json:"discount_percent"`
	Discount        int    `json:"discount"`
	Total           int    `json:"total"`
	Chapters        []struct {
		ChapterID string `json:"chapter_id"`
		ListPrice int    `json:"list_price"`
		Coins     int    `json:"coins"`
	} `json:"chapters"`
}

// arcFixture is a novel with one arc of five 10-coin chapters, arc sales and
// tips enabled, and a funded reader.
type arcFixture struct {
	env      *apitest.Env
	reader   *entities.User
	token    string
	writer   *entities.User
	novel    *entities.Novel
	arc      *entities.Arc
	chapters []*entities.Chapter
}

func newArcFixture(t *testing.T, balance, bonus int) *arcFixture {
	t.Helper()
	env := apitest.New(t)
	m := env.MakeMe

	writer := env.AUser(entities.RoleTranslator)
	reader := env.AUser()

	novel := m.ANewNovel().With(func(n *entities.Novel) {
		n.PrimaryTranslatorID = &writer.ID
		n.SellByArc = true
		n.TipsEnabled = true
	}).Please()

	arc := m.ANewArc().With(func(a *entities.Arc) {
		a.NovelID = novel.ID
		a.ArcNo = 1
		a.FromChapterNo = 1
		a.ToChapterNo = 10
	}).Please()

	chapters := make([]*entities.Chapter, 0, 5)
	for i := 1; i <= 5; i++ {
		c := m.ANewChapter().With(func(c *entities.Chapter) {
			c.NovelID = novel.ID
			c.ChapterNo = i
			c.PriceCoins = 10
			c.Status = entities.ChapterPublished
			c.TranslatorID = &writer.ID
			// arc_id deliberately left NULL: membership must resolve by
			// chapter-number range, not by this column.
			c.ArcID = nil
		}).Please()
		chapters = append(chapters, c)
	}

	if balance > 0 || bonus > 0 {
		var expires *time.Time
		if bonus > 0 {
			future := time.Now().Add(720 * time.Hour)
			expires = &future
		}
		m.ANewWalletBalance().With(func(w *entities.WalletBalance) {
			w.UserID = reader.ID
			w.Balance = balance
			w.BonusBalance = bonus
			w.BonusExpiresAt = expires
		}).Please()

		// The opening balance gets a matching ledger row. Seeding the balance
		// alone would break sum(delta) == balance before a single test ran,
		// and that invariant is the release gate these tests exist to guard.
		m.ANewCoinLedgerEntry().With(func(e *entities.CoinLedgerEntry) {
			e.UserID = reader.ID
			e.Kind = entities.LedgerTopup
			e.Delta = balance
			e.BonusDelta = bonus
			e.BalanceAfter = balance
			e.BonusBalanceAfter = bonus
		}).Please()
	}

	return &arcFixture{
		env: env, reader: reader, token: env.TokenFor(reader),
		writer: writer, novel: novel, arc: arc, chapters: chapters,
	}
}

func (f *arcFixture) quote(t *testing.T) arcBundleResponse {
	t.Helper()
	rec := f.env.GETAuth(fmt.Sprintf("/api/v1/arcs/%d/bundle", f.arc.ID), f.token)
	apitest.AssertStatus(t, rec, http.StatusOK)
	return apitest.DecodeJSON[arcBundleResponse](t, rec)
}

func (f *arcFixture) buyArc(t *testing.T, key string) *httptest.ResponseRecorder {
	t.Helper()
	return f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/arcs/%d/unlock", f.arc.ID),
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": key},
	})
}

func TestArcBundle_QuoteExcludesFreeAndAlreadyOwnedChapters(t *testing.T) {
	f := newArcFixture(t, 500, 0)
	m := f.env.MakeMe

	// A free chapter inside the range is never part of the bundle.
	m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = f.novel.ID
		c.ChapterNo = 6
		c.PriceCoins = 0
		c.Status = entities.ChapterPublished
	}).Please()

	quote := f.quote(t)
	if quote.ChapterCount != 5 || quote.Gross != 50 {
		t.Fatalf("quote = %+v, want the 5 paid chapters at 50 gross", quote)
	}

	// Buying one chapter shrinks the bundle.
	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.chapters[0].ID),
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": "single"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	after := f.quote(t)
	if after.ChapterCount != 4 || after.Gross != 40 {
		t.Fatalf("quote = %+v, want the owned chapter excluded", after)
	}
}

// I-COIN-11 — one ledger row, N unlock rows, priced at 85%.
func TestArcBundle_ChargesFifteenPercentOffAndWritesOneLedgerRowWithNUnlocks(t *testing.T) {
	f := newArcFixture(t, 500, 0)

	quote := f.quote(t)
	if quote.DiscountPercent != 15 || quote.Gross != 50 || quote.Discount != 7 || quote.Total != 43 {
		t.Fatalf("quote = %+v, want 50 gross, 7 discount, 43 total", quote)
	}

	rec := f.buyArc(t, "bundle-1")
	apitest.AssertStatus(t, rec, http.StatusOK)
	receipt := apitest.DecodeJSON[receiptResponse](t, rec)
	if receipt.CoinsSpent != 43 {
		t.Fatalf("coins spent = %d, want 43", receipt.CoinsSpent)
	}

	var ledgerRows int64
	if err := f.env.MakeMe.DB.Model(&entities.CoinLedgerEntry{}).
		Where("user_id = ? AND kind = ?", f.reader.ID, entities.LedgerSpendUnlock).
		Count(&ledgerRows).Error; err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if ledgerRows != 1 {
		t.Fatalf("ledger rows = %d, want exactly 1 for the whole bundle", ledgerRows)
	}

	var unlocks []entities.ChapterUnlock
	if err := f.env.MakeMe.DB.Where("user_id = ?", f.reader.ID).Find(&unlocks).Error; err != nil {
		t.Fatalf("load unlocks: %v", err)
	}
	if len(unlocks) != 5 {
		t.Fatalf("unlock rows = %d, want 5", len(unlocks))
	}

	// The child rows and the debit must describe the same money.
	spent := 0
	for _, u := range unlocks {
		spent += int(u.CoinsSpent)
		if fmt.Sprint(u.LedgerID) != receipt.LedgerID {
			t.Fatalf("unlock references ledger %d, want the bundle's %s", u.LedgerID, receipt.LedgerID)
		}
	}
	if spent != 43 {
		t.Fatalf("sum(coins_spent) = %d, want the 43 actually debited", spent)
	}
}

func TestArcBundle_WritesOneEarningPerChapterSharingOneLedgerID(t *testing.T) {
	f := newArcFixture(t, 500, 0)
	f.buyArc(t, "bundle-earnings")

	var earnings []entities.WriterEarning
	if err := f.env.MakeMe.DB.Where("writer_id = ?", f.writer.ID).Find(&earnings).Error; err != nil {
		t.Fatalf("load earnings: %v", err)
	}
	if len(earnings) != 5 {
		t.Fatalf("earning rows = %d, want one per chapter", len(earnings))
	}

	ledgerID := earnings[0].UnlockLedgerID
	gross := 0
	for _, e := range earnings {
		if e.UnlockLedgerID != ledgerID {
			t.Fatal("all earnings in a bundle must share the one ledger row")
		}
		if e.Kind != entities.EarningUnlock {
			t.Fatalf("earning kind = %q, want unlock", e.Kind)
		}
		gross += e.GrossCoins
	}
	if gross != 43 {
		t.Fatalf("sum(gross_coins) = %d, want the 43 debited", gross)
	}
}

// The bundle must include chapters whose arc_id is NULL but whose number falls
// inside the arc's range, or the reader is undercharged and misses chapters.
func TestArcBundle_IncludesChaptersWhoseArcIDIsNull(t *testing.T) {
	f := newArcFixture(t, 500, 0)

	for _, c := range f.chapters {
		var row entities.Chapter
		if err := f.env.MakeMe.DB.Where("id = ?", c.ID).Take(&row).Error; err != nil {
			t.Fatalf("load chapter: %v", err)
		}
		if row.ArcID != nil {
			t.Fatal("test setup is wrong: arc_id should be NULL")
		}
	}

	if quote := f.quote(t); quote.ChapterCount != 5 {
		t.Fatalf("chapter count = %d, want all 5 resolved by number range", quote.ChapterCount)
	}
}

func TestArcBundle_OwningEveryChapterReturns409(t *testing.T) {
	f := newArcFixture(t, 500, 0)
	f.buyArc(t, "first")

	rec := f.buyArc(t, "second")
	apitest.AssertErrorCode(t, rec, http.StatusConflict, "ARC_ALREADY_OWNED")
}

func TestArcBundle_DisabledToggleReturns400(t *testing.T) {
	f := newArcFixture(t, 500, 0)

	err := f.env.MakeMe.DB.Model(&entities.Novel{}).
		Where("id = ?", f.novel.ID).Update("sell_by_arc", false).Error
	if err != nil {
		t.Fatalf("disable arc sales: %v", err)
	}

	rec := f.env.GETAuth(fmt.Sprintf("/api/v1/arcs/%d/bundle", f.arc.ID), f.token)
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "ARC_NOT_FOR_SALE")
}

func TestArcBundle_InsufficientCoinsReturns402AndWritesNothing(t *testing.T) {
	f := newArcFixture(t, 10, 0)

	rec := f.buyArc(t, "poor")
	apitest.AssertErrorCode(t, rec, http.StatusPaymentRequired, "INSUFFICIENT_COINS")

	var rows int64
	if err := f.env.MakeMe.DB.Model(&entities.ChapterUnlock{}).
		Where("user_id = ?", f.reader.ID).Count(&rows).Error; err != nil {
		t.Fatalf("count unlocks: %v", err)
	}
	if rows != 0 {
		t.Fatalf("unlock rows = %d, want 0", rows)
	}
}

// I-COIN-12 — two concurrent bundle buys yield exactly one debit.
func TestArcBundle_ConcurrentDoubleBuyYieldsOneDebit(t *testing.T) {
	f := newArcFixture(t, 500, 0)

	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i := range 2 {
		wg.Go(func() {
			codes[i] = f.buyArc(t, fmt.Sprintf("race-%d", i)).Code
		})
	}
	wg.Wait()

	var ok, conflict int
	for _, code := range codes {
		switch code {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflict++
		default:
			t.Fatalf("unexpected status %d among %v", code, codes)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("statuses = %v, want one 200 and one 409", codes)
	}

	var ledgerRows int64
	f.env.MakeMe.DB.Model(&entities.CoinLedgerEntry{}).
		Where("user_id = ? AND kind = ?", f.reader.ID, entities.LedgerSpendUnlock).
		Count(&ledgerRows)
	if ledgerRows != 1 {
		t.Fatalf("ledger rows = %d, want exactly 1", ledgerRows)
	}
}

// I-COIN-13 — a bundle racing a single-chapter unlock of a member chapter must
// still debit each chapter exactly once.
func TestArcBundle_RacingASingleChapterUnlockYieldsOneDebitPerChapter(t *testing.T) {
	f := newArcFixture(t, 500, 0)

	var wg sync.WaitGroup
	wg.Go(func() { f.buyArc(t, "bundle-race") })
	wg.Go(func() {
		f.env.Do(apitest.Request{
			Method:  http.MethodPost,
			Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.chapters[2].ID),
			Token:   f.token,
			Headers: map[string]string{"Idempotency-Key": "single-race"},
		})
	})
	wg.Wait()

	// No chapter may be unlocked twice; the composite PK guarantees it, and
	// this asserts the debits agree with the rows.
	type row struct {
		ChapterID int64
		N         int
	}
	var dupes []row
	err := f.env.MakeMe.DB.Model(&entities.ChapterUnlock{}).
		Select("chapter_id, COUNT(*) AS n").
		Where("user_id = ?", f.reader.ID).
		Group("chapter_id").
		Having("COUNT(*) > 1").
		Scan(&dupes).Error
	if err != nil {
		t.Fatalf("check duplicates: %v", err)
	}
	if len(dupes) != 0 {
		t.Fatalf("chapters unlocked more than once: %+v", dupes)
	}

	assertLedgerReconciles(t, f.env, f.reader.ID)
}

// A client that reuses one key across two different operations gets both, not a
// replay of the first: the service namespaces the stored key per operation, so
// "shared" becomes unlock:shared and arc_unlock:shared. Without that, buying an
// arc after unlocking a chapter with the same key would return the chapter's
// receipt and silently skip the purchase.
func TestArcBundle_SameKeyAsAChapterUnlockIsNotAReplay(t *testing.T) {
	f := newArcFixture(t, 500, 0)

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.chapters[0].ID),
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": "shared"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)
	single := apitest.DecodeJSON[receiptResponse](t, rec)

	bundle := apitest.DecodeJSON[receiptResponse](t, f.buyArc(t, "shared"))
	if bundle.Replayed {
		t.Fatal("the arc purchase replayed the chapter unlock's receipt")
	}
	if bundle.LedgerID == single.LedgerID {
		t.Fatal("both operations wrote to the same ledger row")
	}

	// Four chapters were left, so the bundle is 40 gross less a 6-coin discount.
	if bundle.CoinsSpent != 34 {
		t.Fatalf("coins spent = %d, want 34 for the four remaining chapters", bundle.CoinsSpent)
	}
	assertLedgerReconciles(t, f.env, f.reader.ID)
}

func TestArcBundle_SameIdempotencyKeyReplaysTheReceipt(t *testing.T) {
	f := newArcFixture(t, 500, 0)

	first := apitest.DecodeJSON[receiptResponse](t, f.buyArc(t, "replay"))
	second := apitest.DecodeJSON[receiptResponse](t, f.buyArc(t, "replay"))

	if first.LedgerID != second.LedgerID || !second.Replayed {
		t.Fatalf("expected a replay, got %+v then %+v", first, second)
	}
}

func TestArcBundle_LedgerReconciles(t *testing.T) {
	f := newArcFixture(t, 500, 0)
	f.buyArc(t, "reconcile")
	assertLedgerReconciles(t, f.env, f.reader.ID)
}

func assertLedgerReconciles(t *testing.T, env *apitest.Env, userID int64) {
	t.Helper()

	var sums struct {
		Delta      int
		BonusDelta int
	}
	err := env.MakeMe.DB.Model(&entities.CoinLedgerEntry{}).
		Select("COALESCE(SUM(delta),0) AS delta, COALESCE(SUM(bonus_delta),0) AS bonus_delta").
		Where("user_id = ?", userID).
		Take(&sums).Error
	if err != nil {
		t.Fatalf("sum ledger: %v", err)
	}

	var wallet entities.WalletBalance
	if err := env.MakeMe.DB.Where("user_id = ?", userID).Take(&wallet).Error; err != nil {
		t.Fatalf("load wallet: %v", err)
	}
	if sums.Delta != wallet.Balance || sums.BonusDelta != wallet.BonusBalance {
		t.Fatalf("ledger %d/%d does not reconcile with wallet %d/%d",
			sums.Delta, sums.BonusDelta, wallet.Balance, wallet.BonusBalance)
	}
}
