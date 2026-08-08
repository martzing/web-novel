package library_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type shelfItemResponse struct {
	NovelID       string  `json:"novel_id"`
	Slug          string  `json:"slug"`
	TitleTH       string  `json:"title_th"`
	Status        string  `json:"status"`
	ChaptersCount int     `json:"chapters_count"`
	LastChapterNo *int    `json:"last_chapter_no"`
	Pct           float64 `json:"pct"`
}

type shelfListResponse struct {
	Data   []shelfItemResponse `json:"data"`
	Counts struct {
		Reading int `json:"reading"`
		Saved   int `json:"saved"`
		Done    int `json:"done"`
		Total   int `json:"total"`
	} `json:"counts"`
}

type bookmarkResponse struct {
	ID         string `json:"id"`
	NovelID    string `json:"novel_id"`
	ChapterID  string `json:"chapter_id"`
	ChapterNo  int    `json:"chapter_no"`
	ParaAnchor int    `json:"para_anchor"`
	Excerpt    string `json:"excerpt"`
	Note       string `json:"note"`
}

// shelfFixture gives two readers and a novel with one chapter, which is enough
// for every ownership-isolation assertion here.
type shelfFixture struct {
	env         *apitest.Env
	owner       *entities.User
	ownerToken  string
	other       *entities.User
	otherToken  string
	novel       *entities.Novel
	chapter     *entities.Chapter
	secondNovel *entities.Novel
}

func newShelfFixture(t *testing.T) *shelfFixture {
	t.Helper()
	env := apitest.New(t)
	m := env.MakeMe

	owner := env.AUser()
	other := env.AUser()

	novel := m.ANewNovel().With(func(n *entities.Novel) {
		n.ChaptersCount = 214
	}).Please()
	second := m.ANewNovel().Please()

	chapter := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = novel.ID
		c.ChapterNo = 87
		c.Status = entities.ChapterPublished
	}).Please()

	return &shelfFixture{
		env:         env,
		owner:       owner,
		ownerToken:  env.TokenFor(owner),
		other:       other,
		otherToken:  env.TokenFor(other),
		novel:       novel,
		chapter:     chapter,
		secondNovel: second,
	}
}

func (f *shelfFixture) setStatus(t *testing.T, token string, novelID int64, status string) {
	t.Helper()
	rec := f.env.Do(apitest.Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/api/v1/me/library/%d", novelID),
		Token:  token,
		Body:   map[string]string{"status": status},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)
}

func (f *shelfFixture) shelf(t *testing.T, token, tab string) shelfListResponse {
	t.Helper()
	path := "/api/v1/me/library"
	if tab != "" {
		path += "?tab=" + tab
	}
	rec := f.env.GETAuth(path, token)
	apitest.AssertStatus(t, rec, http.StatusOK)
	return apitest.DecodeJSON[shelfListResponse](t, rec)
}

// I-LIB-01 — moving a novel from reading to done updates the tab counts.
func TestLibrary_MoveReadingToDoneUpdatesCounts(t *testing.T) {
	f := newShelfFixture(t)

	f.setStatus(t, f.ownerToken, f.novel.ID, "reading")
	f.setStatus(t, f.ownerToken, f.secondNovel.ID, "saved")

	before := f.shelf(t, f.ownerToken, "")
	if before.Counts.Reading != 1 || before.Counts.Saved != 1 || before.Counts.Done != 0 {
		t.Fatalf("counts = %+v, want 1 reading and 1 saved", before.Counts)
	}

	f.setStatus(t, f.ownerToken, f.novel.ID, "done")

	after := f.shelf(t, f.ownerToken, "")
	if after.Counts.Reading != 0 || after.Counts.Done != 1 {
		t.Fatalf("counts = %+v, want the novel moved to done", after.Counts)
	}
	if after.Counts.Total != 2 {
		t.Fatalf("total = %d, want 2: a move must not duplicate the row", after.Counts.Total)
	}
}

