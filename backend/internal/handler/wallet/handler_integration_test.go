package wallet_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type balanceResponse struct {
	Balance        int    `json:"balance"`
	BonusBalance   int    `json:"bonus_balance"`
	BonusExpiresAt string `json:"bonus_expires_at"`
	Total          int    `json:"total"`
}

type receiptResponse struct {
	LedgerID          string `json:"ledger_id"`
	CoinsSpent        int    `json:"coins_spent"`
	Delta             int    `json:"delta"`
	BonusDelta        int    `json:"bonus_delta"`
	BalanceAfter      int    `json:"balance_after"`
	BonusBalanceAfter int    `json:"bonus_balance_after"`
	Replayed          bool   `json:"replayed"`
}

type purchaseResponse struct {
	PurchaseID   string `json:"purchase_id"`
	Status       string `json:"status"`
	AmountSatang int    `json:"amount_satang"`
}

type ledgerEntryResponse struct {
	ID           string `json:"id"`
	Delta        int    `json:"delta"`
	BonusDelta   int    `json:"bonus_delta"`
	Kind         string `json:"kind"`
	RefType      string `json:"ref_type"`
	Reason       string `json:"reason"`
	BalanceAfter int    `json:"balance_after"`
}

// fixture builds a reader with a funded wallet and a paid chapter to unlock.
type fixture struct {
	env       *apitest.Env
	reader    *entities.User
	token     string
	novel     *entities.Novel
	chapter   *entities.Chapter
	freeChap  *entities.Chapter
	writer    *entities.User
	packID    int64
	chapterID int64
}

func newFixture(t *testing.T, balance, bonus int, bonusExpiresAt *time.Time) *fixture {
	t.Helper()
	env := apitest.New(t)
	m := env.MakeMe

	writer := env.AUser(entities.RoleTranslator)
	reader := env.AUser()

	novel := m.ANewNovel().With(func(n *entities.Novel) {
		n.PrimaryTranslatorID = &writer.ID
	}).Please()

	paid := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = novel.ID
		c.ChapterNo = 87
		c.PriceCoins = 5
		c.Status = entities.ChapterPublished
		c.TranslatorID = &writer.ID
	}).Please()
	m.ANewChapterBody().With(func(b *entities.ChapterBody) {
		b.ChapterID = paid.ID
		b.BodyHTML = "<p>เนื้อหาบทที่ปลดล็อกแล้ว</p>"
	}).Please()

	free := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = novel.ID
		c.ChapterNo = 1
		c.PriceCoins = 0
		c.Status = entities.ChapterPublished
	}).Please()

	if balance > 0 || bonus > 0 || bonusExpiresAt != nil {
		m.ANewWalletBalance().With(func(w *entities.WalletBalance) {
			w.UserID = reader.ID
			w.Balance = balance
			w.BonusBalance = bonus
			w.BonusExpiresAt = bonusExpiresAt
		}).Please()
	}

	pack := m.ANewCoinPack().With(func(p *entities.CoinPack) {
		p.Coins = 240
		p.BonusCoins = 20
		p.PriceSatang = 14900
	}).Please()

	return &fixture{
		env:       env,
		reader:    reader,
		token:     env.TokenFor(reader),
		novel:     novel,
		chapter:   paid,
		freeChap:  free,
		writer:    writer,
		packID:    pack.ID,
		chapterID: paid.ID,
	}
}

func (f *fixture) unlock(t *testing.T, key string) *receiptResponse {
	t.Helper()
	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.chapterID),
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": key},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)
	body := apitest.DecodeJSON[receiptResponse](t, rec)
	return &body
}

func (f *fixture) balance(t *testing.T) balanceResponse {
	t.Helper()
	rec := f.env.GETAuth("/api/v1/me/wallet", f.token)
	apitest.AssertStatus(t, rec, http.StatusOK)
	return apitest.DecodeJSON[balanceResponse](t, rec)
}

