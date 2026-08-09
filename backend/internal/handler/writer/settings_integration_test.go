package writer_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type novelSettingsResponse struct {
	ID                  string `json:"id"`
	Status              string `json:"status"`
	ChaptersCount       int    `json:"chapters_count"`
	SourceChaptersCount int    `json:"source_chapters_count"`
	PricePerChapter     int    `json:"price_per_chapter"`
	FreeUntilChapter    int    `json:"free_until_chapter"`
	SellByArc           bool   `json:"sell_by_arc"`
	TipsEnabled         bool   `json:"tips_enabled"`
	EarlyAccessHours    int    `json:"early_access_hours"`
	ReleaseSchedule     string `json:"release_schedule"`
	CoverStyle          string `json:"cover_style"`
	CoverColor          string `json:"cover_color"`
	CoverText           string `json:"cover_text"`
}

type settingsFixture struct {
	env    *apitest.Env
	writer *entities.User
	token  string
	novel  *entities.Novel
}

func newSettingsFixture(t *testing.T) *settingsFixture {
	t.Helper()
	env := apitest.New(t)
	writer := env.AUser(entities.RoleTranslator)
	novel := env.MakeMe.ANewNovel().With(func(n *entities.Novel) {
		n.PrimaryTranslatorID = &writer.ID
	}).Please()

	return &settingsFixture{env: env, writer: writer, token: env.TokenFor(writer), novel: novel}
}

func (f *settingsFixture) patch(t *testing.T, body map[string]any) novelSettingsResponse {
	t.Helper()
	rec := f.env.Do(apitest.Request{
		Method: http.MethodPatch,
		Path:   fmt.Sprintf("/api/v1/writer/novels/%d", f.novel.ID),
		Token:  f.token,
		Body:   body,
	})
	apitest.AssertStatus(t, rec, http.StatusOK)
	return apitest.DecodeJSON[novelSettingsResponse](t, rec)
}

func TestNovelSettings_PatchRoundTripsEveryField(t *testing.T) {
	f := newSettingsFixture(t)

	got := f.patch(t, map[string]any{
		"source_chapters_count": 412,
		"price_per_chapter":     8,
		"free_until_chapter":    20,
		"sell_by_arc":           true,
		"tips_enabled":          true,
		"early_access_hours":    24,
		"release_schedule":      "weekly",
		"cover_style":           "ink",
		"cover_color":           "#8B1E2D",
		"cover_text":            "จอมดาบ",
	})

	want := novelSettingsResponse{
		SourceChaptersCount: 412, PricePerChapter: 8, FreeUntilChapter: 20,
		SellByArc: true, TipsEnabled: true, EarlyAccessHours: 24,
		ReleaseSchedule: "weekly", CoverStyle: "ink",
		CoverColor: "#8B1E2D", CoverText: "จอมดาบ",
	}
	if got.SourceChaptersCount != want.SourceChaptersCount ||
		got.PricePerChapter != want.PricePerChapter ||
		got.FreeUntilChapter != want.FreeUntilChapter ||
		got.SellByArc != want.SellByArc ||
		got.TipsEnabled != want.TipsEnabled ||
		got.EarlyAccessHours != want.EarlyAccessHours ||
		got.ReleaseSchedule != want.ReleaseSchedule ||
		got.CoverStyle != want.CoverStyle ||
		got.CoverColor != want.CoverColor ||
		got.CoverText != want.CoverText {
		t.Fatalf("settings = %+v, want %+v", got, want)
	}
}

// The whole reason these fields are pointers: a translator must be able to turn
// a toggle back off, and false is not "unset".
func TestNovelSettings_ExplicitFalseAndZeroAreApplied(t *testing.T) {
	f := newSettingsFixture(t)

	f.patch(t, map[string]any{
		"sell_by_arc": true, "tips_enabled": true,
		"free_until_chapter": 20, "price_per_chapter": 8,
	})

	got := f.patch(t, map[string]any{
		"sell_by_arc": false, "tips_enabled": false,
		"free_until_chapter": 0, "price_per_chapter": 0,
	})
	if got.SellByArc || got.TipsEnabled {
		t.Fatalf("settings = %+v, want both toggles off", got)
	}
	if got.FreeUntilChapter != 0 || got.PricePerChapter != 0 {
		t.Fatalf("settings = %+v, want the zeros applied", got)
	}
}

// And a patch that omits them must not reset them.
func TestNovelSettings_OmittedFieldsAreLeftAlone(t *testing.T) {
	f := newSettingsFixture(t)

	f.patch(t, map[string]any{"sell_by_arc": true, "price_per_chapter": 12})
	got := f.patch(t, map[string]any{"title_th": "ชื่อใหม่"})

	if !got.SellByArc || got.PricePerChapter != 12 {
		t.Fatalf("settings = %+v, want the untouched fields preserved", got)
	}
}

func TestNovelSettings_RejectsOutOfRangeValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"early access beyond a week", map[string]any{"early_access_hours": 400}},
		{"negative early access", map[string]any{"early_access_hours": -1}},
		{"price above the ceiling", map[string]any{"price_per_chapter": 5000}},
		{"negative price", map[string]any{"price_per_chapter": -1}},
		{"unknown release schedule", map[string]any{"release_schedule": "fortnightly"}},
		{"unknown cover style", map[string]any{"cover_style": "neon"}},
		{"non-hex cover colour", map[string]any{"cover_color": "crimson"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSettingsFixture(t)
			rec := f.env.Do(apitest.Request{
				Method: http.MethodPatch,
				Path:   fmt.Sprintf("/api/v1/writer/novels/%d", f.novel.ID),
				Token:  f.token,
				Body:   tc.body,
			})
			apitest.AssertStatus(t, rec, http.StatusBadRequest)
		})
	}
}

// A new chapter inherits the novel's price rather than making the translator
// retype it, and free_until_chapter overrides even an explicit price.
func TestCreateChapter_DefaultsPriceFromTheNovelAndHonoursFreeUntil(t *testing.T) {
	f := newSettingsFixture(t)
	f.patch(t, map[string]any{"price_per_chapter": 7, "free_until_chapter": 3})

	create := func(chapterNo, price int) int {
		t.Helper()
		body := map[string]any{"chapter_no": chapterNo, "title": fmt.Sprintf("บท %d", chapterNo)}
		if price > 0 {
			body["price_coins"] = price
		}
		rec := f.env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   fmt.Sprintf("/api/v1/writer/novels/%d/chapters", f.novel.ID),
			Token:  f.token,
			Body:   body,
		})
		apitest.AssertStatus(t, rec, http.StatusCreated)
		return apitest.DecodeJSON[struct {
			PriceCoins int `json:"price_coins"`
		}](t, rec).PriceCoins
	}

	if got := create(5, 0); got != 7 {
		t.Fatalf("price = %d, want the novel's 7 inherited", got)
	}
	if got := create(2, 0); got != 0 {
		t.Fatalf("price = %d, want 0 below free_until_chapter", got)
	}
	// An explicit price inside the free range must still lose: the promise to
	// readers is that those chapters are free.
	if got := create(3, 99); got != 0 {
		t.Fatalf("price = %d, want free_until_chapter to win over an explicit price", got)
	}
	if got := create(9, 15); got != 15 {
		t.Fatalf("price = %d, want the explicit 15 above the free range", got)
	}
}