func TestLibrary_TabFilter(t *testing.T) {
	f := newShelfFixture(t)

	f.setStatus(t, f.ownerToken, f.novel.ID, "reading")
	f.setStatus(t, f.ownerToken, f.secondNovel.ID, "done")

	reading := f.shelf(t, f.ownerToken, "reading")
	if len(reading.Data) != 1 || reading.Data[0].NovelID != fmt.Sprint(f.novel.ID) {
		t.Fatalf("reading tab = %+v, want only the reading novel", reading.Data)
	}

	done := f.shelf(t, f.ownerToken, "done")
	if len(done.Data) != 1 || done.Data[0].NovelID != fmt.Sprint(f.secondNovel.ID) {
		t.Fatalf("done tab = %+v, want only the finished novel", done.Data)
	}
}

func TestLibrary_RejectsUnknownStatus(t *testing.T) {
	f := newShelfFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/api/v1/me/library/%d", f.novel.ID),
		Token:  f.ownerToken,
		Body:   map[string]string{"status": "abandoned"},
	})
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "INVALID_STATUS")

	rec = f.env.GETAuth("/api/v1/me/library?tab=abandoned", f.ownerToken)
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "INVALID_STATUS")
}

func TestLibrary_RemoveFromShelf(t *testing.T) {
	f := newShelfFixture(t)
	f.setStatus(t, f.ownerToken, f.novel.ID, "reading")

	rec := f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   fmt.Sprintf("/api/v1/me/library/%d", f.novel.ID),
		Token:  f.ownerToken,
	})
	apitest.AssertStatus(t, rec, http.StatusNoContent)

	if got := f.shelf(t, f.ownerToken, "").Counts.Total; got != 0 {
		t.Fatalf("shelf total = %d, want 0", got)
	}
}

// The shelf card shows reading progress, so the join must surface it.
func TestLibrary_CarriesReadingProgress(t *testing.T) {
	f := newShelfFixture(t)
	f.setStatus(t, f.ownerToken, f.novel.ID, "reading")

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/api/v1/me/progress/%d", f.novel.ID),
		Token:  f.ownerToken,
		Body: map[string]any{
			"last_chapter_id": fmt.Sprint(f.chapter.ID),
			"last_chapter_no": 87,
			"para_anchor":     42,
			"pct":             41.0,
		},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	shelf := f.shelf(t, f.ownerToken, "reading")
	if len(shelf.Data) != 1 {
		t.Fatalf("shelf = %+v, want one row", shelf.Data)
	}
	item := shelf.Data[0]
	if item.LastChapterNo == nil || *item.LastChapterNo != 87 {
		t.Fatalf("last_chapter_no = %v, want 87", item.LastChapterNo)
	}
	if item.Pct != 41.0 {
		t.Fatalf("pct = %v, want 41", item.Pct)
	}
	if item.ChaptersCount != 214 {
		t.Fatalf("chapters_count = %d, want 214", item.ChaptersCount)
	}
}

func (f *shelfFixture) createBookmark(t *testing.T, token string, anchor int, excerpt string) bookmarkResponse {
	t.Helper()
	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/me/bookmarks",
		Token:  token,
		Body: map[string]any{
			"novel_id":    fmt.Sprint(f.novel.ID),
			"chapter_id":  fmt.Sprint(f.chapter.ID),
			"para_anchor": anchor,
			"excerpt":     excerpt,
		},
	})
	apitest.AssertStatus(t, rec, http.StatusCreated)
	return apitest.DecodeJSON[bookmarkResponse](t, rec)
}

