package catalog_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
	"github.com/mokchan/webnovel-backend/test/makeme"
)

func TestHealthEndpoint(t *testing.T) {
	env := apitest.New(t)

	rec := env.GET("/health")
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[map[string]string](t, rec)
	if body["status"] != "ok" {
		t.Fatalf("status = %q, want ok", body["status"])
	}
}

// The engine must build without panicking under both payment feature-flag
// settings. Gin only detects conflicting route wildcards at registration time,
// so this is the cheapest guard against reintroducing that class of bug.
func TestServerNew_RegistersAllRoutesWithoutPanic(t *testing.T) {
	for _, mockEnabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("PAYMENTS_MOCK_ENABLED=%v", mockEnabled), func(t *testing.T) {
			cfg := apitest.Config()
			cfg.PaymentsMockEnabled = mockEnabled
			env := apitest.NewWith(t, cfg)

			if len(env.Engine.Routes()) == 0 {
				t.Fatal("expected routes to be registered")
			}
		})
	}
}

func TestGenresEndpoint(t *testing.T) {
	env := apitest.New(t)
	m := env.MakeMe

	first := m.ANewGenre().With(func(row *entities.Genre) {
		row.Slug = "xianxia-handler"
		row.NameTH = "เซียน"
	}).Please()
	second := m.ANewGenre().With(func(row *entities.Genre) {
		row.Slug = "wuxia-handler"
		row.NameTH = "กำลังภายใน"
	}).Please()

	rec := env.GET("/api/v1/genres")
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[apitest.List[genreResponse]](t, rec)
	if len(body.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2; body = %s", len(body.Data), rec.Body.String())
	}
	assertGenre(t, body.Data[0], fmt.Sprint(first.ID), "xianxia-handler", "เซียน")
	assertGenre(t, body.Data[1], fmt.Sprint(second.ID), "wuxia-handler", "กำลังภายใน")
}

