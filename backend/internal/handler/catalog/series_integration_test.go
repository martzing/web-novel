package catalog_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type publicNovel struct {
	ID                  string `json:"id"`
	Slug                string `json:"slug"`
	TitleTH             string `json:"title_th"`
	Status              string `json:"status"`
	ChaptersCount       int    `json:"chapters_count"`
	SourceChaptersCount int    `json:"source_chapters_count"`
	CoverStyle          string `json:"cover_style"`
	CoverColor          string `json:"cover_color"`
	CoverText           string `json:"cover_text"`
}

type publicSeries struct {
	ID                  string `json:"id"`
	Slug                string `json:"slug"`
	Title               string `json:"title"`
	Description         string `json:"description"`
	ChaptersCount       int    `json:"chapters_count"`
	SourceChaptersCount int    `json:"source_chapters_count"`
	Books               []struct {
		publicNovel
		Position int    `json:"position"`
		Note     string `json:"note"`
	} `json:"books"`
}

type publicRelation struct {
	publicNovel
	Kind      string `json:"kind"`
	KindLabel string `json:"kind_label"`
	Note      string `json:"note"`
}

// hiddenFixture is a translator with one visible and one hidden novel.
type hiddenFixture struct {
	env     *apitest.Env
	writer  *entities.User
	token   string
	visible *entities.Novel
	hidden  *entities.Novel
}

func newHiddenFixture(t *testing.T) *hiddenFixture {
	t.Helper()
	env := apitest.New(t)
	writer := env.AUser(entities.RoleTranslator)

	visible := env.MakeMe.ANewNovel().With(func(n *entities.Novel) {
		n.PrimaryTranslatorID = &writer.ID
		n.Status = entities.NovelOngoing
		n.TitleTH = "เรื่องที่เปิดอยู่"
	}).Please()

	hidden := env.MakeMe.ANewNovel().With(func(n *entities.Novel) {
		n.PrimaryTranslatorID = &writer.ID
		n.Status = entities.NovelHidden
		n.TitleTH = "เรื่องที่ซ่อนอยู่"
	}).Please()

	return &hiddenFixture{
		env: env, writer: writer, token: env.TokenFor(writer),
		visible: visible, hidden: hidden,
	}
}

func (f *hiddenFixture) listSlugs(t *testing.T, path string) map[string]bool {
	t.Helper()
	rec := f.env.GET(path)
	apitest.AssertStatus(t, rec, http.StatusOK)
	body := apitest.DecodeJSON[struct {
		Data []publicNovel `json:"data"`
	}](t, rec)

	out := map[string]bool{}
	for _, n := range body.Data {
		out[n.Slug] = true
	}
	return out
}

// ซ่อนจากหน้าร้าน has to mean something: a hidden novel must be absent from
// every reader-facing surface, not merely labelled.
func TestHidden_IsExcludedFromBrowseSearchAndRanking(t *testing.T) {
	f := newHiddenFixture(t)

	for _, path := range []string{
		"/api/v1/novels",
		"/api/v1/novels?sort=latest",
		"/api/v1/search?q=" + "เรื่องที่",
		"/api/v1/ranking/weekly",
	} {
		t.Run(path, func(t *testing.T) {
			slugs := f.listSlugs(t, path)
			if slugs[f.hidden.Slug] {
				t.Fatalf("%s still lists the hidden novel", path)
			}
			if !slugs[f.visible.Slug] {
				t.Fatalf("%s dropped the visible novel too", path)
			}
		})
	}
}

func TestHidden_DetailIs404ForReadersButOpenToItsTranslator(t *testing.T) {
	f := newHiddenFixture(t)

	anon := f.env.GET("/api/v1/novels/" + f.hidden.Slug)
	apitest.AssertStatus(t, anon, http.StatusNotFound)

	reader := f.env.AUser()
	other := f.env.GETAuth("/api/v1/novels/"+f.hidden.Slug, f.env.TokenFor(reader))
	apitest.AssertStatus(t, other, http.StatusNotFound)

	// The translator still needs to reach it — the works screen opens hidden
	// novels for editing.
	own := f.env.GETAuth("/api/v1/novels/"+f.hidden.Slug, f.token)
	apitest.AssertStatus(t, own, http.StatusOK)
}

func TestNovelDetail_CarriesBothChapterCountsAndTheCoverTemplate(t *testing.T) {
	env := apitest.New(t)
	novel := env.MakeMe.ANewNovel().With(func(n *entities.Novel) {
		n.ChaptersCount = 87
		n.SourceChaptersCount = 412
		n.CoverStyle = entities.CoverSeal
		colour := "#8B1E2D"
		text := "จอมดาบ"
		n.CoverColor = &colour
		n.CoverText = &text
	}).Please()

	rec := env.GET("/api/v1/novels/" + novel.Slug)
	apitest.AssertStatus(t, rec, http.StatusOK)
	got := apitest.DecodeJSON[publicNovel](t, rec)

	if got.ChaptersCount != 87 || got.SourceChaptersCount != 412 {
		t.Fatalf("counts = %d/%d, want 87 translated of 412", got.ChaptersCount, got.SourceChaptersCount)
	}
	if got.CoverStyle != entities.CoverSeal || got.CoverColor != "#8B1E2D" || got.CoverText != "จอมดาบ" {
		t.Fatalf("cover = %+v, want the stored template", got)
	}
}