// I-BM-01 — bookmarks are private to their owner.
func TestBookmarks_OnlyOwnerSeesOwnBookmarks(t *testing.T) {
	f := newShelfFixture(t)

	mine := f.createBookmark(t, f.ownerToken, 42, "ข้อความของฉัน")
	f.createBookmark(t, f.otherToken, 7, "ข้อความของคนอื่น")

	rec := f.env.GETAuth("/api/v1/me/bookmarks", f.ownerToken)
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[apitest.List[bookmarkResponse]](t, rec)
	if len(body.Data) != 1 {
		t.Fatalf("len(data) = %d, want only the caller's own bookmark", len(body.Data))
	}
	if body.Data[0].ID != mine.ID {
		t.Fatalf("bookmark id = %q, want %q", body.Data[0].ID, mine.ID)
	}
	if body.Data[0].ChapterNo != 87 {
		t.Fatalf("chapter_no = %d, want the joined chapter number 87", body.Data[0].ChapterNo)
	}
}

// I-BM-02 — deleting somebody else's bookmark is a 403, distinct from the 404
// a genuinely missing bookmark returns.
func TestBookmarks_DeleteAnotherUsersBookmarkReturns403(t *testing.T) {
	f := newShelfFixture(t)
	theirs := f.createBookmark(t, f.otherToken, 7, "ไม่ใช่ของคุณ")

	rec := f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   "/api/v1/me/bookmarks/" + theirs.ID,
		Token:  f.ownerToken,
	})
	apitest.AssertErrorCode(t, rec, http.StatusForbidden, "FORBIDDEN")

	// It must still be there for its owner.
	rec = f.env.GETAuth("/api/v1/me/bookmarks", f.otherToken)
	body := apitest.DecodeJSON[apitest.List[bookmarkResponse]](t, rec)
	if len(body.Data) != 1 {
		t.Fatal("the owner's bookmark must survive a forbidden delete")
	}
}

func TestBookmarks_DeleteMissingBookmarkReturns404(t *testing.T) {
	f := newShelfFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   "/api/v1/me/bookmarks/999999",
		Token:  f.ownerToken,
	})
	apitest.AssertErrorCode(t, rec, http.StatusNotFound, "NOT_FOUND")
}

func TestBookmarks_OwnerCanDeleteOwnBookmark(t *testing.T) {
	f := newShelfFixture(t)
	mine := f.createBookmark(t, f.ownerToken, 42, "ของฉันเอง")

	rec := f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   "/api/v1/me/bookmarks/" + mine.ID,
		Token:  f.ownerToken,
	})
	apitest.AssertStatus(t, rec, http.StatusNoContent)

	rec = f.env.GETAuth("/api/v1/me/bookmarks", f.ownerToken)
	body := apitest.DecodeJSON[apitest.List[bookmarkResponse]](t, rec)
	if len(body.Data) != 0 {
		t.Fatalf("len(data) = %d, want 0 after deletion", len(body.Data))
	}
}

func TestBookmarks_RejectAnEmptyExcerpt(t *testing.T) {
	f := newShelfFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/me/bookmarks",
		Token:  f.ownerToken,
		Body: map[string]any{
			"novel_id":    fmt.Sprint(f.novel.ID),
			"chapter_id":  fmt.Sprint(f.chapter.ID),
			"para_anchor": 1,
			"excerpt":     "   ",
		},
	})
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "INVALID_BODY")
}

// The excerpt cap is counted in runes, so a long Thai excerpt is truncated
// cleanly instead of being cut mid-character.
func TestBookmarks_LongThaiExcerptIsTruncatedByRunes(t *testing.T) {
	f := newShelfFixture(t)

	created := f.createBookmark(t, f.ownerToken, 1, strings.Repeat("ก", 600))

	if got := len([]rune(created.Excerpt)); got != 500 {
		t.Fatalf("excerpt runes = %d, want it truncated to 500", got)
	}
	for _, r := range created.Excerpt {
		if r != 'ก' {
			t.Fatalf("excerpt was cut mid-character, found %q", r)
		}
	}
}