// countLedger returns how many ledger rows of a kind the reader has.
func (f *fixture) countLedger(t *testing.T, kind string) int64 {
	t.Helper()
	var count int64
	err := f.env.MakeMe.DB.Model(&entities.CoinLedgerEntry{}).
		Where("user_id = ? AND kind = ?", f.reader.ID, kind).
		Count(&count).Error
	if err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	return count
}

func TestGetWallet_EmptyForANewUser(t *testing.T) {
	f := newFixture(t, 0, 0, nil)

	got := f.balance(t)
	if got != (balanceResponse{}) {
		t.Fatalf("balance = %+v, want an empty wallet", got)
	}
}

func TestUnlockChapter_Succeeds(t *testing.T) {
	f := newFixture(t, 100, 0, nil)

	receipt := f.unlock(t, "unlock-1")
	if receipt.CoinsSpent != 5 {
		t.Fatalf("coins spent = %d, want 5", receipt.CoinsSpent)
	}
	if receipt.BalanceAfter != 95 {
		t.Fatalf("balance after = %d, want 95", receipt.BalanceAfter)
	}
	if got := f.balance(t).Balance; got != 95 {
		t.Fatalf("wallet balance = %d, want 95", got)
	}
}

// I-COIN-05 — the unlock row references the ledger row created in the same
// transaction.
func TestUnlockChapter_ChapterUnlockReferencesCreatedLedgerID(t *testing.T) {
	f := newFixture(t, 100, 0, nil)

	receipt := f.unlock(t, "unlock-ref")

	var unlock entities.ChapterUnlock
	err := f.env.MakeMe.DB.
		Where("user_id = ? AND chapter_id = ?", f.reader.ID, f.chapterID).
		Take(&unlock).Error
	if err != nil {
		t.Fatalf("load unlock row: %v", err)
	}
	if fmt.Sprint(unlock.LedgerID) != receipt.LedgerID {
		t.Fatalf("chapter_unlocks.ledger_id = %d, want the created ledger row %s",
			unlock.LedgerID, receipt.LedgerID)
	}
	if unlock.CoinsSpent != 5 {
		t.Fatalf("coins_spent = %d, want the 5-coin price snapshot", unlock.CoinsSpent)
	}

	// The translator is credited in the same transaction.
	var earning entities.WriterEarning
	err = f.env.MakeMe.DB.Where("unlock_ledger_id = ?", unlock.LedgerID).Take(&earning).Error
	if err != nil {
		t.Fatalf("load writer earning: %v", err)
	}
	if earning.WriterID != f.writer.ID {
		t.Fatalf("earning credited to %d, want the translator %d", earning.WriterID, f.writer.ID)
	}
	if earning.GrossCoins != 5 || earning.NetCoins != 4 {
		t.Fatalf("earning = %d gross / %d net, want 5/4 after the 30%% fee",
			earning.GrossCoins, earning.NetCoins)
	}
}

// I-COIN-02 — a double-click issues two requests with two different keys. Both
// serialize on the wallet lock; exactly one debit results.
func TestUnlockChapter_ConcurrentDoubleClickYieldsOneDebit(t *testing.T) {
	f := newFixture(t, 100, 0, nil)

	// makeme caps the pool at 5 connections; two concurrent requests are well
	// within that. A larger fan-out would need the pool raised first.
	const attempts = 2

	var wg sync.WaitGroup
	codes := make([]int, attempts)
	for i := range attempts {
		wg.Go(func() {
			rec := f.env.Do(apitest.Request{
				Method:  http.MethodPost,
				Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.chapterID),
				Token:   f.token,
				Headers: map[string]string{"Idempotency-Key": fmt.Sprintf("click-%d", i)},
			})
			codes[i] = rec.Code
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
		t.Fatalf("statuses = %v, want exactly one 200 and one 409", codes)
	}

	if n := f.countLedger(t, entities.LedgerSpendUnlock); n != 1 {
		t.Fatalf("spend_unlock rows = %d, want exactly 1", n)
	}
	if got := f.balance(t).Balance; got != 95 {
		t.Fatalf("balance = %d, want exactly one 5-coin debit from 100", got)
	}
}

// A sequential second unlock with a different key is a plain conflict.
func TestUnlockChapter_AlreadyUnlockedReturns409(t *testing.T) {
	f := newFixture(t, 100, 0, nil)
	f.unlock(t, "first-key")

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.chapterID),
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": "second-key"},
	})
	apitest.AssertErrorCode(t, rec, http.StatusConflict, "CHAPTER_ALREADY_UNLOCKED")

	if got := f.balance(t).Balance; got != 95 {
		t.Fatalf("balance = %d, want only the first debit applied", got)
	}
}