// A novel with no cover_style stored still names a renderer, so the client
// never has to guess.
func TestNovelDetail_DefaultsCoverStyleToImage(t *testing.T) {
	env := apitest.New(t)
	novel := env.MakeMe.ANewNovel().Please()

	rec := env.GET("/api/v1/novels/" + novel.Slug)
	apitest.AssertStatus(t, rec, http.StatusOK)
	if got := apitest.DecodeJSON[publicNovel](t, rec); got.CoverStyle != entities.CoverImage {
		t.Fatalf("cover style = %q, want image", got.CoverStyle)
	}
}

// seriesPageFixture is a public series of two books plus one hidden one.
func TestPublicSeries_ReturnsBooksInReadingOrderWithSummedCounts(t *testing.T) {
	env := apitest.New(t)
	writer := env.AUser(entities.RoleTranslator)

	series := env.MakeMe.ANewSeries().With(func(s *entities.Series) {
		s.OwnerUserID = &writer.ID
		s.Title = "มหากาพย์จอมดาบ"
	}).Please()

	newBook := func(position int, chapters, source int, note, status string) *entities.Novel {
		pos := int16(position)
		n := note
		return env.MakeMe.ANewNovel().With(func(v *entities.Novel) {
			v.PrimaryTranslatorID = &writer.ID
			v.SeriesID = &series.ID
			v.SeriesPosition = &pos
			v.SeriesNote = &n
			v.ChaptersCount = chapters
			v.SourceChaptersCount = source
			v.Status = status
		}).Please()
	}

	second := newBook(2, 30, 200, "อ่านต่อจากเล่มแรก", entities.NovelOngoing)
	first := newBook(1, 87, 412, "เริ่มที่เล่มนี้", entities.NovelComplete)
	newBook(3, 5, 50, "ยังไม่เปิด", entities.NovelHidden)

	rec := env.GET(fmt.Sprintf("/api/v1/series/%d", series.ID))
	apitest.AssertStatus(t, rec, http.StatusOK)
	got := apitest.DecodeJSON[publicSeries](t, rec)

	if len(got.Books) != 2 {
		t.Fatalf("books = %d, want the hidden one excluded", len(got.Books))
	}
	if got.Books[0].ID != fmt.Sprint(first.ID) || got.Books[1].ID != fmt.Sprint(second.ID) {
		t.Fatalf("books = %+v, want reading order, not insertion order", got.Books)
	}
	if got.Books[0].Note != "เริ่มที่เล่มนี้" {
		t.Fatalf("note = %q, want the translator's line", got.Books[0].Note)
	}
	// The header stats sum only the books a reader can actually see.
	if got.ChaptersCount != 117 || got.SourceChaptersCount != 612 {
		t.Fatalf("counts = %d/%d, want 117 of 612 across the two visible books",
			got.ChaptersCount, got.SourceChaptersCount)
	}
}

func TestPublicSeries_ResolvesBySlugAndReturns404WhenMissing(t *testing.T) {
	env := apitest.New(t)
	series := env.MakeMe.ANewSeries().Please()

	bySlug := env.GET("/api/v1/series/" + series.Slug)
	apitest.AssertStatus(t, bySlug, http.StatusOK)

	missing := env.GET("/api/v1/series/no-such-series")
	apitest.AssertStatus(t, missing, http.StatusNotFound)
}

func TestPublicRelated_GroupsBothDirectionsWithLabels(t *testing.T) {
	env := apitest.New(t)
	writer := env.AUser(entities.RoleTranslator)

	newNovel := func(title string) *entities.Novel {
		return env.MakeMe.ANewNovel().With(func(n *entities.Novel) {
			n.PrimaryTranslatorID = &writer.ID
			n.TitleTH = title
		}).Please()
	}
	base := newNovel("ปฐมบท")
	sequel := newNovel("ภาคต่อ")
	hidden := newNovel("ที่ซ่อน")
	env.MakeMe.DB.Model(&entities.Novel{}).
		Where("id = ?", hidden.ID).
		Update("status", entities.NovelHidden)

	note := "อ่านต่อได้เลย"
	env.MakeMe.ANewNovelRelation().With(func(r *entities.NovelRelation) {
		r.NovelID = base.ID
		r.RelatedNovelID = sequel.ID
		r.Kind = entities.RelationSequel
		r.Note = &note
	}).Please()
	env.MakeMe.ANewNovelRelation().With(func(r *entities.NovelRelation) {
		r.NovelID = base.ID
		r.RelatedNovelID = hidden.ID
		r.Kind = entities.RelationSpinoff
	}).Please()

	forward := relatedFor(t, env, base.ID)
	if len(forward) != 1 {
		t.Fatalf("related = %+v, want the hidden novel excluded", forward)
	}
	if forward[0].Kind != entities.RelationSequel || forward[0].KindLabel != "ภาคต่อโดยตรง" {
		t.Fatalf("relation = %+v, want a labelled sequel", forward[0])
	}

	// The far novel sees the same link inverted, without a second row.
	backward := relatedFor(t, env, sequel.ID)
	if len(backward) != 1 || backward[0].Kind != entities.RelationPrequel {
		t.Fatalf("related = %+v, want an inverted prequel", backward)
	}
}

func relatedFor(t *testing.T, env *apitest.Env, novelID int64) []publicRelation {
	t.Helper()
	rec := env.GET(fmt.Sprintf("/api/v1/novels/%d/related", novelID))
	apitest.AssertStatus(t, rec, http.StatusOK)
	return apitest.DecodeJSON[struct {
		Data []publicRelation `json:"data"`
	}](t, rec).Data
}
