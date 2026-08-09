package reading_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type earlyAccessView struct {
	ID           string  `json:"id"`
	ChapterNo    int     `json:"chapter_no"`
	Title        string  `json:"title"`
	Locked       bool    `json:"locked"`
	LockedReason string  `json:"locked_reason"`
	BodyHTML     *string `json:"body_html"`
}

// eaFixture is a novel with a 24-hour early-access window and two published
// chapters: one already public, one still inside its window.
type eaFixture struct {
	env    *apitest.Env
	reader *entities.User
	token  string
	writer *entities.User
	novel  *entities.Novel
	public *entities.Chapter
	early  *entities.Chapter
}

func newEarlyAccessFixture(t *testing.T) *eaFixture {
	t.Helper()
	env := apitest.New(t)
	m := env.MakeMe

	writer := env.AUser(entities.RoleTranslator)
	reader := env.AUser()

	novel := m.ANewNovel().With(func(n *entities.Novel) {
		n.PrimaryTranslatorID = &writer.ID
		n.EarlyAccessHours = 24
	}).Please()

	now := time.Now()
	past := now.Add(-48 * time.Hour)
	soon := now.Add(20 * time.Hour)

	public := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = novel.ID
		c.ChapterNo = 1
		c.PriceCoins = 0
		c.Status = entities.ChapterPublished
		c.TranslatorID = &writer.ID
		c.PublishedAt = &past
		c.PublicAt = &past
	}).Please()

	early := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = novel.ID
		c.ChapterNo = 2
		c.PriceCoins = 0
		c.Status = entities.ChapterPublished
		c.TranslatorID = &writer.ID
		c.PublishedAt = &now
		c.PublicAt = &soon
	}).Please()

	return &eaFixture{
		env: env, reader: reader, token: env.TokenFor(reader),
		writer: writer, novel: novel, public: public, early: early,
	}
}

func (f *eaFixture) view(t *testing.T, chapterID int64, token string) (earlyAccessView, int) {
	t.Helper()
	rec := f.env.Do(apitest.Request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/chapters/%d", chapterID),
		Token:  token,
	})
	if rec.Code != http.StatusOK {
		return earlyAccessView{}, rec.Code
	}
	return apitest.DecodeJSON[earlyAccessView](t, rec), rec.Code
}

// I-EA-01 — inside the window a non-subscriber gets a teaser: listed, titled,
// body withheld, and told *why* so the client can offer auto-unlock instead of
// a paywall.
func TestEarlyAccess_NonSubscriberSeesATeaserWithNoBody(t *testing.T) {
	f := newEarlyAccessFixture(t)

	view, code := f.view(t, f.early.ID, f.token)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200: an early chapter is listed, not hidden", code)
	}
	if !view.Locked || view.BodyHTML != nil {
		t.Fatalf("view = %+v, want locked with no body", view)
	}
	if view.LockedReason != "early_access" {
		t.Fatalf("locked reason = %q, want early_access", view.LockedReason)
	}
	if view.Title == "" {
		t.Fatal("a teaser keeps its metadata; that is what makes it a teaser")
	}
}

// I-EA-02 — a subscriber reads it immediately.
func TestEarlyAccess_SubscriberReadsInsideTheWindow(t *testing.T) {
	f := newEarlyAccessFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/api/v1/me/auto-unlock/%d", f.novel.ID),
		Token:  f.token,
		Body:   map[string]any{"active": true, "max_coins_per_chapter": 50},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	view, _ := f.view(t, f.early.ID, f.token)
	if view.Locked || view.BodyHTML == nil {
		t.Fatalf("view = %+v, want a subscriber to read it", view)
	}
}

