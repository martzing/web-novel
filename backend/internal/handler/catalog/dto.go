package catalog

import domain "github.com/mokchan/webnovel-backend/internal/domain/catalog"

// GenreResponse is the wire representation of a domain.Genre.
type GenreResponse struct {
	ID     int64  `json:"id,string"`
	Slug   string `json:"slug"`
	NameTH string `json:"name_th"`
}

// NovelResponse is the wire representation of a domain.Novel for list endpoints.
type NovelResponse struct {
	ID             int64           `json:"id,string"`
	Slug           string          `json:"slug"`
	TitleTH        string          `json:"title_th"`
	TitleCN        string          `json:"title_cn,omitempty"`
	AuthorName     string          `json:"author_name,omitempty"`
	CoverURL       string          `json:"cover_url,omitempty"`
	Status         string          `json:"status"`
	RatingAvg      float64         `json:"rating_avg"`
	RatingCount    int             `json:"rating_count"`
	FollowersCount int             `json:"followers_count"`
	ChaptersCount  int             `json:"chapters_count"`
	Genres         []GenreResponse `json:"genres"`
}

// NovelDetailResponse extends NovelResponse with description, arcs and counts.
type NovelDetailResponse struct {
	NovelResponse
	Description   string        `json:"description,omitempty"`
	Arcs          []ArcResponse `json:"arcs"`
	GlossaryCount int           `json:"glossary_count"`
	CommentsCount int           `json:"comments_count"`
}

// RankedNovelResponse is one row of the weekly leaderboard.
type RankedNovelResponse struct {
	NovelResponse
	Rank  int     `json:"rank"`
	Score float64 `json:"score"`
}

// ArcResponse is the wire representation of a story arc.
type ArcResponse struct {
	ID            int64  `json:"id,string"`
	ArcNo         int    `json:"arc_no"`
	Name          string `json:"name"`
	FromChapterNo int    `json:"from_chapter_no"`
	ToChapterNo   int    `json:"to_chapter_no"`
}

// ChapterListResponse is the wire representation of a single ToC row.
// Unlocked is filled in for authenticated requests.
type ChapterListResponse struct {
	ID          int64  `json:"id,string"`
	ChapterNo   int    `json:"chapter_no"`
	Title       string `json:"title"`
	PriceCoins  int    `json:"price_coins"`
	WordCount   int    `json:"word_count"`
	PublishedAt string `json:"published_at,omitempty"`
	ArcID       *int64 `json:"arc_id,string,omitempty"`
	Unlocked    bool   `json:"unlocked"`
}

// GlossaryGroupResponse is one glossary category with its entries.
type GlossaryGroupResponse struct {
	ID      int64                   `json:"id,string"`
	Name    string                  `json:"name"`
	SortNo  int                     `json:"sort_no"`
	Entries []GlossaryEntryResponse `json:"entries"`
}

// GlossaryEntryResponse is a single glossary term.
type GlossaryEntryResponse struct {
	ID      int64  `json:"id,string"`
	TermKey string `json:"term_key"`
	TitleTH string `json:"title_th"`
	TitleCN string `json:"title_cn,omitempty"`
	Body    string `json:"body"`
	Kind    string `json:"kind,omitempty"`
}

func toGenreResponses(items []domain.Genre) []GenreResponse {
	out := make([]GenreResponse, 0, len(items))
	for _, g := range items {
		out = append(out, GenreResponse{ID: g.ID, Slug: g.Slug, NameTH: g.NameTH})
	}
	return out
}

func toNovelResponses(items []domain.Novel) []NovelResponse {
	out := make([]NovelResponse, 0, len(items))
	for _, n := range items {
		out = append(out, toNovelResponse(n))
	}
	return out
}

func toNovelResponse(n domain.Novel) NovelResponse {
	return NovelResponse{
		ID:             n.ID,
		Slug:           n.Slug,
		TitleTH:        n.TitleTH,
		TitleCN:        n.TitleCN,
		AuthorName:     n.AuthorName,
		CoverURL:       n.CoverURL,
		Status:         n.Status,
		RatingAvg:      n.RatingAvg,
		RatingCount:    n.RatingCount,
		FollowersCount: n.FollowersCount,
		ChaptersCount:  n.ChaptersCount,
		Genres:         toGenreResponses(n.Genres),
	}
}

func toNovelDetailResponse(d domain.NovelDetail) NovelDetailResponse {
	return NovelDetailResponse{
		NovelResponse: toNovelResponse(d.Novel),
		Description:   d.Novel.Description,
		Arcs:          toArcResponses(d.Arcs),
		GlossaryCount: d.GlossaryCount,
		CommentsCount: d.CommentsCount,
	}
}

func toArcResponses(items []domain.Arc) []ArcResponse {
	out := make([]ArcResponse, 0, len(items))
	for _, a := range items {
		out = append(out, ArcResponse{
			ID:            a.ID,
			ArcNo:         a.ArcNo,
			Name:          a.Name,
			FromChapterNo: a.FromChapterNo,
			ToChapterNo:   a.ToChapterNo,
		})
	}
	return out
}

func toRankedNovelResponses(items []domain.RankedNovel) []RankedNovelResponse {
	out := make([]RankedNovelResponse, 0, len(items))
	for _, n := range items {
		out = append(out, RankedNovelResponse{
			NovelResponse: toNovelResponse(n.Novel),
			Rank:          n.Rank,
			Score:         n.Score,
		})
	}
	return out
}

func toChapterListResponses(items []domain.Chapter) []ChapterListResponse {
	out := make([]ChapterListResponse, 0, len(items))
	for _, c := range items {
		out = append(out, ChapterListResponse{
			ID:          c.ID,
			ChapterNo:   c.ChapterNo,
			Title:       c.Title,
			PriceCoins:  c.PriceCoins,
			WordCount:   c.WordCount,
			PublishedAt: c.PublishedAt,
			ArcID:       c.ArcID,
			Unlocked:    c.Unlocked,
		})
	}
	return out
}

func toGlossaryResponses(items []domain.GlossaryGroup) []GlossaryGroupResponse {
	out := make([]GlossaryGroupResponse, 0, len(items))
	for _, g := range items {
		entries := make([]GlossaryEntryResponse, 0, len(g.Entries))
		for _, e := range g.Entries {
			entries = append(entries, GlossaryEntryResponse{
				ID:      e.ID,
				TermKey: e.TermKey,
				TitleTH: e.TitleTH,
				TitleCN: e.TitleCN,
				Body:    e.Body,
				Kind:    e.Kind,
			})
		}
		out = append(out, GlossaryGroupResponse{
			ID:      g.ID,
			Name:    g.Name,
			SortNo:  g.SortNo,
			Entries: entries,
		})
	}
	return out
}
