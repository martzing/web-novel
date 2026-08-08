package writer

import (
	"time"

	domain "github.com/mokchan/webnovel-backend/internal/domain/writer"
)

// NovelResponse is a novel as its translator sees it.
type NovelResponse struct {
	ID          int64   `json:"id,string"`
	Slug        string  `json:"slug"`
	TitleTH     string  `json:"title_th"`
	TitleCN     string  `json:"title_cn,omitempty"`
	AuthorName  string  `json:"author_name,omitempty"`
	Description string  `json:"description,omitempty"`
	CoverURL    string  `json:"cover_url,omitempty"`
	Status      string  `json:"status"`
	GenreIDs    []int64 `json:"genre_ids"`
}

// ArcResponse is a chapter range.
type ArcResponse struct {
	ID            int64  `json:"id,string"`
	NovelID       int64  `json:"novel_id,string"`
	ArcNo         int    `json:"arc_no"`
	Name          string `json:"name"`
	FromChapterNo int    `json:"from_chapter_no"`
	ToChapterNo   int    `json:"to_chapter_no"`
}

// ChapterResponse is the editor's view of a chapter, source included.
type ChapterResponse struct {
	ID          int64  `json:"id,string"`
	NovelID     int64  `json:"novel_id,string"`
	ArcID       *int64 `json:"arc_id,string,omitempty"`
	ChapterNo   int    `json:"chapter_no"`
	Title       string `json:"title"`
	BodySource  string `json:"body_source"`
	BodyHTML    string `json:"body_html,omitempty"`
	PriceCoins  int    `json:"price_coins"`
	WordCount   int    `json:"word_count"`
	Status      string `json:"status"`
	ScheduledAt string `json:"scheduled_at,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// GlossaryGroupResponse is a glossary category with its terms.
type GlossaryGroupResponse struct {
	ID      int64                   `json:"id,string"`
	NovelID int64                   `json:"novel_id,string"`
	Name    string                  `json:"name"`
	SortNo  int                     `json:"sort_no"`
	Entries []GlossaryEntryResponse `json:"entries"`
}

// GlossaryEntryResponse is one term.
type GlossaryEntryResponse struct {
	ID      int64  `json:"id,string"`
	GroupID int64  `json:"group_id,string"`
	TermKey string `json:"term_key"`
	TitleTH string `json:"title_th"`
	TitleCN string `json:"title_cn,omitempty"`
	Body    string `json:"body"`
	Kind    string `json:"kind,omitempty"`
}

// StatsResponse powers the writer stats page.
type StatsResponse struct {
	Reads         int                   `json:"reads"`
	Followers     int                   `json:"followers"`
	CoinsEarned   int                   `json:"coins_earned"`
	ReadsTrendPct float64               `json:"reads_trend_pct"`
	CoinsTrendPct float64               `json:"coins_trend_pct"`
	PeriodFrom    string                `json:"period_from"`
	PeriodTo      string                `json:"period_to"`
	Series        []DailyPointResponse  `json:"series"`
	TopChapters   []ChapterPerfResponse `json:"top_chapters"`
}

// DailyPointResponse is one day on the stats chart.
type DailyPointResponse struct {
	Day         string `json:"day"`
	Reads       int    `json:"reads"`
	CoinsEarned int    `json:"coins_earned"`
	Followers   int    `json:"followers"`
}

// ChapterPerfResponse is one row of the best-performing table.
type ChapterPerfResponse struct {
	ChapterID   int64  `json:"chapter_id,string"`
	ChapterNo   int    `json:"chapter_no"`
	Title       string `json:"title"`
	Reads       int    `json:"reads"`
	CoinsEarned int    `json:"coins_earned"`
}

type novelRequest struct {
	Slug        string  `json:"slug"`
	TitleTH     string  `json:"title_th"`
	TitleCN     string  `json:"title_cn"`
	AuthorName  string  `json:"author_name"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	SeriesID    *int64  `json:"series_id,string"`
	GenreIDs    []int64 `json:"genre_ids"`
}

type arcRequest struct {
	ArcNo         int    `json:"arc_no"`
	Name          string `json:"name"`
	FromChapterNo int    `json:"from_chapter_no"`
	ToChapterNo   int    `json:"to_chapter_no"`
}

type chapterRequest struct {
	ChapterNo  int    `json:"chapter_no"`
	Title      string `json:"title"`
	BodySource string `json:"body_source"`
	PriceCoins int    `json:"price_coins"`
}

type publishRequest struct {
	ScheduledAt *time.Time `json:"scheduled_at"`
}

type glossaryGroupRequest struct {
	Name   string `json:"name"`
	SortNo int    `json:"sort_no"`
}

type glossaryEntryRequest struct {
	GroupID int64  `json:"group_id,string"`
	TermKey string `json:"term_key"`
	TitleTH string `json:"title_th"`
	TitleCN string `json:"title_cn"`
	Body    string `json:"body"`
	Kind    string `json:"kind"`
}