// Replaying the same key is a safe repeat, not a conflict: the caller gets the
// original result and nothing new is written.
func TestUnlockChapter_SameIdempotencyKeyReplaysTheReceipt(t *testing.T) {
	f := newFixture(t, 100, 0, nil)

	first := f.unlock(t, "same-key")
	second := f.unlock(t, "same-key")

	if first.LedgerID != second.LedgerID {
		t.Fatalf("ledger id changed on replay: %s then %s", first.LedgerID, second.LedgerID)
	}
	if !second.Replayed {
		t.Fatal("the second response should be flagged as a replay")
	}
	if n := f.countLedger(t, entities.LedgerSpendUnlock); n != 1 {
		t.Fatalf("spend_unlock rows = %d, want exactly 1", n)
	}
}

// I-COIN-03 — an unaffordable unlock changes nothing at all.
func TestUnlockChapter_InsufficientCoinsReturns402AndWritesNoLedgerRow(t *testing.T) {
	f := newFixture(t, 3, 0, nil)

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.chapterID),
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": "poor"},
	})
	apitest.AssertErrorCode(t, rec, http.StatusPaymentRequired, "INSUFFICIENT_COINS")

	var ledgerRows int64
	if err := f.env.MakeMe.DB.Model(&entities.CoinLedgerEntry{}).
		Where("user_id = ?", f.reader.ID).Count(&ledgerRows).Error; err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if ledgerRows != 0 {
		t.Fatalf("ledger rows = %d, want 0", ledgerRows)
	}
	if got := f.balance(t).Balance; got != 3 {
		t.Fatalf("balance = %d, want it unchanged at 3", got)
	}

	var unlocks int64
	if err := f.env.MakeMe.DB.Model(&entities.ChapterUnlock{}).
		Where("user_id = ?", f.reader.ID).Count(&unlocks).Error; err != nil {
		t.Fatalf("count unlocks: %v", err)
	}
	if unlocks != 0 {
		t.Fatalf("unlock rows = %d, want 0", unlocks)
	}
}

// I-COIN-04 — a free chapter cannot be purchased.
func TestUnlockChapter_ZeroPriceReturns400ChapterNotForSale(t *testing.T) {
	f := newFixture(t, 100, 0, nil)

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.freeChap.ID),
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": "free"},
	})
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "CHAPTER_NOT_FOR_SALE")
}

// U-COIN-01 end to end: bonus coins are consumed before paid coins.
func TestUnlockChapter_SpendsBonusBeforePaidBalance(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	f := newFixture(t, 100, 3, &future)

	receipt := f.unlock(t, "bonus-first")
	if receipt.BonusDelta != -3 {
		t.Fatalf("bonus_delta = %d, want -3 (bonus spent first)", receipt.BonusDelta)
	}
	if receipt.Delta != -2 {
		t.Fatalf("delta = %d, want -2 (remainder from the paid balance)", receipt.Delta)
	}

	got := f.balance(t)
	if got.Balance != 98 || got.BonusBalance != 0 {
		t.Fatalf("balance = %d/%d, want 98 paid and 0 bonus", got.Balance, got.BonusBalance)
	}
}

