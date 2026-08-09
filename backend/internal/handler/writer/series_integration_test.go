package writer_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type seriesResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	BookCount   int    `json:"book_count"`
}

type seriesBookResponse struct {
	NovelID  string `json:"novel_id"`
	Position int    `json:"position"`
	Note     string `json:"note"`
	TitleTH  string `json:"title_th"`
}

type relationResponse struct {
	NovelID        string `json:"novel_id"`
	RelatedNovelID string `json:"related_novel_id"`
	Kind           string `json:"kind"`
	KindLabel      string `json:"kind_label"`
	Note           string `json:"note"`
	Mirrored       bool   `json:"mirrored"`
	TitleTH        string `json:"title_th"`
}

// seriesFixture is a translator with three novels and one empty series.
type seriesFixture struct {
	env    *apitest.Env
	writer *entities.User
	token  string
	novels []*entities.Novel
}

func newSeriesFixture(t *testing.T) *seriesFixture {
	t.Helper()
	env := apitest.New(t)
	writer := env.AUser(entities.RoleTranslator)

	novels := make([]*entities.Novel, 0, 3)
	for i := range 3 {
		n := env.MakeMe.ANewNovel().With(func(n *entities.Novel) {
			n.PrimaryTranslatorID = &writer.ID
			n.ChaptersCount = 10 * (i + 1)
			n.SourceChaptersCount = 100
		}).Please()
		novels = append(novels, n)
	}

	return &seriesFixture{env: env, writer: writer, token: env.TokenFor(writer), novels: novels}
}

func (f *seriesFixture) createSeries(t *testing.T, title string) seriesResponse {
	t.Helper()
	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/series",
		Token:  f.token,
		Body:   map[string]any{"title": title},
	})
	apitest.AssertStatus(t, rec, http.StatusCreated)
	return apitest.DecodeJSON[seriesResponse](t, rec)
}

func (f *seriesFixture) join(t *testing.T, novel *entities.Novel, seriesID string) *httptest.ResponseRecorder {
	t.Helper()
	return f.env.Do(apitest.Request{
		Method: http.MethodPatch,
		Path:   fmt.Sprintf("/api/v1/writer/novels/%d", novel.ID),
		Token:  f.token,
		Body:   map[string]any{"series_id": seriesID},
	})
}

func (f *seriesFixture) books(t *testing.T, seriesID string) []seriesBookResponse {
	t.Helper()
	rec := f.env.GETAuth("/api/v1/writer/series/"+seriesID+"/books", f.token)
	apitest.AssertStatus(t, rec, http.StatusOK)
	return apitest.DecodeJSON[struct {
		Data []seriesBookResponse `json:"data"`
	}](t, rec).Data
}

func TestSeries_CreateListAndDelete(t *testing.T) {
	f := newSeriesFixture(t)

	series := f.createSeries(t, "ตำนานจอมยุทธ์")
	if series.Slug == "" {
		t.Fatal("a series needs a slug; the public page is addressed by it")
	}

	rec := f.env.GETAuth("/api/v1/writer/series", f.token)
	apitest.AssertStatus(t, rec, http.StatusOK)
	list := apitest.DecodeJSON[struct {
		Data []seriesResponse `json:"data"`
	}](t, rec).Data
	if len(list) != 1 || list[0].Title != "ตำนานจอมยุทธ์" {
		t.Fatalf("list = %+v, want the one series", list)
	}

	del := f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   "/api/v1/writer/series/" + series.ID,
		Token:  f.token,
	})
	apitest.AssertStatus(t, del, http.StatusNoContent)
}

// Deleting a collection must never delete the work inside it.
func TestSeries_DeletingLeavesTheNovelsInPlaceAndClearsTheirOrder(t *testing.T) {
	f := newSeriesFixture(t)
	series := f.createSeries(t, "ชุดทดสอบ")

	for _, n := range f.novels {
		apitest.AssertStatus(t, f.join(t, n, series.ID), http.StatusOK)
	}

	del := f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   "/api/v1/writer/series/" + series.ID,
		Token:  f.token,
	})
	apitest.AssertStatus(t, del, http.StatusNoContent)

	for _, n := range f.novels {
		var row entities.Novel
		if err := f.env.MakeMe.DB.Where("id = ?", n.ID).Take(&row).Error; err != nil {
			t.Fatalf("the novel was deleted with its series: %v", err)
		}
		if row.SeriesID != nil {
			t.Fatal("the novel still points at a deleted series")
		}
		if row.SeriesPosition != nil && *row.SeriesPosition != 0 {
			t.Fatalf("series_position = %d, want it cleared with the membership", *row.SeriesPosition)
		}
	}
}

