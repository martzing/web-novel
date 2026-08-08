package reading

import "context"

// Repository is the driven port for reading a chapter and tracking position.
type Repository interface {
	// GetChapter returns published chapter metadata, or ErrNotFound. A draft or
	// scheduled chapter is invisible to readers and reported as not found.
	GetChapter(ctx context.Context, id int64, includeUnpublished bool) (*Chapter, error)
	GetBody(ctx context.Context, chapterID int64) (string, error)
	NeighbourIDs(ctx context.Context, novelID int64, chapterNo int) (prev, next *int64, err error)

	GetProgress(ctx context.Context, userID, novelID int64) (*Progress, error)
	UpsertProgress(ctx context.Context, p Progress) (*Progress, error)
	ListProgress(ctx context.Context, userID int64, novelIDs []int64) (map[int64]Progress, error)

	InsertReadEvent(ctx context.Context, e ReadEvent) error
}

// Entitlements reports chapter ownership. It is satisfied by the wallet
// repository; the reading package never imports wallet.
type Entitlements interface {
	IsChapterUnlocked(ctx context.Context, userID, chapterID int64) (bool, error)
}
