package reading_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type chapterViewResponse struct {
	ID           string  `json:"id"`
	NovelID      string  `json:"novel_id"`
	NovelSlug    string  `json:"novel_slug"`
	NovelTitleTH string  `json:"novel_title_th"`
	ArcNo        int     `json:"arc_no"`
	ArcName      string  `json:"arc_name"`
	ChapterNo    int     `json:"chapter_no"`
	Title        string  `json:"title"`
	PriceCoins   int     `json:"price_coins"`
	Locked       bool    `json:"locked"`
	BodyHTML     *string `json:"body_html"`
	PrevID       string  `json:"prev_id"`
	NextID       string  `json:"next_id"`
}

type progressResponse struct {
	NovelID       string  `json:"novel_id"`
	LastChapterID string  `json:"last_chapter_id"`
	LastChapterNo *int    `json:"last_chapter_no"`
	ParaAnchor    int     `json:"para_anchor"`
	Pct           float64 `json:"pct"`
}

// readerFixture gives a novel with a free chapter, a paid chapter, a draft and
// a scheduled chapter — every entitlement branch in one place.
type readerFixture struct {
	env       *apitest.Env
	reader    *entities.User
	token     string
	writer    *entities.User
	novel     *entities.Novel
	free      *entities.Chapter
	paid      *entities.Chapter
	draft     *entities.Chapter
	scheduled *entities.Chapter
}

const (
	freeBody = "<p>บทอ่านฟรี</p>"
	paidBody = "<p>บทที่ต้องปลดล็อก</p>"
)

func newReaderFixture(t *testing.T) *readerFixture {
	t.Helper()
	env := apitest.New(t)
	m := env.MakeMe

	writer := env.AUser(entities.RoleTranslator)
	reader := env.AUser()

	novel := m.ANewNovel().With(func(n *entities.Novel) {
		n.TitleTH = "เซียนดาบเก้าสายธาร"
		n.PrimaryTranslatorID = &writer.ID
	}).Please()

	arc := m.ANewArc().With(func(a *entities.Arc) {
		a.NovelID = novel.ID
		a.ArcNo = 2
		a.Name = "สำนักเมฆาวสันต์"
		a.FromChapterNo = 49
		a.ToChapterNo = 120
	}).Please()

	chapter := func(no int, price int16, status string, body string) *entities.Chapter {
		c := m.ANewChapter().With(func(c *entities.Chapter) {
			c.NovelID = novel.ID
			c.ArcID = &arc.ID
			c.ChapterNo = no
			c.PriceCoins = price
			c.Status = status
			c.TranslatorID = &writer.ID
		}).Please()
		if body != "" {
			m.ANewChapterBody().With(func(b *entities.ChapterBody) {
				b.ChapterID = c.ID
				b.BodyHTML = body
			}).Please()
		}
		return c
	}

	return &readerFixture{
		env:       env,
		reader:    reader,
		token:     env.TokenFor(reader),
		writer:    writer,
		novel:     novel,
		free:      chapter(86, 0, entities.ChapterPublished, freeBody),
		paid:      chapter(87, 5, entities.ChapterPublished, paidBody),
		draft:     chapter(90, 0, entities.ChapterDraftStatus, "<p>ร่าง</p>"),
		scheduled: chapter(89, 0, entities.ChapterScheduled, "<p>ตั้งเวลาไว้</p>"),
	}
}

func (f *readerFixture) get(t *testing.T, chapterID int64, token string) chapterViewResponse {
	t.Helper()
	rec := f.env.Do(apitest.Request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/api/v1/chapters/%d", chapterID),
		Token:  token,
	})
	apitest.AssertStatus(t, rec, http.StatusOK)
	return apitest.DecodeJSON[chapterViewResponse](t, rec)
}

// I-RD-01 — a free chapter is readable without an account.
func TestGetChapter_FreeChapterAsAnonymousReturnsBody(t *testing.T) {
	f := newReaderFixture(t)

	got := f.get(t, f.free.ID, "")
	if got.Locked {
		t.Fatal("a free chapter must never be locked")
	}
	if got.BodyHTML == nil || *got.BodyHTML != freeBody {
		t.Fatalf("body = %v, want %q", got.BodyHTML, freeBody)
	}
	if got.NovelTitleTH != "เซียนดาบเก้าสายธาร" || got.ArcName != "สำนักเมฆาวสันต์" {
		t.Fatalf("reader chrome fields missing: %+v", got)
	}
}