// I-COIN-10 — bonus that lapsed before the nightly job ran is treated as zero,
// and the write-off is recorded alongside the spend.
func TestUnlockChapter_ExpiredBonusBeforeCronTreatsBonusAsZero(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	f := newFixture(t, 100, 50, &past)

	receipt := f.unlock(t, "stale-bonus")
	if receipt.BonusDelta != 0 {
		t.Fatalf("bonus_delta = %d, want 0: expired bonus is unspendable", receipt.BonusDelta)
	}
	if receipt.Delta != -5 {
		t.Fatalf("delta = %d, want the full -5 from the paid balance", receipt.Delta)
	}

	if n := f.countLedger(t, entities.LedgerBonusExpire); n != 1 {
		t.Fatalf("bonus_expire rows = %d, want 1 written alongside the spend", n)
	}

	got := f.balance(t)
	if got.Balance != 95 || got.BonusBalance != 0 {
		t.Fatalf("balance = %d/%d, want 95 paid and 0 bonus", got.Balance, got.BonusBalance)
	}
	if got.BonusExpiresAt != "" {
		t.Fatal("a zeroed bonus must clear its expiry timestamp")
	}
}

func TestUnlockChapter_RequiresIdempotencyKey(t *testing.T) {
	f := newFixture(t, 100, 0, nil)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/chapters/%d/unlock", f.chapterID),
		Token:  f.token,
	})
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED")
}

func TestUnlockChapter_RequiresAuthentication(t *testing.T) {
	f := newFixture(t, 100, 0, nil)

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.chapterID),
		Headers: map[string]string{"Idempotency-Key": "anon"},
	})
	apitest.AssertStatus(t, rec, http.StatusUnauthorized)
}

// Reusing a key for a different chapter is a conflict, not a replay: returning
// the old receipt would confirm an unlock the caller never requested.
func TestUnlockChapter_SameKeyDifferentChapterConflicts(t *testing.T) {
	f := newFixture(t, 100, 0, nil)
	m := f.env.MakeMe

	other := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = f.novel.ID
		c.ChapterNo = 88
		c.PriceCoins = 5
		c.Status = entities.ChapterPublished
	}).Please()

	f.unlock(t, "shared-key")

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", other.ID),
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": "shared-key"},
	})
	apitest.AssertErrorCode(t, rec, http.StatusConflict, "IDEMPOTENCY_KEY_CONFLICT")
}

func TestCoinPacks_AreListedPublicly(t *testing.T) {
	f := newFixture(t, 0, 0, nil)

	rec := f.env.GET("/api/v1/coin-packs")
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[apitest.List[struct {
		ID          string `json:"id"`
		Coins       int    `json:"coins"`
		BonusCoins  int    `json:"bonus_coins"`
		PriceSatang int    `json:"price_satang"`
	}]](t, rec)
	if len(body.Data) != 1 || body.Data[0].Coins != 240 {
		t.Fatalf("packs = %+v, want the seeded 240-coin pack", body.Data)
	}
}

// I-COIN-01M — retrying POST /purchases with one key creates one pending row.
func TestCreatePurchase_SameIdempotencyKeyCreatesOnePendingRow(t *testing.T) {
	f := newFixture(t, 0, 0, nil)

	create := func() purchaseResponse {
		rec := f.env.Do(apitest.Request{
			Method:  http.MethodPost,
			Path:    "/api/v1/purchases",
			Token:   f.token,
			Body:    map[string]string{"pack_id": fmt.Sprint(f.packID)},
			Headers: map[string]string{"Idempotency-Key": "buy-once"},
		})
		apitest.AssertStatus(t, rec, http.StatusCreated)
		return apitest.DecodeJSON[purchaseResponse](t, rec)
	}

	first := create()
	second := create()

	if first.PurchaseID != second.PurchaseID {
		t.Fatalf("purchase id changed on retry: %s then %s", first.PurchaseID, second.PurchaseID)
	}

	var count int64
	if err := f.env.MakeMe.DB.Model(&entities.Purchase{}).
		Where("user_id = ?", f.reader.ID).Count(&count).Error; err != nil {
		t.Fatalf("count purchases: %v", err)
	}
	if count != 1 {
		t.Fatalf("purchase rows = %d, want exactly 1", count)
	}
}