// I-SEC-04 — no endpoint here exposes another reader's rows.
func TestLibrary_CannotSeeAnotherUsersProgressOrShelf(t *testing.T) {
	f := newShelfFixture(t)

	f.setStatus(t, f.ownerToken, f.novel.ID, "reading")
	f.env.Do(apitest.Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/api/v1/me/progress/%d", f.novel.ID),
		Token:  f.ownerToken,
		Body:   map[string]any{"para_anchor": 42, "pct": 41.0},
	})

	t.Run("the shelf is scoped to the caller", func(t *testing.T) {
		shelf := f.shelf(t, f.otherToken, "")
		if len(shelf.Data) != 0 || shelf.Counts.Total != 0 {
			t.Fatalf("another reader's shelf leaked: %+v", shelf)
		}
	})

	// /me/progress/{novel_id} is keyed by the token, so there is no parameter
	// through which one reader could name another.
	t.Run("progress is scoped to the caller", func(t *testing.T) {
		rec := f.env.GETAuth(fmt.Sprintf("/api/v1/me/progress/%d", f.novel.ID), f.otherToken)
		apitest.AssertStatus(t, rec, http.StatusNotFound)
	})

	t.Run("bookmarks are scoped to the caller", func(t *testing.T) {
		f.createBookmark(t, f.ownerToken, 42, "ส่วนตัว")

		rec := f.env.GETAuth("/api/v1/me/bookmarks", f.otherToken)
		body := apitest.DecodeJSON[apitest.List[bookmarkResponse]](t, rec)
		if len(body.Data) != 0 {
			t.Fatalf("another reader's bookmarks leaked: %+v", body.Data)
		}
	})
}

func TestLibraryRoutes_RequireAuthentication(t *testing.T) {
	f := newShelfFixture(t)

	paths := []string{"/api/v1/me/library", "/api/v1/me/bookmarks"}
	for _, path := range paths {
		rec := f.env.GET(path)
		apitest.AssertStatus(t, rec, http.StatusUnauthorized)
	}
}

// R-17 — following a novel is idempotent and keeps the denormalised counter
// honest, because notification fan-out reads that list.
func TestFollows_AreIdempotentAndMaintainTheCounter(t *testing.T) {
	f := newShelfFixture(t)

	followersCount := func() int {
		var n int
		if err := f.env.MakeMe.DB.Model(&entities.Novel{}).
			Where("id = ?", f.novel.ID).
			Pluck("followers_count", &n).Error; err != nil {
			t.Fatalf("read followers_count: %v", err)
		}
		return n
	}
	before := followersCount()

	follow := func(token string) {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   fmt.Sprintf("/api/v1/me/follows/%d", f.novel.ID),
			Token:  token,
		})
		apitest.AssertStatus(t, rec, http.StatusOK)
	}

	follow(f.ownerToken)
	if got := followersCount(); got != before+1 {
		t.Fatalf("followers_count = %d, want %d", got, before+1)
	}

	// A repeat follow must not inflate the counter.
	follow(f.ownerToken)
	if got := followersCount(); got != before+1 {
		t.Fatalf("followers_count = %d after a repeat follow, want %d", got, before+1)
	}

	rec := f.env.GETAuth(fmt.Sprintf("/api/v1/me/follows/%d", f.novel.ID), f.ownerToken)
	apitest.AssertStatus(t, rec, http.StatusOK)
	if !apitest.DecodeJSON[struct {
		Following bool `json:"following"`
	}](t, rec).Following {
		t.Fatal("expected following=true")
	}

	unfollow := func(token string) {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodDelete,
			Path:   fmt.Sprintf("/api/v1/me/follows/%d", f.novel.ID),
			Token:  token,
		})
		apitest.AssertStatus(t, rec, http.StatusOK)
	}

	unfollow(f.ownerToken)
	if got := followersCount(); got != before {
		t.Fatalf("followers_count = %d after unfollow, want %d", got, before)
	}

	// A repeat unfollow must not drive the counter below its starting point.
	unfollow(f.ownerToken)
	if got := followersCount(); got != before {
		t.Fatalf("followers_count = %d after a repeat unfollow, want %d", got, before)
	}
}