// I-RD-02 — a paid chapter the reader does not own returns locked with a null
// body, exactly as documented.
func TestGetChapter_LockedWithoutUnlockReturnsLockedTrueNullBody(t *testing.T) {
	f := newReaderFixture(t)

	for _, tc := range []struct {
		name  string
		token string
	}{
		{"anonymous", ""},
		{"signed in but not entitled", f.token},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := f.get(t, f.paid.ID, tc.token)
			if !got.Locked {
				t.Fatal("expected locked=true")
			}
			if got.BodyHTML != nil {
				t.Fatalf("body_html = %q, want null", *got.BodyHTML)
			}
			// Metadata still comes through so the paywall can be rendered.
			if got.PriceCoins != 5 || got.ChapterNo != 87 {
				t.Fatalf("metadata = %+v, want price 5 chapter 87", got)
			}
		})
	}
}

// I-RD-03 — after unlocking, the same request returns the body.
func TestGetChapter_LockedAfterUnlockReturnsBody(t *testing.T) {
	f := newReaderFixture(t)
	m := f.env.MakeMe

	m.ANewWalletBalance().With(func(w *entities.WalletBalance) {
		w.UserID = f.reader.ID
		w.Balance = 100
	}).Please()

	rec := f.env.Do(apitest.Request{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/api/v1/chapters/%d/unlock", f.paid.ID),
		Token:   f.token,
		Headers: map[string]string{"Idempotency-Key": "read-after-unlock"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	got := f.get(t, f.paid.ID, f.token)
	if got.Locked {
		t.Fatal("an unlocked chapter must not be locked")
	}
	if got.BodyHTML == nil || *got.BodyHTML != paidBody {
		t.Fatalf("body = %v, want %q", got.BodyHTML, paidBody)
	}

	// The entitlement is per reader, not global.
	other := f.env.AUser()
	stillLocked := f.get(t, f.paid.ID, f.env.TokenFor(other))
	if !stillLocked.Locked {
		t.Fatal("one reader's unlock must not entitle another")
	}
}

// The chapter's own translator reads their paid work without paying, and so
// does an admin.
func TestGetChapter_TranslatorAndAdminReadPaidChaptersFree(t *testing.T) {
	f := newReaderFixture(t)

	t.Run("the chapter's translator", func(t *testing.T) {
		got := f.get(t, f.paid.ID, f.env.TokenFor(f.writer))
		if got.Locked || got.BodyHTML == nil {
			t.Fatalf("the translator must read their own chapter: %+v", got)
		}
	})

	t.Run("an administrator", func(t *testing.T) {
		admin := f.env.AUser(entities.RoleAdmin)
		got := f.get(t, f.paid.ID, f.env.TokenFor(admin))
		if got.Locked || got.BodyHTML == nil {
			t.Fatalf("an admin must be able to read any chapter: %+v", got)
		}
	})

	t.Run("an unrelated translator is still charged", func(t *testing.T) {
		stranger := f.env.AUser(entities.RoleTranslator)
		got := f.get(t, f.paid.ID, f.env.TokenFor(stranger))
		if !got.Locked {
			t.Fatal("holding the translator role must not unlock someone else's chapter")
		}
	})
}

// I-SEC-01 — unpublished chapters are invisible to readers, and the 404 does
// not reveal that a draft exists.
func TestGetChapter_DraftChapterAsReaderReturns404(t *testing.T) {
	f := newReaderFixture(t)

	for _, tc := range []struct {
		name    string
		chapter *entities.Chapter
	}{
		{"draft", f.draft},
		{"scheduled", f.scheduled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, token := range []string{"", f.token} {
				rec := f.env.Do(apitest.Request{
					Method: http.MethodGet,
					Path:   fmt.Sprintf("/api/v1/chapters/%d", tc.chapter.ID),
					Token:  token,
				})
				apitest.AssertErrorCode(t, rec, http.StatusNotFound, "NOT_FOUND")
			}
		})
	}

	// Its own translator may still preview it.
	t.Run("the translator can preview their draft", func(t *testing.T) {
		got := f.get(t, f.draft.ID, f.env.TokenFor(f.writer))
		if got.BodyHTML == nil {
			t.Fatal("the translator must be able to preview their own draft")
		}
	})
}

func TestGetChapter_MissingChapterReturns404(t *testing.T) {
	f := newReaderFixture(t)

	rec := f.env.GET("/api/v1/chapters/999999")
	apitest.AssertErrorCode(t, rec, http.StatusNotFound, "NOT_FOUND")
}

func TestGetChapter_InvalidID(t *testing.T) {
	f := newReaderFixture(t)

	rec := f.env.GET("/api/v1/chapters/not-a-number")
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "BAD_ID")
}

// R-11 — prev/next skip unpublished chapters, and next is absent on the last
// published chapter.
func TestChapterNeighbours(t *testing.T) {
	f := newReaderFixture(t)

	t.Run("the view carries prev and next ids", func(t *testing.T) {
		got := f.get(t, f.paid.ID, "")
		if got.PrevID != fmt.Sprint(f.free.ID) {
			t.Fatalf("prev_id = %q, want the free chapter %d", got.PrevID, f.free.ID)
		}
		// 89 and 90 are scheduled and draft, so 87 is the last readable one.
		if got.NextID != "" {
			t.Fatalf("next_id = %q, want empty: later chapters are unpublished", got.NextID)
		}
	})

	t.Run("the prev endpoint resolves", func(t *testing.T) {
		rec := f.env.GET(fmt.Sprintf("/api/v1/chapters/%d/prev", f.paid.ID))
		apitest.AssertStatus(t, rec, http.StatusOK)

		body := apitest.DecodeJSON[struct {
			ID string `json:"id"`
		}](t, rec)
		if body.ID != fmt.Sprint(f.free.ID) {
			t.Fatalf("id = %q, want %d", body.ID, f.free.ID)
		}
	})

	t.Run("next on the last chapter is a 404", func(t *testing.T) {
		rec := f.env.GET(fmt.Sprintf("/api/v1/chapters/%d/next", f.paid.ID))
		apitest.AssertStatus(t, rec, http.StatusNotFound)
	})
}

// I-RD-04 — reading position round-trips, which is what makes cross-device
// resume work.
func TestProgress_PutThenGetReturnsParaAnchor42(t *testing.T) {
	f := newReaderFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/api/v1/me/progress/%d", f.novel.ID),
		Token:  f.token,
		Body: map[string]any{
			"last_chapter_id": fmt.Sprint(f.paid.ID),
			"last_chapter_no": 87,
			"para_anchor":     42,
			"pct":             41.5,
		},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	rec = f.env.GETAuth(fmt.Sprintf("/api/v1/me/progress/%d", f.novel.ID), f.token)
	apitest.AssertStatus(t, rec, http.StatusOK)

	got := apitest.DecodeJSON[progressResponse](t, rec)
	if got.ParaAnchor != 42 {
		t.Fatalf("para_anchor = %d, want 42", got.ParaAnchor)
	}
	if got.LastChapterNo == nil || *got.LastChapterNo != 87 {
		t.Fatalf("last_chapter_no = %v, want 87", got.LastChapterNo)
	}
	if got.Pct != 41.5 {
		t.Fatalf("pct = %v, want 41.5", got.Pct)
	}
}

func TestProgress_UpsertsRatherThanDuplicates(t *testing.T) {
	f := newReaderFixture(t)

	save := func(anchor int, pct float64) {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodPut,
			Path:   fmt.Sprintf("/api/v1/me/progress/%d", f.novel.ID),
			Token:  f.token,
			Body:   map[string]any{"para_anchor": anchor, "pct": pct},
		})
		apitest.AssertStatus(t, rec, http.StatusOK)
	}

	save(10, 5)
	save(99, 50)

	var count int64
	if err := f.env.MakeMe.DB.Model(&entities.ReadingProgress{}).
		Where("user_id = ? AND novel_id = ?", f.reader.ID, f.novel.ID).
		Count(&count).Error; err != nil {
		t.Fatalf("count progress rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("progress rows = %d, want exactly 1", count)
	}

	rec := f.env.GETAuth(fmt.Sprintf("/api/v1/me/progress/%d", f.novel.ID), f.token)
	if got := apitest.DecodeJSON[progressResponse](t, rec); got.ParaAnchor != 99 {
		t.Fatalf("para_anchor = %d, want the latest value 99", got.ParaAnchor)
	}
}