func TestSeries_ReorderRenumbersFromOneAndSurvivesAPartialList(t *testing.T) {
	f := newSeriesFixture(t)
	series := f.createSeries(t, "ชุดเรียงลำดับ")
	for _, n := range f.novels {
		apitest.AssertStatus(t, f.join(t, n, series.ID), http.StatusOK)
	}

	// Reverse the order.
	ids := []string{
		fmt.Sprint(f.novels[2].ID),
		fmt.Sprint(f.novels[1].ID),
		fmt.Sprint(f.novels[0].ID),
	}
	rec := f.env.Do(apitest.Request{
		Method: http.MethodPut,
		Path:   "/api/v1/writer/series/" + series.ID + "/order",
		Token:  f.token,
		Body:   map[string]any{"novel_ids": ids},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	books := f.books(t, series.ID)
	if len(books) != 3 {
		t.Fatalf("books = %d, want 3", len(books))
	}
	for i, b := range books {
		if b.Position != i+1 {
			t.Fatalf("positions = %+v, want a gapless 1..3", books)
		}
	}
	if books[0].NovelID != fmt.Sprint(f.novels[2].ID) {
		t.Fatalf("first book = %s, want the reversed order applied", books[0].NovelID)
	}

	// A stale tab sends only two of the three ids. The omitted novel must keep
	// a position rather than falling out of the reading order entirely.
	partial := f.env.Do(apitest.Request{
		Method: http.MethodPut,
		Path:   "/api/v1/writer/series/" + series.ID + "/order",
		Token:  f.token,
		Body:   map[string]any{"novel_ids": []string{ids[1], ids[0]}},
	})
	apitest.AssertStatus(t, partial, http.StatusOK)

	after := f.books(t, series.ID)
	if len(after) != 3 {
		t.Fatalf("books = %d, want the omitted novel kept", len(after))
	}
	for i, b := range after {
		if b.Position != i+1 {
			t.Fatalf("positions = %+v, want a gapless 1..3 after a partial reorder", after)
		}
	}
}

// Permuting an existing order is the case that breaks a naive single-statement
// renumber: novels_series_position is a partial unique index, so Postgres
// enforces it row by row and 1,2,3 → 2,3,1 collides halfway through.
func TestSeries_RepeatedPermutationsDoNotTripTheUniquePositionIndex(t *testing.T) {
	f := newSeriesFixture(t)
	series := f.createSeries(t, "ชุดสลับลำดับ")
	for _, n := range f.novels {
		apitest.AssertStatus(t, f.join(t, n, series.ID), http.StatusOK)
	}

	rotations := [][]string{
		{fmt.Sprint(f.novels[0].ID), fmt.Sprint(f.novels[1].ID), fmt.Sprint(f.novels[2].ID)},
		{fmt.Sprint(f.novels[1].ID), fmt.Sprint(f.novels[2].ID), fmt.Sprint(f.novels[0].ID)},
		{fmt.Sprint(f.novels[2].ID), fmt.Sprint(f.novels[0].ID), fmt.Sprint(f.novels[1].ID)},
		{fmt.Sprint(f.novels[0].ID), fmt.Sprint(f.novels[2].ID), fmt.Sprint(f.novels[1].ID)},
	}
	for i, order := range rotations {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodPut,
			Path:   "/api/v1/writer/series/" + series.ID + "/order",
			Token:  f.token,
			Body:   map[string]any{"novel_ids": order},
		})
		apitest.AssertStatus(t, rec, http.StatusOK)

		books := f.books(t, series.ID)
		for j, b := range books {
			if b.Position != j+1 || b.NovelID != order[j] {
				t.Fatalf("rotation %d: books = %+v, want %v at 1..3", i, books, order)
			}
		}
	}
}

func TestSeries_NoteIsStoredAgainstTheBook(t *testing.T) {
	f := newSeriesFixture(t)
	series := f.createSeries(t, "ชุดมีโน้ต")
	apitest.AssertStatus(t, f.join(t, f.novels[0], series.ID), http.StatusOK)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/api/v1/writer/novels/%d/series-note", f.novels[0].ID),
		Token:  f.token,
		Body:   map[string]any{"note": "อ่านเล่มนี้ก่อนได้ ไม่สปอยล์"},
	})
	apitest.AssertStatus(t, rec, http.StatusNoContent)

	books := f.books(t, series.ID)
	if len(books) != 1 || books[0].Note != "อ่านเล่มนี้ก่อนได้ ไม่สปอยล์" {
		t.Fatalf("books = %+v, want the note stored", books)
	}
}

// A novel may only be filed under a series the caller owns.
func TestSeries_JoiningAnotherTranslatorsSeriesIsForbidden(t *testing.T) {
	f := newSeriesFixture(t)

	stranger := f.env.AUser(entities.RoleTranslator)
	strangerSeries := f.env.MakeMe.ANewSeries().With(func(s *entities.Series) {
		s.OwnerUserID = &stranger.ID
	}).Please()

	rec := f.join(t, f.novels[0], fmt.Sprint(strangerSeries.ID))
	apitest.AssertStatus(t, rec, http.StatusForbidden)
}

