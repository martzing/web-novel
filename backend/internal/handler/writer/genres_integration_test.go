package writer_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

// novelGenresResponse reads only what these tests assert on.
//
// genre_ids are JSON *numbers*: `encoding/json`'s ",string" option is silently
// ignored on slices, so this is the one id in the API that is not a string.
type novelGenresResponse struct {
	ID       string  `json:"id"`
	GenreIDs []int64 `json:"genre_ids"`
}

func sortedIDs(ids []int64) []int64 {
	out := append([]int64(nil), ids...)
	slices.Sort(out)
	return out
}

// I-WR-09 — the works tree returns each novel's genres.
//
// Regression test. ListNovels built its rows without touching novel_genres, so
// GET /writer/novels always answered `genre_ids: []` no matter what the novel
// actually had. The editor seeds its chips from this endpoint, so every work
// looked as though it had no genres — and a save that included the field would
// replace the real set with an empty one.
func TestListNovels_CarriesGenreIDsForEveryNovel(t *testing.T) {
	f := newWriterFixture(t)
	m := f.env.MakeMe

	first := m.ANewGenre().Please()
	second := m.ANewGenre().Please()

	for _, g := range []*entities.Genre{first, second} {
		m.ANewNovelGenre().With(func(link *entities.NovelGenre) {
			link.NovelID = f.novel.ID
			link.GenreID = g.ID
		}).Please()
	}

	// A second novel with no genres at all, to prove the join does not leak
	// one novel's genres onto another.
	bare := m.ANewNovel().With(func(n *entities.Novel) {
		n.PrimaryTranslatorID = &f.writer.ID
	}).Please()

	rec := f.env.GETAuth("/api/v1/writer/novels", f.token)
	apitest.AssertStatus(t, rec, http.StatusOK)
	list := apitest.DecodeJSON[apitest.List[novelGenresResponse]](t, rec)

	var withGenres, without *novelGenresResponse
	for i := range list.Data {
		switch list.Data[i].ID {
		case f.novelID:
			withGenres = &list.Data[i]
		case fmt.Sprint(bare.ID):
			without = &list.Data[i]
		}
	}
	if withGenres == nil || without == nil {
		t.Fatalf("both novels should be listed, got %+v", list.Data)
	}

	want := sortedIDs([]int64{first.ID, second.ID})
	got := sortedIDs(withGenres.GenreIDs)
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("genre_ids = %v, want %v — the list must carry them, not just GET by id", got, want)
	}
	if len(without.GenreIDs) != 0 {
		t.Fatalf("a novel with no genres has %v, want none", without.GenreIDs)
	}
}

// I-WR-10 — a client can send back the genre_ids it was just given.
//
// The round trip is the assertion that matters: response and request must agree
// on the JSON type. They did not, and checking either direction alone is what
// let the disagreement through — the editor read numbers, mixed in strings from
// /genres, and every save failed with "cannot unmarshal string into Go struct
// field novelRequest.genre_ids of type int64".
func TestUpdateNovel_AcceptsTheGenreIDsItJustReturned(t *testing.T) {
	f := newWriterFixture(t)
	m := f.env.MakeMe

	first := m.ANewGenre().Please()
	second := m.ANewGenre().Please()

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPatch,
		Path:   "/api/v1/writer/novels/" + f.novelID,
		Token:  f.token,
		Body: map[string]any{
			"title_th":  "คัมภีร์วิถีเซียน",
			"genre_ids": []int64{first.ID, second.ID},
		},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	saved := apitest.DecodeJSON[novelGenresResponse](t, rec)
	if len(saved.GenreIDs) != 2 {
		t.Fatalf("genre_ids = %v, want the 2 just sent", saved.GenreIDs)
	}

	// Feed the response straight back, which is what the editor does when a
	// translator opens the tab and saves.
	rec = f.env.Do(apitest.Request{
		Method: http.MethodPatch,
		Path:   "/api/v1/writer/novels/" + f.novelID,
		Token:  f.token,
		Body:   map[string]any{"genre_ids": saved.GenreIDs},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	again := apitest.DecodeJSON[novelGenresResponse](t, rec)
	if len(again.GenreIDs) != 2 {
		t.Fatalf("genre_ids = %v after a round trip, want the same 2", again.GenreIDs)
	}
}

// I-WR-11 — omitting genre_ids leaves the novel's genres alone.
//
// This is what lets the ข้อมูลเรื่อง tab save a title without disturbing
// genres the translator never touched. An empty array still means "remove them
// all", which is why the client must omit the key rather than send [].
func TestUpdateNovel_OmittedGenreIDsLeaveTheGenresAlone(t *testing.T) {
	f := newWriterFixture(t)
	m := f.env.MakeMe

	genre := m.ANewGenre().Please()
	m.ANewNovelGenre().With(func(link *entities.NovelGenre) {
		link.NovelID = f.novel.ID
		link.GenreID = genre.ID
	}).Please()

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPatch,
		Path:   "/api/v1/writer/novels/" + f.novelID,
		Token:  f.token,
		Body:   map[string]any{"title_th": "ชื่อใหม่"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	got := apitest.DecodeJSON[novelGenresResponse](t, rec)
	if len(got.GenreIDs) != 1 || got.GenreIDs[0] != genre.ID {
		t.Fatalf("genre_ids = %v, want the untouched %d", got.GenreIDs, genre.ID)
	}

	// An explicit empty array is a real edit and does clear them.
	rec = f.env.Do(apitest.Request{
		Method: http.MethodPatch,
		Path:   "/api/v1/writer/novels/" + f.novelID,
		Token:  f.token,
		Body:   map[string]any{"genre_ids": []int64{}},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	cleared := apitest.DecodeJSON[novelGenresResponse](t, rec)
	if len(cleared.GenreIDs) != 0 {
		t.Fatalf("genre_ids = %v, want an explicit [] to clear them", cleared.GenreIDs)
	}
}