// A percentage outside 0–100 is clamped rather than rejected: the client
// computes it from scroll position and small overshoots are normal.
func TestProgress_ClampsPercentage(t *testing.T) {
	f := newReaderFixture(t)

	for _, tc := range []struct {
		sent float64
		want float64
	}{
		{-5, 0},
		{150, 100},
		{50, 50},
	} {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodPut,
			Path:   fmt.Sprintf("/api/v1/me/progress/%d", f.novel.ID),
			Token:  f.token,
			Body:   map[string]any{"para_anchor": 1, "pct": tc.sent},
		})
		apitest.AssertStatus(t, rec, http.StatusOK)

		if got := apitest.DecodeJSON[progressResponse](t, rec); got.Pct != tc.want {
			t.Fatalf("pct %v stored as %v, want %v", tc.sent, got.Pct, tc.want)
		}
	}
}

func TestProgress_RequiresAuthentication(t *testing.T) {
	f := newReaderFixture(t)

	rec := f.env.GET(fmt.Sprintf("/api/v1/me/progress/%d", f.novel.ID))
	apitest.AssertStatus(t, rec, http.StatusUnauthorized)
}

func TestReadEvent_IsAcceptedAndRecorded(t *testing.T) {
	f := newReaderFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/chapters/%d/read-event", f.free.ID),
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusAccepted)

	var count int64
	if err := f.env.MakeMe.DB.Model(&entities.ChapterReadEvent{}).
		Where("chapter_id = ?", f.free.ID).Count(&count).Error; err != nil {
		t.Fatalf("count read events: %v", err)
	}
	if count != 1 {
		t.Fatalf("read events = %d, want 1", count)
	}
}