func TestNovelsEndpoint_SearchGenreAndSort(t *testing.T) {
	env := apitest.New(t)
	m := env.MakeMe

	translator := m.ANewUser().Please()
	xianxia := m.ANewGenre().With(func(row *entities.Genre) {
		row.Slug = "xianxia-handler"
		row.NameTH = "เซียน"
	}).Please()
	mystery := m.ANewGenre().With(func(row *entities.Genre) {
		row.Slug = "mystery-handler"
		row.NameTH = "สืบสวน"
	}).Please()

	oldTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	newTime := oldTime.Add(24 * time.Hour)

	xianxiaNovel := m.ANewNovel().With(func(row *entities.Novel) {
		row.Slug = "mist-sword-handler"
		row.TitleTH = "เซียนดาบหมอกจันทร์"
		row.TitleCN = apitest.Ptr("雾月剑仙")
		row.PrimaryTranslatorID = &translator.ID
		row.FollowersCount = 10
	}).Please()
	mysteryNovel := m.ANewNovel().With(func(row *entities.Novel) {
		row.Slug = "hidden-court-handler"
		row.TitleTH = "ศาลลับใต้เมฆ"
		row.TitleCN = apitest.Ptr("云下密庭")
		row.PrimaryTranslatorID = &translator.ID
		row.FollowersCount = 20
	}).Please()

	m.ANewNovelGenre().With(func(row *entities.NovelGenre) {
		row.NovelID = xianxiaNovel.ID
		row.GenreID = xianxia.ID
	}).Please()
	m.ANewNovelGenre().With(func(row *entities.NovelGenre) {
		row.NovelID = mysteryNovel.ID
		row.GenreID = mystery.ID
	}).Please()

	// autoUpdateTime overwrites UpdatedAt on insert, so set it afterwards.
	setNovelUpdatedAt(t, m, xianxiaNovel.ID, oldTime)
	setNovelUpdatedAt(t, m, mysteryNovel.ID, newTime)

	// I-CAT-01 — Thai has no word boundaries, so "เซียนดาบ" is a substring of
	// the single lexeme "เซียนดาบหมอกจันทร์". A pure tsvector match misses it;
	// the blended trigram/ILIKE ranking is what makes this pass.
	t.Run("I-CAT-01 search by Thai title ranks the match first", func(t *testing.T) {
		rec := env.GET("/api/v1/novels?q=" + url.QueryEscape("เซียนดาบ"))
		apitest.AssertStatus(t, rec, http.StatusOK)

		body := apitest.DecodeJSON[apitest.List[novelResponse]](t, rec)
		if len(body.Data) == 0 {
			t.Fatalf("expected at least one result; body = %s", rec.Body.String())
		}
		if body.Data[0].Slug != xianxiaNovel.Slug {
			t.Fatalf("top result = %q, want %q", body.Data[0].Slug, xianxiaNovel.Slug)
		}
	})

	t.Run("search also matches the Chinese title", func(t *testing.T) {
		rec := env.GET("/api/v1/novels?q=" + url.QueryEscape("雾月"))
		apitest.AssertStatus(t, rec, http.StatusOK)

		body := apitest.DecodeJSON[apitest.List[novelResponse]](t, rec)
		if len(body.Data) != 1 || body.Data[0].Slug != xianxiaNovel.Slug {
			t.Fatalf("expected only the xianxia novel; body = %s", rec.Body.String())
		}
	})

	// I-CAT-02 — only novels linked through novel_genres come back.
	t.Run("I-CAT-02 genre filter", func(t *testing.T) {
		rec := env.GET("/api/v1/novels?genre=xianxia-handler")
		apitest.AssertStatus(t, rec, http.StatusOK)

		body := apitest.DecodeJSON[apitest.List[novelResponse]](t, rec)
		if len(body.Data) != 1 {
			t.Fatalf("len(data) = %d, want 1; body = %s", len(body.Data), rec.Body.String())
		}
		if body.Data[0].Slug != xianxiaNovel.Slug {
			t.Fatalf("slug = %q, want %q", body.Data[0].Slug, xianxiaNovel.Slug)
		}
	})

	t.Run("latest sort", func(t *testing.T) {
		rec := env.GET("/api/v1/novels?sort=latest")
		apitest.AssertStatus(t, rec, http.StatusOK)

		body := apitest.DecodeJSON[apitest.List[novelResponse]](t, rec)
		if len(body.Data) != 2 {
			t.Fatalf("len(data) = %d, want 2; body = %s", len(body.Data), rec.Body.String())
		}
		if body.Data[0].Slug != mysteryNovel.Slug {
			t.Fatalf("first slug = %q, want %q", body.Data[0].Slug, mysteryNovel.Slug)
		}
	})

	t.Run("popular sort orders by followers", func(t *testing.T) {
		rec := env.GET("/api/v1/novels?sort=popular")
		apitest.AssertStatus(t, rec, http.StatusOK)

		body := apitest.DecodeJSON[apitest.List[novelResponse]](t, rec)
		if len(body.Data) != 2 || body.Data[0].Slug != mysteryNovel.Slug {
			t.Fatalf("expected the 20-follower novel first; body = %s", rec.Body.String())
		}
	})

	t.Run("a bad cursor is rejected rather than silently ignored", func(t *testing.T) {
		rec := env.GET("/api/v1/novels?cursor=not-a-cursor")
		apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "BAD_CURSOR")
	})
}

// I-SEC-02 — user input reaches the database only as a bound parameter, and the
// LIKE metacharacters it may contain are escaped rather than acting as
// wildcards.
func TestSearchNovels_SQLInjectionInQueryIsParameterized(t *testing.T) {
	env := apitest.New(t)
	m := env.MakeMe

	novel := m.ANewNovel().With(func(row *entities.Novel) {
		row.Slug = "injection-target-handler"
		row.TitleTH = "นิยายทดสอบการฉีดคำสั่ง"
	}).Please()

	t.Run("injection payloads return no rows and no error", func(t *testing.T) {
		payloads := []string{
			`' OR 1=1--`,
			`'; DROP TABLE novels; --`,
			`" OR ""="`,
			`\'`,
		}
		for _, payload := range payloads {
			rec := env.GET("/api/v1/novels?q=" + url.QueryEscape(payload))
			apitest.AssertStatus(t, rec, http.StatusOK)

			body := apitest.DecodeJSON[apitest.List[novelResponse]](t, rec)
			if len(body.Data) != 0 {
				t.Fatalf("payload %q returned %d rows, want 0", payload, len(body.Data))
			}
		}
	})

	// Without escaping, "%" would match every novel in the table.
	t.Run("LIKE wildcards in the query are literal", func(t *testing.T) {
		for _, payload := range []string{"%", "_", "%%"} {
			rec := env.GET("/api/v1/novels?q=" + url.QueryEscape(payload))
			apitest.AssertStatus(t, rec, http.StatusOK)

			body := apitest.DecodeJSON[apitest.List[novelResponse]](t, rec)
			if len(body.Data) != 0 {
				t.Fatalf("query %q behaved as a wildcard and matched %d rows", payload, len(body.Data))
			}
		}
	})

	t.Run("the table is still intact", func(t *testing.T) {
		rec := env.GET("/api/v1/novels/" + novel.Slug)
		apitest.AssertStatus(t, rec, http.StatusOK)
	})
}