// I-COIN-07 — completing the same purchase twice credits the wallet once.
func TestMockComplete_TwiceCreditsWalletOnce(t *testing.T) {
	f := newFixture(t, 0, 0, nil)

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    "/api/v1/purchases",
		Token:   f.token,
		Body:    map[string]string{"pack_id": fmt.Sprint(f.packID)},
		Headers: map[string]string{"Idempotency-Key": "buy"},
	})
	apitest.AssertStatus(t, rec, http.StatusCreated)
	purchase := apitest.DecodeJSON[purchaseResponse](t, rec)

	complete := func() receiptResponse {
		rec := f.env.Do(apitest.Request{
			Method:  http.MethodPost,
			Path:    "/api/v1/purchases/" + purchase.PurchaseID + "/mock-complete",
			Token:   f.token,
			Headers: map[string]string{"Idempotency-Key": "complete"},
		})
		apitest.AssertStatus(t, rec, http.StatusOK)
		return apitest.DecodeJSON[receiptResponse](t, rec)
	}

	first := complete()
	second := complete()

	if first.LedgerID != second.LedgerID {
		t.Fatalf("ledger id changed on replay: %s then %s", first.LedgerID, second.LedgerID)
	}
	if !second.Replayed {
		t.Fatal("the second completion should be flagged as a replay")
	}
	if n := f.countLedger(t, entities.LedgerTopup); n != 1 {
		t.Fatalf("topup rows = %d, want exactly 1", n)
	}

	got := f.balance(t)
	if got.Balance != 240 || got.BonusBalance != 20 {
		t.Fatalf("balance = %d/%d, want the pack credited once (240 + 20 bonus)",
			got.Balance, got.BonusBalance)
	}
	if got.BonusExpiresAt == "" {
		t.Fatal("a bonus grant must set an expiry deadline")
	}
}

func TestMockComplete_MarksThePurchaseSucceeded(t *testing.T) {
	f := newFixture(t, 0, 0, nil)

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    "/api/v1/purchases",
		Token:   f.token,
		Body:    map[string]string{"pack_id": fmt.Sprint(f.packID)},
		Headers: map[string]string{"Idempotency-Key": "buy"},
	})
	purchase := apitest.DecodeJSON[purchaseResponse](t, rec)

	rec = f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    "/api/v1/purchases/" + purchase.PurchaseID + "/mock-complete",
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": "complete"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	var row entities.Purchase
	if err := f.env.MakeMe.DB.Where("id = ?", purchase.PurchaseID).Take(&row).Error; err != nil {
		t.Fatalf("load purchase: %v", err)
	}
	if row.Status != entities.PurchaseSucceeded {
		t.Fatalf("status = %q, want succeeded", row.Status)
	}
	if row.LedgerID == nil {
		t.Fatal("a completed purchase must reference its credit ledger row")
	}
	if row.CompletedAt == nil {
		t.Fatal("a completed purchase must be timestamped")
	}
}

func TestMockFail_LeavesTheWalletUntouched(t *testing.T) {
	f := newFixture(t, 0, 0, nil)

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    "/api/v1/purchases",
		Token:   f.token,
		Body:    map[string]string{"pack_id": fmt.Sprint(f.packID)},
		Headers: map[string]string{"Idempotency-Key": "buy"},
	})
	purchase := apitest.DecodeJSON[purchaseResponse](t, rec)

	rec = f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/purchases/" + purchase.PurchaseID + "/mock-fail",
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	if got := f.balance(t).Total; got != 0 {
		t.Fatalf("balance = %d, want 0 after a failed purchase", got)
	}
	if n := f.countLedger(t, entities.LedgerTopup); n != 0 {
		t.Fatalf("topup rows = %d, want 0", n)
	}
}

