package wallet_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

// tipFixture is one published chapter by a translator, plus a funded reader.
type tipFixture struct {
	env     *apitest.Env
	reader  *entities.User
	token   string
	writer  *entities.User
	novel   *entities.Novel
	chapter *entities.Chapter
}

func newTipFixture(t *testing.T, balance, bonus int) *tipFixture {
	t.Helper()
	env := apitest.New(t)
	m := env.MakeMe

	writer := env.AUser(entities.RoleTranslator)
	reader := env.AUser()

	novel := m.ANewNovel().With(func(n *entities.Novel) {
		n.PrimaryTranslatorID = &writer.ID
		n.TipsEnabled = true
	}).Please()

	chapter := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = novel.ID
		c.ChapterNo = 1
		c.PriceCoins = 0
		c.Status = entities.ChapterPublished
		c.TranslatorID = &writer.ID
	}).Please()

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
		m.ANewCoinLedgerEntry().With(func(e *entities.CoinLedgerEntry) {
			e.UserID = reader.ID
			e.Kind = entities.LedgerTopup
			e.Delta = balance
			e.BonusDelta = bonus
			e.BalanceAfter = balance
			e.BonusBalanceAfter = bonus
		}).Please()
	}

	return &tipFixture{
		env: env, reader: reader, token: env.TokenFor(reader),
		writer: writer, novel: novel, chapter: chapter,
	}
}

func (f *tipFixture) tip(t *testing.T, coins int, key string) *httptest.ResponseRecorder {
	t.Helper()
	return f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/tip", f.chapter.ID),
		Token:   f.token,
		Body:    map[string]any{"coins": coins},
		Headers: map[string]string{"Idempotency-Key": key},
	})
}

func (f *tipFixture) tipAs(t *testing.T, token string, coins int, key string) *httptest.ResponseRecorder {
	t.Helper()
	return f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/tip", f.chapter.ID),
		Token:   token,
		Body:    map[string]any{"coins": coins},
		Headers: map[string]string{"Idempotency-Key": key},
	})
}

// I-TIP-01 — a tip writes a `tip` ledger row and a `tip` earning, net of fee.
func TestTip_WritesTipLedgerRowAndWriterEarningNetOfPlatformFee(t *testing.T) {
	f := newTipFixture(t, 500, 0)

	rec := f.tip(t, 100, "tip-1")
	apitest.AssertStatus(t, rec, http.StatusOK)
	receipt := apitest.DecodeJSON[receiptResponse](t, rec)
	if receipt.CoinsSpent != 100 || receipt.BalanceAfter != 400 {
		t.Fatalf("receipt = %+v, want 100 spent and 400 left", receipt)
	}

	var entry entities.CoinLedgerEntry
	if err := f.env.MakeMe.DB.
		Where("user_id = ? AND kind = ?", f.reader.ID, entities.LedgerTip).
		Take(&entry).Error; err != nil {
		t.Fatalf("load tip ledger row: %v", err)
	}
	// A tip must not be filed as spend_unlock, or the reader's history calls it
	// "ปลดล็อกบท" and every unlock-derived statistic counts it as a sale.
	if entry.Delta != -100 || entry.BonusDelta != 0 {
		t.Fatalf("ledger delta = %d/%d, want -100/0", entry.Delta, entry.BonusDelta)
	}

	var earning entities.WriterEarning
	if err := f.env.MakeMe.DB.
		Where("writer_id = ? AND kind = ?", f.writer.ID, entities.EarningTip).
		Take(&earning).Error; err != nil {
		t.Fatalf("load tip earning: %v", err)
	}
	// The 30% platform fee applies to tips too; a fee-free channel would be an
	// arbitrage against priced chapters.
	if earning.GrossCoins != 100 || earning.NetCoins != 70 {
		t.Fatalf("earning = %d gross/%d net, want 100/70", earning.GrossCoins, earning.NetCoins)
	}

	// The translator is paid through writer_earnings, never a second wallet:
	// locking two wallet rows in one command is what would let two opposing
	// tips deadlock.
	var writerWallets int64
	if err := f.env.MakeMe.DB.Model(&entities.WalletBalance{}).
		Where("user_id = ?", f.writer.ID).Count(&writerWallets).Error; err != nil {
		t.Fatalf("count writer wallets: %v", err)
	}
	if writerWallets != 0 {
		t.Fatal("a tip credited the translator's spendable wallet")
	}

	assertLedgerReconciles(t, f.env, f.reader.ID)
}

// I-TIP-02 — bonus coins can never fund a tip, even when they would cover it.
func TestTip_RefusesToSpendBonusCoinsEvenWhenAmpleAndReports402(t *testing.T) {
	f := newTipFixture(t, 10, 500)

	rec := f.tip(t, 100, "bonus-tip")
	// A distinct code, because "เหรียญไม่พอ" next to a visible 500-coin bonus
	// balance is a guaranteed support ticket.
	apitest.AssertErrorCode(t, rec, http.StatusPaymentRequired, "INSUFFICIENT_PAID_COINS")

	var count int64
	if err := f.env.MakeMe.DB.Model(&entities.CoinLedgerEntry{}).
		Where("user_id = ? AND kind = ?", f.reader.ID, entities.LedgerTip).
		Count(&count).Error; err != nil {
		t.Fatalf("count tips: %v", err)
	}
	if count != 0 {
		t.Fatal("a refused tip still wrote a ledger row")
	}
	assertLedgerReconciles(t, f.env, f.reader.ID)
}