func toNovelResponse(n domain.NovelDraft) NovelResponse {
	out := NovelResponse{
		ID:          n.ID,
		Slug:        n.Slug,
		TitleTH:     n.TitleTH,
		TitleCN:     n.TitleCN,
		AuthorName:  n.AuthorName,
		Description: n.Description,
		CoverURL:    n.CoverURL,
		Status:      n.Status,
		GenreIDs:    n.GenreIDs,
	}
	if out.GenreIDs == nil {
		out.GenreIDs = []int64{}
	}
	return out
}

func toNovelResponses(novels []domain.NovelDraft) []NovelResponse {
	out := make([]NovelResponse, 0, len(novels))
	for _, n := range novels {
		out = append(out, toNovelResponse(n))
	}
	return out
}

func toArcResponse(a domain.Arc) ArcResponse {
	return ArcResponse{
		ID:            a.ID,
		NovelID:       a.NovelID,
		ArcNo:         a.ArcNo,
		Name:          a.Name,
		FromChapterNo: a.FromChapterNo,
		ToChapterNo:   a.ToChapterNo,
	}
}

func toArcResponses(arcs []domain.Arc) []ArcResponse {
	out := make([]ArcResponse, 0, len(arcs))
	for _, a := range arcs {
		out = append(out, toArcResponse(a))
	}
	return out
}

func toChapterResponse(c domain.Chapter) ChapterResponse {
	out := ChapterResponse{
		ID:         c.ID,
		NovelID:    c.NovelID,
		ArcID:      c.ArcID,
		ChapterNo:  c.ChapterNo,
		Title:      c.Title,
		BodySource: c.BodySource,
		BodyHTML:   c.BodyHTML,
		PriceCoins: c.PriceCoins,
		WordCount:  c.WordCount,
		Status:     c.Status,
	}
	if c.ScheduledAt != nil {
		out.ScheduledAt = c.ScheduledAt.UTC().Format(time.RFC3339)
	}
	if c.PublishedAt != nil {
		out.PublishedAt = c.PublishedAt.UTC().Format(time.RFC3339)
	}
	if !c.UpdatedAt.IsZero() {
		out.UpdatedAt = c.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func toChapterResponses(chapters []domain.Chapter) []ChapterResponse {
	out := make([]ChapterResponse, 0, len(chapters))
	for _, c := range chapters {
		out = append(out, toChapterResponse(c))
	}
	return out
}

func toGlossaryEntryResponse(e domain.GlossaryEntry) GlossaryEntryResponse {
	return GlossaryEntryResponse{
		ID:      e.ID,
		GroupID: e.GroupID,
		TermKey: e.TermKey,
		TitleTH: e.TitleTH,
		TitleCN: e.TitleCN,
		Body:    e.Body,
		Kind:    e.Kind,
	}
}

func toGlossaryResponses(groups []domain.GlossaryGroup) []GlossaryGroupResponse {
	out := make([]GlossaryGroupResponse, 0, len(groups))
	for _, g := range groups {
		entries := make([]GlossaryEntryResponse, 0, len(g.Entries))
		for _, e := range g.Entries {
			entries = append(entries, toGlossaryEntryResponse(e))
		}
		out = append(out, GlossaryGroupResponse{
			ID:      g.ID,
			NovelID: g.NovelID,
			Name:    g.Name,
			SortNo:  g.SortNo,
			Entries: entries,
		})
	}
	return out
}

func toStatsResponse(s domain.NovelStats) StatsResponse {
	series := make([]DailyPointResponse, 0, len(s.Series))
	for _, p := range s.Series {
		series = append(series, DailyPointResponse{
			Day:         p.Day.UTC().Format("2006-01-02"),
			Reads:       p.Reads,
			CoinsEarned: p.CoinsEarned,
			Followers:   p.Followers,
		})
	}

	top := make([]ChapterPerfResponse, 0, len(s.TopChapters))
	for _, c := range s.TopChapters {
		top = append(top, ChapterPerfResponse{
			ChapterID:   c.ChapterID,
			ChapterNo:   c.ChapterNo,
			Title:       c.Title,
			Reads:       c.Reads,
			CoinsEarned: c.CoinsEarned,
		})
	}

	return StatsResponse{
		Reads:         s.Reads,
		Followers:     s.Followers,
		CoinsEarned:   s.CoinsEarned,
		ReadsTrendPct: s.ReadsTrendPct,
		CoinsTrendPct: s.CoinsTrendPct,
		PeriodFrom:    s.PeriodFrom.UTC().Format("2006-01-02"),
		PeriodTo:      s.PeriodTo.UTC().Format("2006-01-02"),
		Series:        series,
		TopChapters:   top,
	}
}