// I-COIN-08 — with PAYMENTS_MOCK_ENABLED=false the mock routes are not
// registered at all, so gin answers 404 structurally.
func TestMockComplete_Returns404WhenPaymentsMockDisabled(t *testing.T) {
	cfg := apitest.Config()
	cfg.PaymentsMockEnabled = false
	env := apitest.NewWith(t, cfg)

	reader := env.AUser()
	token := env.TokenFor(reader)

	for _, path := range []string{"/api/v1/purchases/1/mock-complete", "/api/v1/purchases/1/mock-fail"} {
		rec := env.Do(apitest.Request{
			Method:  http.MethodPost,
			Path:    path,
			Token:   token,
			Headers: map[string]string{"Idempotency-Key": "nope"},
		})
		apitest.AssertStatus(t, rec, http.StatusNotFound)
	}

	// The rest of the coin API still works with the flag off.
	apitest.AssertStatus(t, env.GET("/api/v1/coin-packs"), http.StatusOK)
}

// I-COIN-06 — an admin adjustment records the actor and the reason.
func TestAdminWalletAdjust_WritesAdjustLedgerWithActorAndReason(t *testing.T) {
	f := newFixture(t, 10, 0, nil)
	admin := f.env.AUser(entities.RoleAdmin)
	adminToken := f.env.TokenFor(admin)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/admin/wallet-adjust",
		Token:  adminToken,
		Body: map[string]any{
			"user_id": fmt.Sprint(f.reader.ID),
			"delta":   25,
			"reason":  "ชดเชยระบบขัดข้อง",
		},
		Headers: map[string]string{"Idempotency-Key": "adjust-1"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	var entry entities.CoinLedgerEntry
	err := f.env.MakeMe.DB.
		Where("user_id = ? AND kind = ?", f.reader.ID, entities.LedgerAdjust).
		Take(&entry).Error
	if err != nil {
		t.Fatalf("load adjust ledger row: %v", err)
	}
	if entry.ActorUserID == nil || *entry.ActorUserID != admin.ID {
		t.Fatalf("actor = %v, want the acting admin %d", entry.ActorUserID, admin.ID)
	}
	if entry.Reason == nil || *entry.Reason != "ชดเชยระบบขัดข้อง" {
		t.Fatalf("reason = %v, want the supplied reason", entry.Reason)
	}
	if entry.Delta != 25 {
		t.Fatalf("delta = %d, want 25", entry.Delta)
	}
	if got := f.balance(t).Balance; got != 35 {
		t.Fatalf("balance = %d, want 35", got)
	}
}

func TestAdminWalletAdjust_RequiresAdminRoleAndReason(t *testing.T) {
	f := newFixture(t, 10, 0, nil)

	t.Run("a plain reader is forbidden", func(t *testing.T) {
		rec := f.env.Do(apitest.Request{
			Method:  http.MethodPost,
			Path:    "/api/v1/admin/wallet-adjust",
			Token:   f.token,
			Body:    map[string]any{"user_id": fmt.Sprint(f.reader.ID), "delta": 25, "reason": "nope"},
			Headers: map[string]string{"Idempotency-Key": "denied"},
		})
		apitest.AssertStatus(t, rec, http.StatusForbidden)
	})

	t.Run("a reason is mandatory for the audit trail", func(t *testing.T) {
		admin := f.env.AUser(entities.RoleAdmin)
		rec := f.env.Do(apitest.Request{
			Method:  http.MethodPost,
			Path:    "/api/v1/admin/wallet-adjust",
			Token:   f.env.TokenFor(admin),
			Body:    map[string]any{"user_id": fmt.Sprint(f.reader.ID), "delta": 25},
			Headers: map[string]string{"Idempotency-Key": "no-reason"},
		})
		apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "REASON_REQUIRED")
	})
}