// I-CAT-03 — detail carries arcs and the rating aggregate.
func TestNovelDetailEndpoint(t *testing.T) {
	env := apitest.New(t)
	m := env.MakeMe

	translator := m.ANewUser().Please()
	genre := m.ANewGenre().With(func(row *entities.Genre) {
		row.Slug = "cultivation-handler"
		row.NameTH = "บำเพ็ญ"
	}).Please()
	novel := m.ANewNovel().With(func(row *entities.Novel) {
		row.Slug = "nine-streams-detail-handler"
		row.TitleTH = "เซียนดาบเก้าสายธาร"
		row.PrimaryTranslatorID = &translator.ID
		row.RatingAvg = 4.75
		row.RatingCount = 321
	}).Please()
	m.ANewNovelGenre().With(func(row *entities.NovelGenre) {
		row.NovelID = novel.ID
		row.GenreID = genre.ID
	}).Please()
	m.ANewArc().With(func(row *entities.Arc) {
		row.NovelID = novel.ID
		row.ArcNo = 1
		row.Name = "ธุลีเมืองชายแดน"
		row.FromChapterNo = 1
		row.ToChapterNo = 48
	}).Please()
	m.ANewArc().With(func(row *entities.Arc) {
		row.NovelID = novel.ID
		row.ArcNo = 2
		row.Name = "สำนักเมฆาวสันต์"
		row.FromChapterNo = 49
		row.ToChapterNo = 120
	}).Please()

	assertDetail := func(t *testing.T, body novelDetailResponse) {
		t.Helper()
		if body.Slug != novel.Slug {
			t.Fatalf("slug = %q, want %q", body.Slug, novel.Slug)
		}
		if body.RatingAvg != 4.75 || body.RatingCount != 321 {
			t.Fatalf("rating = %v/%d, want 4.75/321", body.RatingAvg, body.RatingCount)
		}
		if len(body.Arcs) != 2 {
			t.Fatalf("len(arcs) = %d, want 2", len(body.Arcs))
		}
		if body.Arcs[0].ArcNo != 1 || body.Arcs[0].Name != "ธุลีเมืองชายแดน" {
			t.Fatalf("first arc = %+v, want arc 1", body.Arcs[0])
		}
		if len(body.Genres) != 1 || body.Genres[0].Slug != "cultivation-handler" {
			t.Fatalf("genres = %+v, want the linked genre", body.Genres)
		}
	}

	t.Run("by slug", func(t *testing.T) {
		rec := env.GET("/api/v1/novels/" + novel.Slug)
		apitest.AssertStatus(t, rec, http.StatusOK)
		assertDetail(t, apitest.DecodeJSON[novelDetailResponse](t, rec))
	})

	// The route parameter is a single wildcard, so the same path resolves both
	// spellings the API spec uses.
	t.Run("by numeric id", func(t *testing.T) {
		rec := env.GET(fmt.Sprintf("/api/v1/novels/%d", novel.ID))
		apitest.AssertStatus(t, rec, http.StatusOK)
		assertDetail(t, apitest.DecodeJSON[novelDetailResponse](t, rec))
	})
}

func TestNovelDetailEndpoint_NotFound(t *testing.T) {
	env := apitest.New(t)

	for _, path := range []string{"/api/v1/novels/does-not-exist", "/api/v1/novels/999999"} {
		rec := env.GET(path)
		apitest.AssertErrorCode(t, rec, http.StatusNotFound, "NOT_FOUND")
	}
}

func TestNovelArcsEndpoint(t *testing.T) {
	env := apitest.New(t)
	m := env.MakeMe

	novel := m.ANewNovel().Please()
	m.ANewArc().With(func(row *entities.Arc) {
		row.NovelID = novel.ID
		row.ArcNo = 2
		row.Name = "สำนักเมฆาวสันต์"
		row.FromChapterNo = 49
		row.ToChapterNo = 120
	}).Please()
	m.ANewArc().With(func(row *entities.Arc) {
		row.NovelID = novel.ID
		row.ArcNo = 1
		row.Name = "ธุลีเมืองชายแดน"
		row.FromChapterNo = 1
		row.ToChapterNo = 48
	}).Please()

	rec := env.GET(fmt.Sprintf("/api/v1/novels/%d/arcs", novel.ID))
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[apitest.List[arcResponse]](t, rec)
	if len(body.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2", len(body.Data))
	}
	if body.Data[0].ArcNo != 1 || body.Data[1].ArcNo != 2 {
		t.Fatalf("arcs are not ordered by arc_no: %+v", body.Data)
	}
}