func TestSeries_LeavingASeriesClearsThePositionAndNote(t *testing.T) {
	f := newSeriesFixture(t)
	series := f.createSeries(t, "ชุดที่จะออก")
	apitest.AssertStatus(t, f.join(t, f.novels[0], series.ID), http.StatusOK)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPatch,
		Path:   fmt.Sprintf("/api/v1/writer/novels/%d", f.novels[0].ID),
		Token:  f.token,
		Body:   map[string]any{"series_id": nil},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	var row entities.Novel
	if err := f.env.MakeMe.DB.Where("id = ?", f.novels[0].ID).Take(&row).Error; err != nil {
		t.Fatalf("load novel: %v", err)
	}
	if row.SeriesID != nil || row.SeriesNote != nil {
		t.Fatalf("novel = %+v, want the membership fully cleared", row)
	}
}

// Omitting series_id must leave the membership alone — the settings tab patches
// one field at a time, and a nil id would otherwise read as "remove it".
func TestSeries_OmittingSeriesIDLeavesTheMembershipAlone(t *testing.T) {
	f := newSeriesFixture(t)
	series := f.createSeries(t, "ชุดที่ต้องอยู่")
	apitest.AssertStatus(t, f.join(t, f.novels[0], series.ID), http.StatusOK)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPatch,
		Path:   fmt.Sprintf("/api/v1/writer/novels/%d", f.novels[0].ID),
		Token:  f.token,
		Body:   map[string]any{"title_th": "ชื่อใหม่"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	var row entities.Novel
	if err := f.env.MakeMe.DB.Where("id = ?", f.novels[0].ID).Take(&row).Error; err != nil {
		t.Fatalf("load novel: %v", err)
	}
	if row.SeriesID == nil {
		t.Fatal("a patch that never mentioned series_id removed the novel from its series")
	}
}

func TestRelations_LinkListAndUnlink(t *testing.T) {
	f := newSeriesFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/writer/novels/%d/relations", f.novels[0].ID),
		Token:  f.token,
		Body: map[string]any{
			"related_novel_id": fmt.Sprint(f.novels[1].ID),
			"kind":             "sequel",
			"note":             "อ่านต่อจากเล่มแรก",
		},
	})
	apitest.AssertStatus(t, rec, http.StatusCreated)

	relations := f.relations(t, f.novels[0].ID)
	if len(relations) != 1 {
		t.Fatalf("relations = %+v, want 1", relations)
	}
	if relations[0].Kind != "sequel" || relations[0].KindLabel != "ภาคต่อโดยตรง" {
		t.Fatalf("relation = %+v, want a labelled sequel", relations[0])
	}
	if relations[0].Mirrored {
		t.Fatal("the declaring novel's own relation must not be marked mirrored")
	}

	del := f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path: fmt.Sprintf("/api/v1/writer/novels/%d/relations/%d",
			f.novels[0].ID, f.novels[1].ID),
		Token: f.token,
	})
	apitest.AssertStatus(t, del, http.StatusNoContent)

	if got := f.relations(t, f.novels[0].ID); len(got) != 0 {
		t.Fatalf("relations = %+v, want none after unlinking", got)
	}
}

// A relation declared on one novel appears on the other with the inverse kind,
// so a translator enters it once.
func TestRelations_AppearOnTheFarNovelWithTheInverseKind(t *testing.T) {
	f := newSeriesFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/writer/novels/%d/relations", f.novels[0].ID),
		Token:  f.token,
		Body: map[string]any{
			"related_novel_id": fmt.Sprint(f.novels[1].ID),
			"kind":             "sequel",
		},
	})
	apitest.AssertStatus(t, rec, http.StatusCreated)

	far := f.relations(t, f.novels[1].ID)
	if len(far) != 1 {
		t.Fatalf("relations on the far novel = %+v, want 1", far)
	}
	if far[0].Kind != "prequel" {
		t.Fatalf("kind = %q, want the inverse prequel", far[0].Kind)
	}
	if !far[0].Mirrored {
		t.Fatal("a relation stored on the other novel must be marked mirrored")
	}
}

func TestRelations_SelfLinkIsRejected(t *testing.T) {
	f := newSeriesFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/writer/novels/%d/relations", f.novels[0].ID),
		Token:  f.token,
		Body: map[string]any{
			"related_novel_id": fmt.Sprint(f.novels[0].ID),
			"kind":             "sequel",
		},
	})
	apitest.AssertStatus(t, rec, http.StatusBadRequest)
}

func TestRelations_UnknownKindIsRejected(t *testing.T) {
	f := newSeriesFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/writer/novels/%d/relations", f.novels[0].ID),
		Token:  f.token,
		Body: map[string]any{
			"related_novel_id": fmt.Sprint(f.novels[1].ID),
			"kind":             "fanfic",
		},
	})
	apitest.AssertStatus(t, rec, http.StatusBadRequest)
}

func (f *seriesFixture) relations(t *testing.T, novelID int64) []relationResponse {
	t.Helper()
	rec := f.env.GETAuth(fmt.Sprintf("/api/v1/writer/novels/%d/relations", novelID), f.token)
	apitest.AssertStatus(t, rec, http.StatusOK)
	return apitest.DecodeJSON[struct {
		Data []relationResponse `json:"data"`
	}](t, rec).Data
}