// The PRD's release gate: sum(delta) must equal the cached balance.
func TestLedger_ReconcilesWithTheWalletBalance(t *testing.T) {
	f := newFixture(t, 0, 0, nil)

	// A mixed sequence: top up, unlock, admin adjust.
	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    "/api/v1/purchases",
		Token:   f.token,
		Body:    map[string]string{"pack_id": fmt.Sprint(f.packID)},
		Headers: map[string]string{"Idempotency-Key": "buy"},
	})
	purchase := apitest.DecodeJSON[purchaseResponse](t, rec)
	f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    "/api/v1/purchases/" + purchase.PurchaseID + "/mock-complete",
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": "complete"},
	})
	f.unlock(t, "reconcile-unlock")

	admin := f.env.AUser(entities.RoleAdmin)
	f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    "/api/v1/admin/wallet-adjust",
		Token:   f.env.TokenFor(admin),
		Body:    map[string]any{"user_id": fmt.Sprint(f.reader.ID), "delta": 7, "reason": "test"},
		Headers: map[string]string{"Idempotency-Key": "adjust"},
	})

	var sums struct {
		Delta      int
		BonusDelta int
	}
	err := f.env.MakeMe.DB.Model(&entities.CoinLedgerEntry{}).
		Select("COALESCE(SUM(delta),0) AS delta, COALESCE(SUM(bonus_delta),0) AS bonus_delta").
		Where("user_id = ?", f.reader.ID).
		Take(&sums).Error
	if err != nil {
		t.Fatalf("sum ledger: %v", err)
	}

	got := f.balance(t)
	if sums.Delta != got.Balance {
		t.Fatalf("sum(delta) = %d but wallet balance = %d", sums.Delta, got.Balance)
	}
	if sums.BonusDelta != got.BonusBalance {
		t.Fatalf("sum(bonus_delta) = %d but bonus balance = %d", sums.BonusDelta, got.BonusBalance)
	}
}

func TestLedgerEndpoint_ListsHistoryNewestFirst(t *testing.T) {
	f := newFixture(t, 100, 0, nil)
	f.unlock(t, "ledger-unlock")

	rec := f.env.GETAuth("/api/v1/me/wallet/ledger", f.token)
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[apitest.List[ledgerEntryResponse]](t, rec)
	if len(body.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1; body = %s", len(body.Data), rec.Body.String())
	}
	entry := body.Data[0]
	if entry.Kind != entities.LedgerSpendUnlock {
		t.Fatalf("kind = %q, want spend_unlock", entry.Kind)
	}
	if entry.Delta != -5 || entry.BalanceAfter != 95 {
		t.Fatalf("entry = %+v, want a -5 debit leaving 95", entry)
	}
}

func TestLedgerEndpoint_IsScopedToTheCaller(t *testing.T) {
	f := newFixture(t, 100, 0, nil)
	f.unlock(t, "mine")

	other := f.env.AUser()
	rec := f.env.GETAuth("/api/v1/me/wallet/ledger", f.env.TokenFor(other))
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[apitest.List[ledgerEntryResponse]](t, rec)
	if len(body.Data) != 0 {
		t.Fatalf("another user's ledger leaked %d rows", len(body.Data))
	}
}

// A translator unlocking their own chapter must not pay themselves.
func TestUnlockChapter_TranslatorUnlockingOwnChapterEarnsNothing(t *testing.T) {
	f := newFixture(t, 100, 0, nil)
	m := f.env.MakeMe

	m.ANewWalletBalance().With(func(w *entities.WalletBalance) {
		w.UserID = f.writer.ID
		w.Balance = 100
	}).Please()

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.chapterID),
		Token:   f.env.TokenFor(f.writer),
		Headers: map[string]string{"Idempotency-Key": "self-unlock"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	var earnings int64
	if err := f.env.MakeMe.DB.Model(&entities.WriterEarning{}).
		Where("writer_id = ?", f.writer.ID).Count(&earnings).Error; err != nil {
		t.Fatalf("count earnings: %v", err)
	}
	if earnings != 0 {
		t.Fatalf("earnings = %d, want 0: a translator must not pay themselves", earnings)
	}
}
