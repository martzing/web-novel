package reading

import domain "github.com/mokchan/webnovel-backend/internal/domain/reading"

// ChapterViewResponse is the wire representation of a readable chapter.
// BodyHTML is a pointer so a locked chapter serialises `"body_html": null`
// rather than omitting the field, matching docs/api-spec.md.
type ChapterViewResponse struct {
	ID           int64  `json:"id,string"`
	NovelID      int64  `json:"novel_id,string"`
	NovelSlug    string `json:"novel_slug"`
	NovelTitleTH string `json:"novel_title_th"`
	ArcID        *int64 `json:"arc_id,string,omitempty"`
	ArcNo        int    `json:"arc_no,omitempty"`
	ArcName      string `json:"arc_name,omitempty"`
	ChapterNo    int    `json:"chapter_no"`
	Title        string `json:"title"`
	PriceCoins   int    `json:"price_coins"`
	WordCount    int    `json:"word_count"`
	PublishedAt  string `json:"published_at,omitempty"`
	Locked       bool   `json:"locked"`
	// LockedReason tells the client which call to action to show: "paywall"
	// (buy the chapter) or "early_access" (subscribe to auto-unlock).
	LockedReason string `json:"locked_reason,omitempty"`
	// TipsEnabled drives the tip control at the end of the chapter.
	TipsEnabled bool    `json:"tips_enabled"`
	BodyHTML    *string `json:"body_html"`
	PrevID      *int64  `json:"prev_id,string,omitempty"`
	NextID      *int64  `json:"next_id,string,omitempty"`
}

// ProgressResponse is a reader's saved position in one novel.
type ProgressResponse struct {
	NovelID       int64   `json:"novel_id,string"`
	LastChapterID *int64  `json:"last_chapter_id,string,omitempty"`
	LastChapterNo *int    `json:"last_chapter_no,omitempty"`
	ParaAnchor    int     `json:"para_anchor"`
	Pct           float64 `json:"pct"`
	UpdatedAt     string  `json:"updated_at,omitempty"`
}

// neighbourResponse answers /chapters/{id}/next and /prev.
type neighbourResponse struct {
	ID *int64 `json:"id,string"`
}

// progressRequest is the PUT body for /me/progress/{novel_id}.
type progressRequest struct {
	LastChapterID *int64  `json:"last_chapter_id,string"`
	LastChapterNo *int    `json:"last_chapter_no"`
	ParaAnchor    int     `json:"para_anchor"`
	Pct           float64 `json:"pct"`
}

// readEventRequest is the POST body for /chapters/{id}/read-event.
type readEventRequest struct {
	SessionID string `json:"session_id"`
}

func toChapterViewResponse(v domain.ChapterView) ChapterViewResponse {
	out := ChapterViewResponse{
		ID:           v.ID,
		NovelID:      v.NovelID,
		NovelSlug:    v.NovelSlug,
		NovelTitleTH: v.NovelTitleTH,
		ArcID:        v.ArcID,
		ArcNo:        v.ArcNo,
		ArcName:      v.ArcName,
		ChapterNo:    v.ChapterNo,
		Title:        v.Title,
		PriceCoins:   v.PriceCoins,
		WordCount:    v.WordCount,
		PublishedAt:  v.PublishedAt,
		Locked:       v.Locked,
		LockedReason: v.LockedReason,
		TipsEnabled:  v.TipsEnabled,
		PrevID:       v.PrevID,
		NextID:       v.NextID,
	}
	if !v.Locked {
		body := v.BodyHTML
		out.BodyHTML = &body
	}
	return out
}

func toProgressResponse(p domain.Progress) ProgressResponse {
	out := ProgressResponse{
		NovelID:       p.NovelID,
		LastChapterID: p.LastChapterID,
		LastChapterNo: p.LastChapterNo,
		ParaAnchor:    p.ParaAnchor,
		Pct:           p.Pct,
	}
	if !p.UpdatedAt.IsZero() {
		out.UpdatedAt = p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return out
}