func TestNovelGlossaryEndpoint(t *testing.T) {
	env := apitest.New(t)
	m := env.MakeMe

	novel := m.ANewNovel().Please()
	ranks := m.ANewGlossaryGroup().With(func(row *entities.GlossaryGroup) {
		row.NovelID = novel.ID
		row.Name = "ลำดับขั้นการบำเพ็ญ"
		row.SortNo = 1
	}).Please()
	terms := m.ANewGlossaryGroup().With(func(row *entities.GlossaryGroup) {
		row.NovelID = novel.ID
		row.Name = "ศัพท์การบำเพ็ญ"
		row.SortNo = 2
	}).Please()

	m.ANewGlossaryEntry().With(func(row *entities.GlossaryEntry) {
		row.GroupID = ranks.ID
		row.TermKey = "foundation"
		row.TitleTH = "ขั้นหลอมรากฐาน"
		row.TitleCN = apitest.Ptr("筑基")
	}).Please()
	m.ANewGlossaryEntry().With(func(row *entities.GlossaryEntry) {
		row.GroupID = terms.ID
		row.TermKey = "qi"
		row.TitleTH = "ชี่"
		row.TitleCN = apitest.Ptr("气")
	}).Please()

	rec := env.GET(fmt.Sprintf("/api/v1/novels/%d/glossary", novel.ID))
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[apitest.List[glossaryGroupResponse]](t, rec)
	if len(body.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2; body = %s", len(body.Data), rec.Body.String())
	}
	if body.Data[0].Name != "ลำดับขั้นการบำเพ็ญ" {
		t.Fatalf("groups are not ordered by sort_no: %+v", body.Data)
	}
	if len(body.Data[0].Entries) != 1 || body.Data[0].Entries[0].TermKey != "foundation" {
		t.Fatalf("first group entries = %+v", body.Data[0].Entries)
	}
	if body.Data[1].Entries[0].TitleCN != "气" {
		t.Fatalf("expected the Chinese title to round-trip, got %+v", body.Data[1].Entries[0])
	}
}

func TestListChaptersEndpoint(t *testing.T) {
	env := apitest.New(t)
	m := env.MakeMe

	novel := m.ANewNovel().Please()
	published := m.ANewChapter().With(func(row *entities.Chapter) {
		row.NovelID = novel.ID
		row.ChapterNo = 1
		row.Title = "บทที่หนึ่ง"
		row.Status = entities.ChapterPublished
	}).Please()
	paid := m.ANewChapter().With(func(row *entities.Chapter) {
		row.NovelID = novel.ID
		row.ChapterNo = 2
		row.Title = "บทที่สอง"
		row.Status = entities.ChapterPublished
		row.PriceCoins = 5
	}).Please()
	m.ANewChapter().With(func(row *entities.Chapter) {
		row.NovelID = novel.ID
		row.ChapterNo = 3
		row.Status = entities.ChapterDraftStatus
	}).Please()
	m.ANewChapter().With(func(row *entities.Chapter) {
		row.NovelID = novel.ID
		row.ChapterNo = 4
		row.Status = entities.ChapterScheduled
	}).Please()

	t.Run("only published chapters are listed", func(t *testing.T) {
		rec := env.GET(fmt.Sprintf("/api/v1/novels/%d/chapters", novel.ID))
		apitest.AssertStatus(t, rec, http.StatusOK)

		body := apitest.DecodeJSON[apitest.List[chapterListResponse]](t, rec)
		if len(body.Data) != 2 {
			t.Fatalf("len(data) = %d, want 2; body = %s", len(body.Data), rec.Body.String())
		}
		if body.Data[0].ID != fmt.Sprint(published.ID) || body.Data[1].ID != fmt.Sprint(paid.ID) {
			t.Fatalf("unexpected chapter ids: %+v", body.Data)
		}
	})

	t.Run("anonymous readers see paid chapters as locked", func(t *testing.T) {
		rec := env.GET(fmt.Sprintf("/api/v1/novels/%d/chapters", novel.ID))
		body := apitest.DecodeJSON[apitest.List[chapterListResponse]](t, rec)

		if !body.Data[0].Unlocked {
			t.Fatal("a free chapter is always unlocked")
		}
		if body.Data[1].Unlocked {
			t.Fatal("a paid chapter must not be unlocked for an anonymous reader")
		}
	})
}

