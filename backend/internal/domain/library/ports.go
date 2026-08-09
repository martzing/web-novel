package library

import (
	"context"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
)

// Repository is the driven port for the reader's shelf.
type Repository interface {
	ListShelf(ctx context.Context, userID int64, tab string, p page.Page) ([]EntryWithNovel, string, error)
	CountShelf(ctx context.Context, userID int64) (Counts, error)
	UpsertShelf(ctx context.Context, e Entry) (*Entry, error)
	RemoveShelf(ctx context.Context, userID, novelID int64) error

	ListBookmarks(ctx context.Context, userID int64, novelID *int64, p page.Page) ([]Bookmark, string, error)
	CreateBookmark(ctx context.Context, b Bookmark) (*Bookmark, error)
	// DeleteBookmark must distinguish a missing bookmark (404) from one owned
	// by somebody else (403), so a bare conditional DELETE is not enough.
	DeleteBookmark(ctx context.Context, id, userID int64) error

	Follow(ctx context.Context, userID, novelID int64) error
	Unfollow(ctx context.Context, userID, novelID int64) error
	IsFollowing(ctx context.Context, userID, novelID int64) (bool, error)
	ListFollowerIDs(ctx context.Context, novelID, afterID int64, limit int) ([]int64, error)

	// FollowMany and UnfollowMany back ติดตามทั้งชุด. They return how many rows
	// actually changed so followers_count stays exact under concurrent calls:
	// the count comes from the rows Postgres really inserted or deleted, not
	// from the size of the request.
	FollowMany(ctx context.Context, userID int64, novelIDs []int64) (int, error)
	UnfollowMany(ctx context.Context, userID int64, novelIDs []int64) (int, error)
	CountFollowing(ctx context.Context, userID int64, novelIDs []int64) (int, error)
}

// SeriesBooks resolves a series to the novels it holds.
//
// Declared here rather than imported, so the library context never depends on
// catalog. The catalog repository satisfies it and the composition root wires
// the two together — the same shape as reading.Entitlements over wallet.
type SeriesBooks interface {
	NovelIDsInSeries(ctx context.Context, idOrSlug string) ([]int64, error)
}
