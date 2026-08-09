package catalog

import (
	"context"
	"errors"
	"reflect"
	"testing"

	domain "github.com/mokchan/webnovel-backend/internal/domain/catalog"
)

// fakeRepo lets each test wire only the methods it exercises; an unwired
// method panics, which keeps a test from silently depending on a call it did
// not intend to make.
type fakeRepo struct {
	listGenres     func(ctx context.Context) ([]domain.Genre, error)
	listNovels     func(ctx context.Context, filter domain.NovelFilter) ([]domain.Novel, bool, error)
	getNovelBySlug func(ctx context.Context, slug string) (*domain.NovelDetail, error)
	getNovelByID   func(ctx context.Context, id int64) (*domain.NovelDetail, error)
	listArcs       func(ctx context.Context, novelID int64) ([]domain.Arc, error)
	listChapters   func(ctx context.Context, novelID int64, limit int) ([]domain.Chapter, error)
	getGlossary    func(ctx context.Context, novelID int64) ([]domain.GlossaryGroup, error)
	weeklyRanking  func(ctx context.Context, limit int) ([]domain.RankedNovel, error)
}

func (f *fakeRepo) ListGenres(ctx context.Context) ([]domain.Genre, error) {
	return f.listGenres(ctx)
}

func (f *fakeRepo) ListNovels(ctx context.Context, filter domain.NovelFilter) ([]domain.Novel, bool, error) {
	return f.listNovels(ctx, filter)
}

func (f *fakeRepo) GetNovelBySlug(ctx context.Context, slug string) (*domain.NovelDetail, error) {
	return f.getNovelBySlug(ctx, slug)
}

func (f *fakeRepo) GetNovelByID(ctx context.Context, id int64) (*domain.NovelDetail, error) {
	return f.getNovelByID(ctx, id)
}

func (f *fakeRepo) ListArcs(ctx context.Context, novelID int64) ([]domain.Arc, error) {
	return f.listArcs(ctx, novelID)
}

func (f *fakeRepo) ListChapters(ctx context.Context, novelID int64, limit int) ([]domain.Chapter, error) {
	return f.listChapters(ctx, novelID, limit)
}

func (f *fakeRepo) GetGlossary(ctx context.Context, novelID int64) ([]domain.GlossaryGroup, error) {
	return f.getGlossary(ctx, novelID)
}

func (f *fakeRepo) WeeklyRanking(ctx context.Context, limit int) ([]domain.RankedNovel, error) {
	return f.weeklyRanking(ctx, limit)
}

func (f *fakeRepo) GetSeries(context.Context, string) (*domain.SeriesDetail, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeRepo) RelatedNovels(context.Context, int64) ([]domain.RelatedNovel, error) {
	return nil, nil
}

// fakeEntitlements stands in for the wallet repository.
type fakeEntitlements struct {
	unlocked func(ctx context.Context, userID int64, chapterIDs []int64) (map[int64]bool, error)
}

func (f *fakeEntitlements) ListUnlockedChapterIDs(ctx context.Context, userID int64, chapterIDs []int64) (map[int64]bool, error) {
	return f.unlocked(ctx, userID, chapterIDs)
}

func TestListNovels_AppliesDefaults(t *testing.T) {
	tests := []struct {
		name   string
		input  domain.NovelFilter
		expect domain.NovelFilter
	}{
		{
			name:   "empty filter falls back to popular sort and limit 20",
			input:  domain.NovelFilter{},
			expect: domain.NovelFilter{Sort: "popular", Limit: 20},
		},
		{
			name:   "over-large limit is clamped to 20",
			input:  domain.NovelFilter{Limit: 999, Sort: "popular"},
			expect: domain.NovelFilter{Sort: "popular", Limit: 20},
		},
		{
			name:   "latest sort is preserved",
			input:  domain.NovelFilter{Sort: "latest", Limit: 15},
			expect: domain.NovelFilter{Sort: "latest", Limit: 15},
		},
		{
			name:   "unknown sort falls back to popular",
			input:  domain.NovelFilter{Sort: "trending", Limit: 5},
			expect: domain.NovelFilter{Sort: "popular", Limit: 5},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got domain.NovelFilter
			repo := &fakeRepo{
				listNovels: func(_ context.Context, filter domain.NovelFilter) ([]domain.Novel, bool, error) {
					got = filter
					return []domain.Novel{}, false, nil
				},
			}
			svc := New(repo, nil)
			if _, _, err := svc.ListNovels(context.Background(), tc.input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.expect) {
				t.Fatalf("filter mismatch\n got: %+v\nwant: %+v", got, tc.expect)
			}
		})
	}
}