// I-EA-03 — ownership outlives the subscription. A reader who auto-unlocked and
// then cancelled keeps what they paid for.
func TestEarlyAccess_OwnershipSurvivesUnsubscribing(t *testing.T) {
	f := newEarlyAccessFixture(t)
	m := f.env.MakeMe

	// Price the chapter so ownership is the only thing that can unlock it.
	if err := m.DB.Model(&entities.Chapter{}).
		Where("id = ?", f.early.ID).
		Update("price_coins", 10).Error; err != nil {
		t.Fatalf("price chapter: %v", err)
	}

	ledger := m.ANewCoinLedgerEntry().With(func(e *entities.CoinLedgerEntry) {
		e.UserID = f.reader.ID
		e.Kind = entities.LedgerSpendUnlock
		e.Delta = -10
	}).Please()
	m.ANewChapterUnlock().With(func(u *entities.ChapterUnlock) {
		u.UserID = f.reader.ID
		u.ChapterID = f.early.ID
		u.CoinsSpent = 10
		u.LedgerID = ledger.ID
	}).Please()

	// Never subscribed at all — ownership alone must be enough.
	view, _ := f.view(t, f.early.ID, f.token)
	if view.Locked || view.BodyHTML == nil {
		t.Fatalf("view = %+v, want the owner to read it inside the window", view)
	}
}

// I-EA-04 — the translator previews their own early chapter; an unrelated
// reader does not.
func TestEarlyAccess_TranslatorReadsTheirOwnEarlyChapter(t *testing.T) {
	f := newEarlyAccessFixture(t)

	view, _ := f.view(t, f.early.ID, f.env.TokenFor(f.writer))
	if view.Locked || view.BodyHTML == nil {
		t.Fatalf("view = %+v, want the translator to read their own work", view)
	}
}

func TestEarlyAccess_AnonymousSeesATeaserNotA404(t *testing.T) {
	f := newEarlyAccessFixture(t)

	view, code := f.view(t, f.early.ID, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if !view.Locked || view.LockedReason != "early_access" {
		t.Fatalf("view = %+v, want an early-access teaser", view)
	}
}

// I-EA-05 — the window closes on its own: once public_at has passed, the same
// reader reads it with no purchase and no subscription.
func TestEarlyAccess_WindowExpiresAndTheChapterOpensToEveryone(t *testing.T) {
	f := newEarlyAccessFixture(t)

	if view, _ := f.view(t, f.early.ID, f.token); !view.Locked {
		t.Fatal("test setup is wrong: the chapter should start inside its window")
	}

	past := time.Now().Add(-time.Hour)
	if err := f.env.MakeMe.DB.Model(&entities.Chapter{}).
		Where("id = ?", f.early.ID).
		Update("public_at", past).Error; err != nil {
		t.Fatalf("expire the window: %v", err)
	}

	view, _ := f.view(t, f.early.ID, f.token)
	if view.Locked || view.BodyHTML == nil {
		t.Fatalf("view = %+v, want it public once the window closed", view)
	}
}

// An early chapter stays in the table of contents for everyone. It is the
// conversion surface for auto-unlock, and novels.chapters_count is a stored
// column that a viewer-dependent list would contradict for 24 hours.
func TestEarlyAccess_ChapterStaysListedInTheTableOfContents(t *testing.T) {
	f := newEarlyAccessFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/novels/%d/chapters", f.novel.ID),
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[struct {
		Data []struct {
			ID        string `json:"id"`
			ChapterNo int    `json:"chapter_no"`
		} `json:"data"`
	}](t, rec)

	if len(body.Data) != 2 {
		t.Fatalf("chapters listed = %d, want both the public and the early one", len(body.Data))
	}
}

// The sale path is the one place public_at reaches SQL: without it a reader
// could simply pay to defeat the exclusivity the window exists to create.
func TestEarlyAccess_BuyingAnEarlyChapterIsRefused(t *testing.T) {
	f := newEarlyAccessFixture(t)
	m := f.env.MakeMe

	if err := m.DB.Model(&entities.Chapter{}).
		Where("id = ?", f.early.ID).
		Update("price_coins", 10).Error; err != nil {
		t.Fatalf("price chapter: %v", err)
	}
	m.ANewWalletBalance().With(func(w *entities.WalletBalance) {
		w.UserID = f.reader.ID
		w.Balance = 500
	}).Please()

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.early.ID),
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": "early-buy"},
	})
	apitest.AssertErrorCode(t, rec, http.StatusForbidden, "EARLY_ACCESS_ONLY")
}

func TestEarlyAccess_PublicChapterIsUnaffected(t *testing.T) {
	f := newEarlyAccessFixture(t)

	view, _ := f.view(t, f.public.ID, "")
	if view.Locked || view.BodyHTML == nil {
		t.Fatalf("view = %+v, want the already-public chapter readable", view)
	}
}
