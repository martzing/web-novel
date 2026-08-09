package library_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type seriesFollowResponse struct {
	State     string `json:"state"`
	Total     int    `json:"total"`
	Following int    `json:"following"`
}

// seriesFixture is a reader and a translator's series of three books, one of
// them hidden.
type seriesFixture struct {
	env    *apitest.Env
	token  string
	reader *entities.User
	series *entities.Series
	books  []*entities.Novel
	hidden *entities.Novel
	// baseline is followers_count per novel as built, since the builder seeds a
	// plausible following rather than zero.
	baseline map[int64]int
}

func newSeriesFixture(t *testing.T) *seriesFixture {
	t.Helper()
	env := apitest.New(t)

	reader := env.AUser(entities.RoleReader)
	writer := env.AUser(entities.RoleTranslator)

	series := env.MakeMe.ANewSeries().With(func(s *entities.Series) {
		s.OwnerUserID = &writer.ID
	}).Please()

	book := func(position int, status string) *entities.Novel {
		pos := int16(position)
		return env.MakeMe.ANewNovel().With(func(n *entities.Novel) {
			n.PrimaryTranslatorID = &writer.ID
			n.SeriesID = &series.ID
			n.SeriesPosition = &pos
			n.Status = status
		}).Please()
	}

	first := book(1, entities.NovelOngoing)
	second := book(2, entities.NovelOngoing)
	hidden := book(3, entities.NovelHidden)

	return &seriesFixture{
		env:    env,
		token:  env.TokenFor(reader),
		reader: reader,
		series: series,
		books:  []*entities.Novel{first, second},
		hidden: hidden,
		baseline: map[int64]int{
			first.ID:  first.FollowersCount,
			second.ID: second.FollowersCount,
			hidden.ID: hidden.FollowersCount,
		},
	}
}

func (f *seriesFixture) follow(t *testing.T, method string) seriesFollowResponse {
	t.Helper()
	rec := f.env.Do(apitest.Request{
		Method: method,
		Path:   fmt.Sprintf("/api/v1/series/%d/follow", f.series.ID),
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusOK)
	return apitest.DecodeJSON[seriesFollowResponse](t, rec)
}

func (f *seriesFixture) countFollows(t *testing.T) int64 {
	t.Helper()
	var n int64
	err := f.env.MakeMe.DB.Model(&entities.Follow{}).
		Where("user_id = ?", f.reader.ID).
		Count(&n).Error
	if err != nil {
		t.Fatalf("count follows: %v", err)
	}
	return n
}

// followerDelta reports the change in followers_count since the fixture was
// built. The builder seeds a novel with a plausible following, so the absolute
// number says nothing; what these tests care about is that one follow moves the
// counter by exactly one.
func (f *seriesFixture) followerDelta(t *testing.T, novelID int64) int {
	t.Helper()
	var n entities.Novel
	if err := f.env.MakeMe.DB.Where("id = ?", novelID).Take(&n).Error; err != nil {
		t.Fatalf("load novel: %v", err)
	}
	return n.FollowersCount - f.baseline[novelID]
}

// I-SER-05 — ติดตามทั้งชุด writes a follow row per visible book.
//
// Hidden books are excluded deliberately: following a series must not subscribe
// a reader to a work they cannot open.
func TestFollowSeries_FollowsEveryVisibleBookAndSkipsHiddenOnes(t *testing.T) {
	f := newSeriesFixture(t)

	got := f.follow(t, http.MethodPost)

	if got.State != "all" || got.Total != 2 || got.Following != 2 {
		t.Fatalf("state = %+v, want all 2/2 — the hidden book is not part of the set", got)
	}
	if n := f.countFollows(t); n != 2 {
		t.Fatalf("follow rows = %d, want one per visible book", n)
	}
	for _, book := range f.books {
		if c := f.followerDelta(t, book.ID); c != 1 {
			t.Fatalf("novel %d followers_count moved by %d, want +1", book.ID, c)
		}
	}
	if c := f.followerDelta(t, f.hidden.ID); c != 0 {
		t.Fatalf("hidden novel followers_count moved by %d, want it untouched", c)
	}
}

// I-SER-06 — following twice is a no-op, and the counter does not drift.
//
// The insert returns only the rows Postgres really created, which is what keeps
// followers_count exact when a reader taps twice or already followed one book.
func TestFollowSeries_RepeatingDoesNotDoubleCountFollowers(t *testing.T) {
	f := newSeriesFixture(t)

	f.follow(t, http.MethodPost)
	got := f.follow(t, http.MethodPost)

	if got.State != "all" {
		t.Fatalf("state = %q, want all on a repeat", got.State)
	}
	if n := f.countFollows(t); n != 2 {
		t.Fatalf("follow rows = %d, want the repeat to add nothing", n)
	}
	for _, book := range f.books {
		if c := f.followerDelta(t, book.ID); c != 1 {
			t.Fatalf("novel %d followers_count moved by %d, want +1 after two follow calls", book.ID, c)
		}
	}
}

// I-SER-07 — one book followed on its own reads as partial, not all.
func TestFollowSeries_StateIsPartialWhenOnlySomeBooksAreFollowed(t *testing.T) {
	f := newSeriesFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/me/follows/%d", f.books[0].ID),
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	rec = f.env.GETAuth(fmt.Sprintf("/api/v1/series/%d/follow", f.series.ID), f.token)
	apitest.AssertStatus(t, rec, http.StatusOK)
	got := apitest.DecodeJSON[seriesFollowResponse](t, rec)

	if got.State != "partial" || got.Following != 1 || got.Total != 2 {
		t.Fatalf("state = %+v, want partial 1/2", got)
	}
}

// I-SER-08 — unfollowing a series releases its books and only its books.
func TestUnfollowSeries_LeavesFollowsOnNovelsOutsideTheSeries(t *testing.T) {
	f := newSeriesFixture(t)

	outsider := f.env.MakeMe.ANewNovel().Please()
	f.baseline[outsider.ID] = outsider.FollowersCount

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/api/v1/me/follows/%d", outsider.ID),
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	f.follow(t, http.MethodPost)
	got := f.follow(t, http.MethodDelete)

	if got.State != "none" || got.Following != 0 {
		t.Fatalf("state = %+v, want none after unfollowing the series", got)
	}
	if n := f.countFollows(t); n != 1 {
		t.Fatalf("follow rows = %d, want only the unrelated novel left", n)
	}
	for _, book := range f.books {
		if c := f.followerDelta(t, book.ID); c != 0 {
			t.Fatalf("novel %d followers_count moved by %d, want back to baseline", book.ID, c)
		}
	}
	if c := f.followerDelta(t, outsider.ID); c != 1 {
		t.Fatalf("outsider followers_count moved by %d, want its own follow intact", c)
	}
}

// I-SEC-06 — the series follow routes are caller-scoped like every other
// /me route.
func TestSeriesFollow_RequiresAuthentication(t *testing.T) {
	f := newSeriesFixture(t)

	rec := f.env.GET(fmt.Sprintf("/api/v1/series/%d/follow", f.series.ID))
	apitest.AssertStatus(t, rec, http.StatusUnauthorized)
}
