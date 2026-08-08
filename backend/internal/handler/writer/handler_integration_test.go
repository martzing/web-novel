package writer_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type chapterResponse struct {
	ID          string `json:"id"`
	NovelID     string `json:"novel_id"`
	ArcID       string `json:"arc_id"`
	ChapterNo   int    `json:"chapter_no"`
	Title       string `json:"title"`
	BodySource  string `json:"body_source"`
	BodyHTML    string `json:"body_html"`
	PriceCoins  int    `json:"price_coins"`
	Status      string `json:"status"`
	ScheduledAt string `json:"scheduled_at"`
	PublishedAt string `json:"published_at"`
}

type novelResponse struct {
	ID      string `json:"id"`
	Slug    string `json:"slug"`
	TitleTH string `json:"title_th"`
	Status  string `json:"status"`
}

type statsResponse struct {
	Reads         int     `json:"reads"`
	Followers     int     `json:"followers"`
	CoinsEarned   int     `json:"coins_earned"`
	ReadsTrendPct float64 `json:"reads_trend_pct"`
	Series        []struct {
		Day   string `json:"day"`
		Reads int    `json:"reads"`
	} `json:"series"`
	TopChapters []struct {
		ChapterNo int `json:"chapter_no"`
		Reads     int `json:"reads"`
	} `json:"top_chapters"`
}

type writerFixture struct {
	env      *apitest.Env
	writer   *entities.User
	token    string
	stranger *entities.User
	strToken string
	novel    *entities.Novel
	novelID  string
}

func newWriterFixture(t *testing.T) *writerFixture {
	t.Helper()
	env := apitest.New(t)
	m := env.MakeMe

	writer := env.AUser(entities.RoleTranslator)
	stranger := env.AUser(entities.RoleTranslator)

	novel := m.ANewNovel().With(func(n *entities.Novel) {
		n.PrimaryTranslatorID = &writer.ID
	}).Please()

	return &writerFixture{
		env:      env,
		writer:   writer,
		token:    env.TokenFor(writer),
		stranger: stranger,
		strToken: env.TokenFor(stranger),
		novel:    novel,
		novelID:  fmt.Sprint(novel.ID),
	}
}

func (f *writerFixture) createChapter(t *testing.T, no int, title, source string, price int) chapterResponse {
	t.Helper()
	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/novels/" + f.novelID + "/chapters",
		Token:  f.token,
		Body: map[string]any{
			"chapter_no": no, "title": title, "body_source": source, "price_coins": price,
		},
	})
	apitest.AssertStatus(t, rec, http.StatusCreated)
	return apitest.DecodeJSON[chapterResponse](t, rec)
}

func (f *writerFixture) save(t *testing.T, id string, source string) chapterResponse {
	t.Helper()
	rec := f.env.Do(apitest.Request{
		Method: http.MethodPut,
		Path:   "/api/v1/writer/chapters/" + id,
		Token:  f.token,
		Body:   map[string]any{"title": "บทที่แก้ไข", "body_source": source, "price_coins": 5},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)
	return apitest.DecodeJSON[chapterResponse](t, rec)
}

func TestCreateNovel_GeneratesAUsableSlug(t *testing.T) {
	env := apitest.New(t)
	writer := env.AUser(entities.RoleTranslator)

	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/novels",
		Token:  env.TokenFor(writer),
		Body:   map[string]any{"title_th": "เซียนดาบเก้าสายธาร"},
	})
	apitest.AssertStatus(t, rec, http.StatusCreated)

	body := apitest.DecodeJSON[novelResponse](t, rec)
	if body.Slug == "" {
		t.Fatal("expected a generated slug")
	}
	// The slug must not be all-numeric or /novels/:id would shadow it.
	if _, err := fmt.Sscanf(body.Slug, "%d", new(int64)); err == nil && !strings.Contains(body.Slug, "-") {
		t.Fatalf("slug %q would be shadowed by the id route", body.Slug)
	}

	// It resolves through the public catalog route.
	apitest.AssertStatus(t, env.GET("/api/v1/novels/"+body.Slug), http.StatusOK)
}

func TestWriterRoutes_RequireTheTranslatorRole(t *testing.T) {
	env := apitest.New(t)
	reader := env.AUser()

	rec := env.GETAuth("/api/v1/writer/novels", env.TokenFor(reader))
	apitest.AssertStatus(t, rec, http.StatusForbidden)

	apitest.AssertStatus(t, env.GET("/api/v1/writer/novels"), http.StatusUnauthorized)
}