// GetNovel takes one route parameter that may be either an id or a slug,
// because gin cannot host two differently named wildcards in one path segment.
func TestGetNovel_ResolvesIDOrSlug(t *testing.T) {
	tests := []struct {
		name       string
		param      string
		wantByID   int64
		wantBySlug string
	}{
		{"numeric parameter resolves by id", "42", 42, ""},
		{"slug parameter resolves by slug", "nine-streams-sword-immortal", 0, "nine-streams-sword-immortal"},
		{"slug that merely starts with digits resolves by slug", "9-streams", 0, "9-streams"},
		{"zero is not a valid id and falls through to slug", "0", 0, "0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotID int64
			var gotSlug string
			repo := &fakeRepo{
				getNovelByID: func(_ context.Context, id int64) (*domain.NovelDetail, error) {
					gotID = id
					return &domain.NovelDetail{}, nil
				},
				getNovelBySlug: func(_ context.Context, slug string) (*domain.NovelDetail, error) {
					gotSlug = slug
					return &domain.NovelDetail{}, nil
				},
			}
			svc := New(repo, nil)

			if _, err := svc.GetNovel(context.Background(), tc.param, 0); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotID != tc.wantByID {
				t.Fatalf("lookup by id = %d, want %d", gotID, tc.wantByID)
			}
			if gotSlug != tc.wantBySlug {
				t.Fatalf("lookup by slug = %q, want %q", gotSlug, tc.wantBySlug)
			}
		})
	}
}

func TestGetNovel_PropagatesNotFound(t *testing.T) {
	repo := &fakeRepo{
		getNovelBySlug: func(_ context.Context, _ string) (*domain.NovelDetail, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := New(repo, nil)

	if _, err := svc.GetNovel(context.Background(), "missing", 0); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListChapters_LimitClamped(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero becomes default 100", 0, 100},
		{"negative becomes default 100", -1, 100},
		{"over-cap becomes default 100", 9999, 100},
		{"in-range is preserved", 42, 42},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got int
			repo := &fakeRepo{
				listChapters: func(_ context.Context, _ int64, limit int) ([]domain.Chapter, error) {
					got = limit
					return nil, nil
				},
			}
			svc := New(repo, nil)
			if _, err := svc.ListChapters(context.Background(), 1, tc.input, 0); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Fatalf("limit = %d, want %d", got, tc.expected)
			}
		})
	}
}

func TestListChapters_AnonymousViewerSkipsEntitlementLookup(t *testing.T) {
	repo := &fakeRepo{
		listChapters: func(_ context.Context, _ int64, _ int) ([]domain.Chapter, error) {
			return []domain.Chapter{{ID: 1, PriceCoins: 5}}, nil
		},
	}
	entitlements := &fakeEntitlements{
		unlocked: func(context.Context, int64, []int64) (map[int64]bool, error) {
			t.Fatal("entitlement lookup must not run for an anonymous viewer")
			return nil, nil
		},
	}

	chapters, err := New(repo, entitlements).ListChapters(context.Background(), 1, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chapters[0].Unlocked {
		t.Fatal("paid chapter must not be marked unlocked for an anonymous viewer")
	}
}

// The table of contents resolves ownership with one bulk query, and only for
// paid rows — free chapters are unlocked by definition.
func TestListChapters_MarksUnlockedForEntitledViewer(t *testing.T) {
	var asked []int64
	repo := &fakeRepo{
		listChapters: func(_ context.Context, _ int64, _ int) ([]domain.Chapter, error) {
			return []domain.Chapter{
				{ID: 1, ChapterNo: 1, PriceCoins: 0},
				{ID: 2, ChapterNo: 2, PriceCoins: 5},
				{ID: 3, ChapterNo: 3, PriceCoins: 5},
			}, nil
		},
	}
	entitlements := &fakeEntitlements{
		unlocked: func(_ context.Context, userID int64, chapterIDs []int64) (map[int64]bool, error) {
			if userID != 77 {
				t.Fatalf("viewer id = %d, want 77", userID)
			}
			asked = chapterIDs
			return map[int64]bool{2: true}, nil
		},
	}

	chapters, err := New(repo, entitlements).ListChapters(context.Background(), 1, 10, 77)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := []int64{2, 3}; !reflect.DeepEqual(asked, want) {
		t.Fatalf("entitlement lookup asked for %v, want only the paid chapters %v", asked, want)
	}
	want := []bool{true, true, false}
	for i, c := range chapters {
		if c.Unlocked != want[i] {
			t.Fatalf("chapter %d unlocked = %v, want %v", c.ChapterNo, c.Unlocked, want[i])
		}
	}
}

func TestWeeklyRanking_ClampsLimit(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero becomes default 10", 0, 10},
		{"negative becomes default 10", -5, 10},
		{"over-cap becomes default 10", 500, 10},
		{"in-range is preserved", 25, 25},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got int
			repo := &fakeRepo{
				weeklyRanking: func(_ context.Context, limit int) ([]domain.RankedNovel, error) {
					got = limit
					return nil, nil
				},
			}
			if _, err := New(repo, nil).WeeklyRanking(context.Background(), tc.input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Fatalf("limit = %d, want %d", got, tc.expected)
			}
		})
	}
}
