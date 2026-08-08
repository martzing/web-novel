package social

import (
	"context"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
)

// Repository is the driven port for comments, likes and reviews.
type Repository interface {
	// ListComments returns top-level comments with their replies attached.
	ListComments(ctx context.Context, chapterID int64, sort string, viewerID int64, p page.Page) ([]Comment, string, error)
	GetComment(ctx context.Context, id int64) (*Comment, error)
	// CreateComment stamps is_translator server-side by comparing the author
	// against the chapter's translator; the client never supplies it.
	CreateComment(ctx context.Context, c Comment) (*Comment, error)
	SoftDeleteComment(ctx context.Context, id int64) error
	// ChapterTranslatorID resolves who may moderate a chapter's comments.
	ChapterTranslatorID(ctx context.Context, chapterID int64) (*int64, error)

	// Like and Unlike are idempotent and keep comments.likes_count in step.
	Like(ctx context.Context, userID, commentID int64) (int, error)
	Unlike(ctx context.Context, userID, commentID int64) (int, error)

	// UpsertReview writes the review and recomputes the novel's rating
	// aggregate in the same transaction.
	UpsertReview(ctx context.Context, r Review) (*Review, error)
	ListReviews(ctx context.Context, novelID int64, p page.Page) ([]Review, string, error)
	GetUserReview(ctx context.Context, novelID, userID int64) (*Review, error)
}
