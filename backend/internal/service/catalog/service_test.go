package catalog

import (
	"context"
	"errors"
	"reflect"
	"testing"

	domain "github.com/mokchan/webnovel-backend/internal/domain/catalog"
)

// fakeRepo lets each test wire only the methods it exercises.
type fakeRepo struct {
	listGenres     func(ctx context.Context) ([]domain.Genre, error)
	listNovels     func(ctx context.Context, filter domain.NovelFilter) ([]domain.Novel, error)
	getNovelBySlug func(ctx context.Context, slug string) (*domain.NovelDetail, error)
	listChapters   func(ctx context.Context, novelID int64, limit int) ([]domain.Chapter, error)
	getChapter     func(ctx context.Context, id int64) (*domain.Chapter, error)
	getChapterBody func(ctx context.Context, chapterID int64) (string, error)
}

func (f *fakeRepo) ListGenres(ctx context.Context) ([]domain.Genre, error) {
	return f.listGenres(ctx)
}
func (f *fakeRepo) ListNovels(ctx context.Context, filter domain.NovelFilter) ([]domain.Novel, error) {
	return f.listNovels(ctx, filter)
}
func (f *fakeRepo) GetNovelBySlug(ctx context.Context, slug string) (*domain.NovelDetail, error) {
	return f.getNovelBySlug(ctx, slug)
}
func (f *fakeRepo) ListChapters(ctx context.Context, novelID int64, limit int) ([]domain.Chapter, error) {
	return f.listChapters(ctx, novelID, limit)
}
func (f *fakeRepo) GetChapter(ctx context.Context, id int64) (*domain.Chapter, error) {
	return f.getChapter(ctx, id)
}
func (f *fakeRepo) GetChapterBody(ctx context.Context, chapterID int64) (string, error) {
	return f.getChapterBody(ctx, chapterID)
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
				listNovels: func(ctx context.Context, filter domain.NovelFilter) ([]domain.Novel, error) {
					got = filter
					return []domain.Novel{}, nil
				},
			}
			svc := New(repo)
			if _, err := svc.ListNovels(context.Background(), tc.input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.expect) {
				t.Fatalf("filter mismatch\n got: %+v\nwant: %+v", got, tc.expect)
			}
		})
	}
}

func TestGetChapter_FreeChapterReturnsBody(t *testing.T) {
	repo := &fakeRepo{
		getChapter: func(_ context.Context, id int64) (*domain.Chapter, error) {
			return &domain.Chapter{ID: id, PriceCoins: 0, Title: "อ่านฟรี"}, nil
		},
		getChapterBody: func(_ context.Context, chapterID int64) (string, error) {
			if chapterID != 42 {
				t.Fatalf("expected chapter id 42, got %d", chapterID)
			}
			return "<p>body</p>", nil
		},
	}
	svc := New(repo)

	view, err := svc.GetChapter(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Locked {
		t.Fatalf("expected locked=false for free chapter")
	}
	if view.BodyHTML != "<p>body</p>" {
		t.Fatalf("unexpected body: %q", view.BodyHTML)
	}
}

func TestGetChapter_PaidChapterIsLocked(t *testing.T) {
	bodyCalled := false
	repo := &fakeRepo{
		getChapter: func(_ context.Context, id int64) (*domain.Chapter, error) {
			return &domain.Chapter{ID: id, PriceCoins: 5, Title: "ต้องปลดล็อก"}, nil
		},
		getChapterBody: func(_ context.Context, _ int64) (string, error) {
			bodyCalled = true
			return "should not be read", nil
		},
	}
	svc := New(repo)

	view, err := svc.GetChapter(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !view.Locked {
		t.Fatalf("expected locked=true for paid chapter")
	}
	if view.BodyHTML != "" {
		t.Fatalf("expected empty body for locked chapter, got %q", view.BodyHTML)
	}
	if bodyCalled {
		t.Fatalf("body fetch must not be invoked for locked chapters")
	}
}

func TestGetChapter_PropagatesNotFound(t *testing.T) {
	repo := &fakeRepo{
		getChapter: func(_ context.Context, _ int64) (*domain.Chapter, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := New(repo)

	_, err := svc.GetChapter(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
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
			svc := New(repo)
			if _, err := svc.ListChapters(context.Background(), 1, tc.input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Fatalf("limit = %d, want %d", got, tc.expected)
			}
		})
	}
}