// W-02 — a chapter's arc is resolved from its number.
func TestCreateChapter_ResolvesArcFromChapterNumber(t *testing.T) {
	f := newWriterFixture(t)
	m := f.env.MakeMe

	arcOne := m.ANewArc().With(func(a *entities.Arc) {
		a.NovelID = f.novel.ID
		a.ArcNo = 1
		a.FromChapterNo = 1
		a.ToChapterNo = 48
	}).Please()
	arcTwo := m.ANewArc().With(func(a *entities.Arc) {
		a.NovelID = f.novel.ID
		a.ArcNo = 2
		a.FromChapterNo = 49
		a.ToChapterNo = 120
	}).Please()

	if got := f.createChapter(t, 10, "บทที่สิบ", "เนื้อหา", 0); got.ArcID != fmt.Sprint(arcOne.ID) {
		t.Fatalf("arc = %q, want the first arc %d", got.ArcID, arcOne.ID)
	}
	if got := f.createChapter(t, 87, "บทที่ 87", "เนื้อหา", 5); got.ArcID != fmt.Sprint(arcTwo.ID) {
		t.Fatalf("arc = %q, want the second arc %d", got.ArcID, arcTwo.ID)
	}
	// A chapter beyond every arc simply has none.
	if got := f.createChapter(t, 500, "บทที่ 500", "เนื้อหา", 5); got.ArcID != "" {
		t.Fatalf("arc = %q, want none", got.ArcID)
	}
}

// I-WR-01 — autosave keeps the last 20 revisions.
func TestAutosave_KeepsLast20Revisions(t *testing.T) {
	f := newWriterFixture(t)
	chapter := f.createChapter(t, 1, "บทแรก", "ฉบับที่ 0", 0)

	for i := 1; i <= 25; i++ {
		f.save(t, chapter.ID, fmt.Sprintf("ฉบับที่ %d", i))
	}

	var count int64
	if err := f.env.MakeMe.DB.Model(&entities.ChapterDraft{}).
		Where("chapter_id = ?", chapter.ID).Count(&count).Error; err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if count != 20 {
		t.Fatalf("revisions = %d, want exactly 20", count)
	}

	// The survivors are the newest ones.
	var newest entities.ChapterDraft
	err := f.env.MakeMe.DB.Where("chapter_id = ?", chapter.ID).
		Order("saved_at DESC, id DESC").Take(&newest).Error
	if err != nil {
		t.Fatalf("load newest revision: %v", err)
	}
	if newest.BodySource != "ฉบับที่ 25" {
		t.Fatalf("newest revision = %q, want the last save", newest.BodySource)
	}

	var oldest entities.ChapterDraft
	err = f.env.MakeMe.DB.Where("chapter_id = ?", chapter.ID).
		Order("saved_at ASC, id ASC").Take(&oldest).Error
	if err != nil {
		t.Fatalf("load oldest revision: %v", err)
	}
	if oldest.BodySource == "ฉบับที่ 1" {
		t.Fatal("the oldest revisions should have been pruned")
	}
}

