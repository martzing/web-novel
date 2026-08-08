package social

import (
	"time"

	domain "github.com/mokchan/webnovel-backend/internal/domain/social"
)

// AuthorResponse is the identity shown beside a comment or review.
type AuthorResponse struct {
	ID          int64  `json:"id,string"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Role        string `json:"role"`
}

// CommentResponse is one comment with its replies.
type CommentResponse struct {
	ID              int64             `json:"id,string"`
	ChapterID       int64             `json:"chapter_id,string"`
	ParentID        *int64            `json:"parent_id,string,omitempty"`
	Body            string            `json:"body"`
	IsSpoilerHidden bool              `json:"is_spoiler_hidden"`
	LikesCount      int               `json:"likes_count"`
	Liked           bool              `json:"liked"`
	IsTranslator    bool              `json:"is_translator"`
	CreatedAt       string            `json:"created_at"`
	Author          AuthorResponse    `json:"author"`
	Replies         []CommentResponse `json:"replies"`
}

// ReviewResponse is a star rating with optional prose.
type ReviewResponse struct {
	ID        int64          `json:"id,string"`
	NovelID   int64          `json:"novel_id,string"`
	Rating    int            `json:"rating"`
	Body      string         `json:"body,omitempty"`
	CreatedAt string         `json:"created_at"`
	Author    AuthorResponse `json:"author"`
}

type commentRequest struct {
	Body            string `json:"body"`
	ParentID        *int64 `json:"parent_id,string"`
	IsSpoilerHidden bool   `json:"is_spoiler_hidden"`
}

type reviewRequest struct {
	Rating int    `json:"rating"`
	Body   string `json:"body"`
}

func toAuthorResponse(a domain.Author) AuthorResponse {
	return AuthorResponse{
		ID:          a.ID,
		DisplayName: a.DisplayName,
		AvatarURL:   a.AvatarURL,
		Role:        a.Role,
	}
}

func toCommentResponse(c domain.Comment) CommentResponse {
	out := CommentResponse{
		ID:              c.ID,
		ChapterID:       c.ChapterID,
		ParentID:        c.ParentID,
		Body:            c.Body,
		IsSpoilerHidden: c.IsSpoilerHidden,
		LikesCount:      c.LikesCount,
		Liked:           c.Liked,
		IsTranslator:    c.IsTranslator,
		CreatedAt:       c.CreatedAt.UTC().Format(time.RFC3339),
		Author:          toAuthorResponse(c.Author),
		Replies:         []CommentResponse{},
	}
	for _, reply := range c.Replies {
		out.Replies = append(out.Replies, toCommentResponse(reply))
	}
	return out
}

func toCommentResponses(comments []domain.Comment) []CommentResponse {
	out := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		out = append(out, toCommentResponse(c))
	}
	return out
}

func toReviewResponse(r domain.Review) ReviewResponse {
	return ReviewResponse{
		ID:        r.ID,
		NovelID:   r.NovelID,
		Rating:    r.Rating,
		Body:      r.Body,
		CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
		Author:    toAuthorResponse(r.Author),
	}
}

func toReviewResponses(reviews []domain.Review) []ReviewResponse {
	out := make([]ReviewResponse, 0, len(reviews))
	for _, r := range reviews {
		out = append(out, toReviewResponse(r))
	}
	return out
}