func TestTip_SpendsPaidCoinsWhileLeavingBonusUntouched(t *testing.T) {
	f := newTipFixture(t, 200, 500)

	rec := f.tip(t, 100, "paid-only")
	apitest.AssertStatus(t, rec, http.StatusOK)
	receipt := apitest.DecodeJSON[receiptResponse](t, rec)

	if receipt.BalanceAfter != 100 || receipt.BonusBalanceAfter != 500 {
		t.Fatalf("receipt = %+v, want 100 paid left and the 500 bonus intact", receipt)
	}
	if receipt.BonusDelta != 0 {
		t.Fatalf("bonus delta = %d, want 0", receipt.BonusDelta)
	}
	assertLedgerReconciles(t, f.env, f.reader.ID)
}

// I-TIP-03 — repeat tipping is legitimate, so the idempotency key is the only
// dedupe there is: two tips with different keys both land, the same key replays.
func TestTip_RepeatsWithNewKeysAndReplaysWithTheSameKey(t *testing.T) {
	f := newTipFixture(t, 500, 0)

	first := apitest.DecodeJSON[receiptResponse](t, f.tip(t, 50, "a"))
	second := apitest.DecodeJSON[receiptResponse](t, f.tip(t, 50, "b"))
	if first.LedgerID == second.LedgerID {
		t.Fatal("a second tip with a new key must be a second debit")
	}
	if second.BalanceAfter != 400 {
		t.Fatalf("balance = %d, want 400 after two 50-coin tips", second.BalanceAfter)
	}

	replay := apitest.DecodeJSON[receiptResponse](t, f.tip(t, 50, "b"))
	if !replay.Replayed || replay.LedgerID != second.LedgerID {
		t.Fatalf("replay = %+v, want the second receipt back", replay)
	}
	if replay.BalanceAfter != 400 {
		t.Fatalf("balance = %d, want the replay to charge nothing", replay.BalanceAfter)
	}

	var count int64
	if err := f.env.MakeMe.DB.Model(&entities.CoinLedgerEntry{}).
		Where("user_id = ? AND kind = ?", f.reader.ID, entities.LedgerTip).
		Count(&count).Error; err != nil {
		t.Fatalf("count tips: %v", err)
	}
	if count != 2 {
		t.Fatalf("tip rows = %d, want 2", count)
	}
	assertLedgerReconciles(t, f.env, f.reader.ID)
}

func TestTip_RequiresAnIdempotencyKey(t *testing.T) {
	f := newTipFixture(t, 500, 0)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/chapters/%d/tip", f.chapter.ID),
		Token:  f.token,
		Body:   map[string]any{"coins": 50},
	})
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED")
}

func TestTip_DisabledOnTheNovelReturns400(t *testing.T) {
	f := newTipFixture(t, 500, 0)

	if err := f.env.MakeMe.DB.Model(&entities.Novel{}).
		Where("id = ?", f.novel.ID).
		Update("tips_enabled", false).Error; err != nil {
		t.Fatalf("disable tips: %v", err)
	}

	apitest.AssertErrorCode(t, f.tip(t, 50, "off"), http.StatusBadRequest, "TIPS_DISABLED")
}

func TestTip_OwnChapterReturns400(t *testing.T) {
	f := newTipFixture(t, 500, 0)

	// Fund the translator so the refusal cannot be mistaken for lack of coins.
	f.env.MakeMe.ANewWalletBalance().With(func(w *entities.WalletBalance) {
		w.UserID = f.writer.ID
		w.Balance = 500
	}).Please()

	rec := f.tipAs(t, f.env.TokenFor(f.writer), 50, "self")
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "CANNOT_TIP_SELF")
}

func TestTip_RejectsAmountsOutsideTheAllowedRange(t *testing.T) {
	for _, tc := range []struct {
		name  string
		coins int
	}{
		{"zero", 0},
		{"negative", -5},
		{"above the cap", 1001},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newTipFixture(t, 5000, 0)
			rec := f.tip(t, tc.coins, "range")
			apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "INVALID_AMOUNT")
		})
	}
}

func TestTip_AtTheCapIsAccepted(t *testing.T) {
	f := newTipFixture(t, 5000, 0)
	apitest.AssertStatus(t, f.tip(t, 1000, "cap"), http.StatusOK)
}

func TestTip_RequiresAuthentication(t *testing.T) {
	f := newTipFixture(t, 500, 0)

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/tip", f.chapter.ID),
		Body:    map[string]any{"coins": 50},
		Headers: map[string]string{"Idempotency-Key": "anon"},
	})
	apitest.AssertStatus(t, rec, http.StatusUnauthorized)
}