// I-GLO-01 — publishing renders glossary markers into spans, records the bound
// entries, and stamps the glossary revision it rendered against.
func TestPublishChapter_RendersGlossarySpansAndStampsGlossaryRev(t *testing.T) {
	f := newWriterFixture(t)
	m := f.env.MakeMe

	group := m.ANewGlossaryGroup().With(func(g *entities.GlossaryGroup) {
		g.NovelID = f.novel.ID
	}).Please()
	qi := m.ANewGlossaryEntry().With(func(e *entities.GlossaryEntry) {
		e.GroupID = group.ID
		e.TermKey = "qi"
		e.TitleTH = "ชี่"
	}).Please()
	m.ANewGlossaryEntry().With(func(e *entities.GlossaryEntry) {
		e.GroupID = group.ID
		e.TermKey = "ye"
		e.TitleTH = "เยี่ยหลิงเฟิง"
	}).Please()

	chapter := f.createChapter(t, 87, "ดาบแรกใต้ฟ้าหมอก", "<p>{{ye}} รู้สึกว่า {{qi}} ไหลเวียน และ {{typo}}</p>", 5)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/chapters/" + chapter.ID + "/publish",
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	published := apitest.DecodeJSON[chapterResponse](t, rec)
	if published.Status != "published" {
		t.Fatalf("status = %q, want published", published.Status)
	}

	var body entities.ChapterBody
	if err := f.env.MakeMe.DB.Where("chapter_id = ?", chapter.ID).Take(&body).Error; err != nil {
		t.Fatalf("load chapter body: %v", err)
	}

	for _, want := range []string{`<span data-k="ye">เยี่ยหลิงเฟิง</span>`, `<span data-k="qi">ชี่</span>`} {
		if !strings.Contains(body.BodyHTML, want) {
			t.Fatalf("body_html missing %s:\n%s", want, body.BodyHTML)
		}
	}
	// An unknown marker survives verbatim rather than deleting the writer's text.
	if !strings.Contains(body.BodyHTML, "{{typo}}") {
		t.Fatalf("an unknown marker must be preserved:\n%s", body.BodyHTML)
	}

	var novel entities.Novel
	if err := f.env.MakeMe.DB.Where("id = ?", f.novel.ID).Take(&novel).Error; err != nil {
		t.Fatalf("load novel: %v", err)
	}
	if body.GlossaryRev != novel.GlossaryRev {
		t.Fatalf("chapter glossary_rev = %d, want the novel's %d", body.GlossaryRev, novel.GlossaryRev)
	}

	// The bound entries are recorded for the re-render worker.
	var refs []entities.ChapterGlossaryRef
	if err := f.env.MakeMe.DB.Where("chapter_id = ?", chapter.ID).Find(&refs).Error; err != nil {
		t.Fatalf("load glossary refs: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("glossary refs = %d, want 2", len(refs))
	}
	found := false
	for _, ref := range refs {
		if ref.EntryID == qi.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the bound qi entry to be recorded")
	}
}

// I-WR-02 — a chapter scheduled for the future stays invisible to readers.
func TestPublish_FutureScheduledAtHiddenFromReadersUntilTime(t *testing.T) {
	f := newWriterFixture(t)
	chapter := f.createChapter(t, 90, "ตั้งเวลาไว้", "<p>เนื้อหา</p>", 0)

	future := time.Now().Add(24 * time.Hour)
	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/chapters/" + chapter.ID + "/publish",
		Token:  f.token,
		Body:   map[string]any{"scheduled_at": future.Format(time.RFC3339)},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	scheduled := apitest.DecodeJSON[chapterResponse](t, rec)
	if scheduled.Status != "scheduled" {
		t.Fatalf("status = %q, want scheduled", scheduled.Status)
	}
	if scheduled.PublishedAt != "" {
		t.Fatalf("published_at = %q, want empty until it goes live", scheduled.PublishedAt)
	}

	// Readers cannot see it.
	apitest.AssertStatus(t, f.env.GET("/api/v1/chapters/"+chapter.ID), http.StatusNotFound)

	rec = f.env.GET("/api/v1/novels/" + f.novelID + "/chapters")
	apitest.AssertStatus(t, rec, http.StatusOK)
	toc := apitest.DecodeJSON[apitest.List[struct {
		ID string `json:"id"`
	}]](t, rec)
	if len(toc.Data) != 0 {
		t.Fatalf("table of contents = %+v, want a scheduled chapter excluded", toc.Data)
	}

	// Publishing without a schedule makes it live immediately.
	rec = f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/chapters/" + chapter.ID + "/publish",
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusOK)
	if got := apitest.DecodeJSON[chapterResponse](t, rec); got.Status != "published" {
		t.Fatalf("status = %q, want published", got.Status)
	}
	apitest.AssertStatus(t, f.env.GET("/api/v1/chapters/"+chapter.ID), http.StatusOK)
}

func TestUnpublish_ReturnsAChapterToDraft(t *testing.T) {
	f := newWriterFixture(t)
	chapter := f.createChapter(t, 1, "เผยแพร่แล้ว", "<p>เนื้อหา</p>", 0)

	f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/chapters/" + chapter.ID + "/publish",
		Token:  f.token,
	})
	apitest.AssertStatus(t, f.env.GET("/api/v1/chapters/"+chapter.ID), http.StatusOK)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/chapters/" + chapter.ID + "/unpublish",
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	apitest.AssertStatus(t, f.env.GET("/api/v1/chapters/"+chapter.ID), http.StatusNotFound)
}

