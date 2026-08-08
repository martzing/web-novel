package library

import (
	"time"

	domain "github.com/mokchan/webnovel-backend/internal/domain/library"
)

// ShelfItemResponse is one novel on the shelf, with enough detail for the card.
type ShelfItemResponse struct {
	NovelID       int64   `json:"novel_id,string"`
	Slug          string  `json:"slug"`
	TitleTH       string  `json:"title_th"`
	TitleCN       string  `json:"title_cn,omitempty"`
	CoverURL      string  `json:"cover_url,omitempty"`
	Status        string  `json:"status"`
	ChaptersCount int     `json:"chapters_count"`
	LastChapterNo *int    `json:"last_chapter_no,omitempty"`
	Pct           float64 `json:"pct"`
}

// CountsResponse drives the shelf tab badges.
type CountsResponse struct {
	Reading int `json:"reading"`
	Saved   int `json:"saved"`
	Done    int `json:"done"`
	Total   int `json:"total"`
}

// EntryResponse is a bare shelf row.
type EntryResponse struct {
	NovelID int64  `json:"novel_id,string"`
	Status  string `json:"status"`
}

// BookmarkResponse is one saved paragraph.
type BookmarkResponse struct {
	ID         int64  `json:"id,string"`
	NovelID    int64  `json:"novel_id,string"`
	ChapterID  int64  `json:"chapter_id,string"`
	ChapterNo  int    `json:"chapter_no"`
	Title      string `json:"title"`
	ParaAnchor int    `json:"para_anchor"`
	Excerpt    string `json:"excerpt"`
	Note       string `json:"note,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type shelfRequest struct {
	Status string `json:"status"`
}

type bookmarkRequest struct {
	NovelID    int64  `json:"novel_id,string"`
	ChapterID  int64  `json:"chapter_id,string"`
	ParaAnchor int    `json:"para_anchor"`
	Excerpt    string `json:"excerpt"`
	Note       string `json:"note"`
}

func toShelfResponses(entries []domain.EntryWithNovel) []ShelfItemResponse {
	out := make([]ShelfItemResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, ShelfItemResponse{
			NovelID:       e.NovelID,
			Slug:          e.Slug,
			TitleTH:       e.TitleTH,
			TitleCN:       e.TitleCN,
			CoverURL:      e.CoverURL,
			Status:        e.Status,
			ChaptersCount: e.ChaptersCount,
			LastChapterNo: e.LastChapterNo,
			Pct:           e.Pct,
		})
	}
	return out
}

func toEntryResponse(e domain.Entry) EntryResponse {
	return EntryResponse{NovelID: e.NovelID, Status: e.Status}
}

func toBookmarkResponse(b domain.Bookmark) BookmarkResponse {
	return BookmarkResponse{
		ID:         b.ID,
		NovelID:    b.NovelID,
		ChapterID:  b.ChapterID,
		ChapterNo:  b.ChapterNo,
		Title:      b.Title,
		ParaAnchor: b.ParaAnchor,
		Excerpt:    b.Excerpt,
		Note:       b.Note,
		CreatedAt:  b.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toBookmarkResponses(bookmarks []domain.Bookmark) []BookmarkResponse {
	out := make([]BookmarkResponse, 0, len(bookmarks))
	for _, b := range bookmarks {
		out = append(out, toBookmarkResponse(b))
	}
	return out
}