func TestReadEvent_WorksAnonymously(t *testing.T) {
	f := newReaderFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/chapters/%d/read-event", f.free.ID),
	})
	apitest.AssertStatus(t, rec, http.StatusAccepted)
}

// I-SEC-05 — body fetches are capped by breadth, not volume: sweeping the
// catalogue is throttled while re-reading one chapter is not.
func TestGetChapter_LimitsDistinctBodiesPerUserPerMinute(t *testing.T) {
	cfg := apitest.Config()
	cfg.RateLimitEnabled = true
	cfg.BodyFetchPerMin = 3
	env := apitest.NewWith(t, cfg)
	m := env.MakeMe

	reader := env.AUser()
	token := env.TokenFor(reader)
	novel := m.ANewNovel().Please()

	chapters := make([]*entities.Chapter, 0, 5)
	for i := range 5 {
		c := m.ANewChapter().With(func(c *entities.Chapter) {
			c.NovelID = novel.ID
			c.ChapterNo = i + 1
			c.Status = entities.ChapterPublished
		}).Please()
		chapters = append(chapters, c)
	}

	fetch := func(id int64) int {
		return env.Do(apitest.Request{
			Method: http.MethodGet,
			Path:   fmt.Sprintf("/api/v1/chapters/%d", id),
			Token:  token,
		}).Code
	}

	t.Run("distinct chapters are capped", func(t *testing.T) {
		for i := range cfg.BodyFetchPerMin {
			if code := fetch(chapters[i].ID); code != http.StatusOK {
				t.Fatalf("chapter %d returned %d, want 200", i+1, code)
			}
		}
		if code := fetch(chapters[cfg.BodyFetchPerMin].ID); code != http.StatusTooManyRequests {
			t.Fatalf("the %dth distinct chapter returned %d, want 429", cfg.BodyFetchPerMin+1, code)
		}
	})

	// Re-reading a chapter already inside the window is the normal case and
	// must never be throttled.
	t.Run("re-reading the same chapter stays allowed", func(t *testing.T) {
		for i := range 20 {
			if code := fetch(chapters[0].ID); code != http.StatusOK {
				t.Fatalf("re-read %d returned %d, want 200", i+1, code)
			}
		}
	})

	t.Run("the limit is per reader", func(t *testing.T) {
		fresh := env.AUser()
		if code := fetch(chapters[4].ID); code != http.StatusTooManyRequests {
			t.Fatalf("the first reader should still be throttled, got %d", code)
		}
		code := env.Do(apitest.Request{
			Method: http.MethodGet,
			Path:   fmt.Sprintf("/api/v1/chapters/%d", chapters[4].ID),
			Token:  env.TokenFor(fresh),
		}).Code
		if code != http.StatusOK {
			t.Fatalf("a different reader returned %d, want 200", code)
		}
	})
}