// Publishing keeps novels.chapters_count honest rather than incrementing it.
func TestPublish_RefreshesTheChapterCount(t *testing.T) {
	f := newWriterFixture(t)
	chapter := f.createChapter(t, 1, "บทแรก", "<p>เนื้อหา</p>", 0)

	publish := func() {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   "/api/v1/writer/chapters/" + chapter.ID + "/publish",
			Token:  f.token,
		})
		apitest.AssertStatus(t, rec, http.StatusOK)
	}

	publish()
	publish() // republishing must not double-count

	var novel entities.Novel
	if err := f.env.MakeMe.DB.Where("id = ?", f.novel.ID).Take(&novel).Error; err != nil {
		t.Fatalf("load novel: %v", err)
	}
	if novel.ChaptersCount != 1 {
		t.Fatalf("chapters_count = %d, want 1", novel.ChaptersCount)
	}
}

// I-WR-03 — one translator cannot touch another's work.
func TestWriterA_CannotEditWriterBsChapter(t *testing.T) {
	f := newWriterFixture(t)
	chapter := f.createChapter(t, 1, "ของนักแปล A", "<p>เนื้อหา</p>", 0)

	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"read the chapter", http.MethodGet, "/api/v1/writer/chapters/" + chapter.ID, nil},
		{"save the chapter", http.MethodPut, "/api/v1/writer/chapters/" + chapter.ID, map[string]any{"title": "แก้ไข", "body_source": "x"}},
		{"publish the chapter", http.MethodPost, "/api/v1/writer/chapters/" + chapter.ID + "/publish", nil},
		{"unpublish the chapter", http.MethodPost, "/api/v1/writer/chapters/" + chapter.ID + "/unpublish", nil},
		{"list the novel's chapters", http.MethodGet, "/api/v1/writer/novels/" + f.novelID + "/chapters", nil},
		{"create a chapter", http.MethodPost, "/api/v1/writer/novels/" + f.novelID + "/chapters", map[string]any{"chapter_no": 9, "title": "แทรก"}},
		{"patch the novel", http.MethodPatch, "/api/v1/writer/novels/" + f.novelID, map[string]any{"title_th": "ยึดครอง"}},
		{"read its stats", http.MethodGet, "/api/v1/writer/stats/novels/" + f.novelID, nil},
		{"read its glossary", http.MethodGet, "/api/v1/writer/novels/" + f.novelID + "/glossary", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := f.env.Do(apitest.Request{
				Method: tc.method,
				Path:   tc.path,
				Token:  f.strToken, // a translator, but not this novel's
				Body:   tc.body,
			})
			apitest.AssertStatus(t, rec, http.StatusForbidden)
		})
	}

	// And nothing changed.
	var row entities.Chapter
	if err := f.env.MakeMe.DB.Where("id = ?", chapter.ID).Take(&row).Error; err != nil {
		t.Fatalf("load chapter: %v", err)
	}
	if row.Title != "ของนักแปล A" {
		t.Fatalf("title = %q, want it untouched", row.Title)
	}
}

// I-WR-04 — the KPI totals equal the sum of the daily rollup rows.
func TestWriterStats_TotalsMatchChapterDailyStatsFixture(t *testing.T) {
	f := newWriterFixture(t)
	m := f.env.MakeMe

	chapter := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = f.novel.ID
		c.ChapterNo = 87
		c.Status = entities.ChapterPublished
	}).Please()

	today := time.Now().UTC()
	days := []struct {
		offset      int
		reads       int
		coins       int
		followers   int
		chapterRead int
	}{
		{-1, 100, 20, 3, 100},
		{-2, 150, 30, 2, 150},
		{-3, 250, 50, 5, 250},
	}
	for _, d := range days {
		day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, d.offset)
		m.ANewNovelDailyStat().With(func(s *entities.NovelDailyStat) {
			s.NovelID = f.novel.ID
			s.Day = day
			s.Reads = d.reads
			s.CoinsEarned = d.coins
			s.FollowersGained = d.followers
		}).Please()
		m.ANewChapterDailyStat().With(func(s *entities.ChapterDailyStat) {
			s.ChapterID = chapter.ID
			s.Day = day
			s.Reads = d.chapterRead
			s.CoinsEarned = d.coins
		}).Please()
	}

	rec := f.env.GETAuth("/api/v1/writer/stats/novels/"+f.novelID+"?period=14d", f.token)
	apitest.AssertStatus(t, rec, http.StatusOK)

	stats := apitest.DecodeJSON[statsResponse](t, rec)
	if stats.Reads != 500 {
		t.Fatalf("reads = %d, want the fixture sum 500", stats.Reads)
	}
	if stats.CoinsEarned != 100 {
		t.Fatalf("coins = %d, want 100", stats.CoinsEarned)
	}
	if stats.Followers != 10 {
		t.Fatalf("followers = %d, want 10", stats.Followers)
	}
	if len(stats.Series) != 3 {
		t.Fatalf("series has %d points, want 3", len(stats.Series))
	}
	if len(stats.TopChapters) != 1 || stats.TopChapters[0].Reads != 500 {
		t.Fatalf("top chapters = %+v, want the one chapter with 500 reads", stats.TopChapters)
	}
}