func TestListChaptersEndpoint_InvalidID(t *testing.T) {
	env := apitest.New(t)

	rec := env.GET("/api/v1/novels/not-a-number/chapters")
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "BAD_ID")
}

// With no snapshot yet, the weekly ranking falls back to live popularity so the
// home page is never empty on a fresh install.
func TestWeeklyRankingEndpoint_FallsBackToPopularity(t *testing.T) {
	env := apitest.New(t)
	m := env.MakeMe

	m.ANewNovel().With(func(row *entities.Novel) {
		row.Slug = "rank-low-handler"
		row.FollowersCount = 5
	}).Please()
	top := m.ANewNovel().With(func(row *entities.Novel) {
		row.Slug = "rank-high-handler"
		row.FollowersCount = 5000
	}).Please()

	rec := env.GET("/api/v1/ranking/weekly?limit=5")
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[apitest.List[rankedNovelResponse]](t, rec)
	if len(body.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2; body = %s", len(body.Data), rec.Body.String())
	}
	if body.Data[0].Slug != top.Slug || body.Data[0].Rank != 1 {
		t.Fatalf("top row = %+v, want %q at rank 1", body.Data[0], top.Slug)
	}
}

func assertGenre(t *testing.T, got genreResponse, id, slug, nameTH string) {
	t.Helper()
	if got.ID != id || got.Slug != slug || got.NameTH != nameTH {
		t.Fatalf("genre = %+v, want id=%s slug=%s name_th=%s", got, id, slug, nameTH)
	}
}

// setNovelUpdatedAt works around GORM's autoUpdateTime, which overwrites the
// column on insert.
func setNovelUpdatedAt(t *testing.T, m *makeme.MakeMe, id int64, updatedAt time.Time) {
	t.Helper()
	if err := m.DB.Model(&entities.Novel{}).
		Where("id = ?", id).
		Update("updated_at", updatedAt).Error; err != nil {
		t.Fatalf("set novel updated_at: %v", err)
	}
}

// Wire-shape structs, declared locally so a handler DTO change that breaks the
// contract shows up as a test failure rather than compiling silently.

type genreResponse struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	NameTH string `json:"name_th"`
}

type novelResponse struct {
	ID             string          `json:"id"`
	Slug           string          `json:"slug"`
	TitleTH        string          `json:"title_th"`
	TitleCN        string          `json:"title_cn"`
	AuthorName     string          `json:"author_name"`
	CoverURL       string          `json:"cover_url"`
	Status         string          `json:"status"`
	RatingAvg      float64         `json:"rating_avg"`
	RatingCount    int             `json:"rating_count"`
	FollowersCount int             `json:"followers_count"`
	ChaptersCount  int             `json:"chapters_count"`
	Genres         []genreResponse `json:"genres"`
}

type novelDetailResponse struct {
	novelResponse
	Description   string        `json:"description"`
	Arcs          []arcResponse `json:"arcs"`
	GlossaryCount int           `json:"glossary_count"`
	CommentsCount int           `json:"comments_count"`
}

type rankedNovelResponse struct {
	novelResponse
	Rank  int     `json:"rank"`
	Score float64 `json:"score"`
}

type arcResponse struct {
	ID            string `json:"id"`
	ArcNo         int    `json:"arc_no"`
	Name          string `json:"name"`
	FromChapterNo int    `json:"from_chapter_no"`
	ToChapterNo   int    `json:"to_chapter_no"`
}

type chapterListResponse struct {
	ID          string `json:"id"`
	ChapterNo   int    `json:"chapter_no"`
	Title       string `json:"title"`
	PriceCoins  int    `json:"price_coins"`
	WordCount   int    `json:"word_count"`
	PublishedAt string `json:"published_at"`
	ArcID       string `json:"arc_id"`
	Unlocked    bool   `json:"unlocked"`
}

type glossaryGroupResponse struct {
	ID      string                  `json:"id"`
	Name    string                  `json:"name"`
	SortNo  int                     `json:"sort_no"`
	Entries []glossaryEntryResponse `json:"entries"`
}

type glossaryEntryResponse struct {
	ID      string `json:"id"`
	TermKey string `json:"term_key"`
	TitleTH string `json:"title_th"`
	TitleCN string `json:"title_cn"`
	Body    string `json:"body"`
	Kind    string `json:"kind"`
}