func TestGlossary_CreateGroupThenEntry(t *testing.T) {
	f := newWriterFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/novels/" + f.novelID + "/glossary",
		Token:  f.token,
		Body:   map[string]any{"name": "ศัพท์การบำเพ็ญ", "sort_no": 1},
	})
	apitest.AssertStatus(t, rec, http.StatusCreated)
	group := apitest.DecodeJSON[struct {
		ID string `json:"id"`
	}](t, rec)

	rec = f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/novels/" + f.novelID + "/glossary",
		Token:  f.token,
		Body: map[string]any{
			"group_id": group.ID,
			"term_key": "qi",
			"title_th": "ชี่",
			"title_cn": "气",
			"body":     "พลังชีวิตที่ไหลเวียนในร่าง",
		},
	})
	apitest.AssertStatus(t, rec, http.StatusCreated)

	// It shows up on the public glossary endpoint too.
	rec = f.env.GET("/api/v1/novels/" + f.novelID + "/glossary")
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[apitest.List[struct {
		Name    string `json:"name"`
		Entries []struct {
			TermKey string `json:"term_key"`
			TitleTH string `json:"title_th"`
		} `json:"entries"`
	}]](t, rec)
	if len(body.Data) != 1 || len(body.Data[0].Entries) != 1 {
		t.Fatalf("glossary = %+v, want one group with one entry", body.Data)
	}
	if body.Data[0].Entries[0].TermKey != "qi" {
		t.Fatalf("entry = %+v, want the qi term", body.Data[0].Entries[0])
	}
}

// Editing a term bumps the novel's glossary_rev via the database trigger, which
// is what makes the re-render worker pick the chapter up.
func TestUpdateGlossaryEntry_BumpsNovelGlossaryRev(t *testing.T) {
	f := newWriterFixture(t)
	m := f.env.MakeMe

	group := m.ANewGlossaryGroup().With(func(g *entities.GlossaryGroup) {
		g.NovelID = f.novel.ID
	}).Please()
	entry := m.ANewGlossaryEntry().With(func(e *entities.GlossaryEntry) {
		e.GroupID = group.ID
		e.TermKey = "qi"
		e.TitleTH = "ชี่"
	}).Please()

	revBefore := func() int {
		var n entities.Novel
		if err := f.env.MakeMe.DB.Where("id = ?", f.novel.ID).Take(&n).Error; err != nil {
			t.Fatalf("load novel: %v", err)
		}
		return n.GlossaryRev
	}
	before := revBefore()

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPatch,
		Path:   fmt.Sprintf("/api/v1/writer/glossary-entries/%d", entry.ID),
		Token:  f.token,
		Body:   map[string]any{"title_th": "ชี่ (แก้ไข)"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	if after := revBefore(); after <= before {
		t.Fatalf("glossary_rev = %d, want it bumped above %d", after, before)
	}
}

func TestCreateChapter_RejectsDuplicateChapterNumbers(t *testing.T) {
	f := newWriterFixture(t)
	f.createChapter(t, 5, "บทที่ห้า", "เนื้อหา", 0)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/novels/" + f.novelID + "/chapters",
		Token:  f.token,
		Body:   map[string]any{"chapter_no": 5, "title": "ซ้ำ", "body_source": "x"},
	})
	apitest.AssertErrorCode(t, rec, http.StatusConflict, "CHAPTER_NO_TAKEN")
}

// W-07 — a price of zero means free; a negative price is rejected.
func TestChapterPrice(t *testing.T) {
	f := newWriterFixture(t)

	free := f.createChapter(t, 1, "อ่านฟรี", "เนื้อหา", 0)
	if free.PriceCoins != 0 {
		t.Fatalf("price = %d, want 0", free.PriceCoins)
	}

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/novels/" + f.novelID + "/chapters",
		Token:  f.token,
		Body:   map[string]any{"chapter_no": 2, "title": "ติดลบ", "price_coins": -5},
	})
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "INVALID_PRICE")
}
