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
	CoverURL       string          `json:"cover_url,omitempty"`
	Status         string          `json:"status"`
	RatingAvg      float64         `json:"rating_avg"`
	RatingCount    int             `json:"rating_count"`
	FollowersCount int             `json:"followers_count"`
	ChaptersCount  int             `json:"chapters_count"`
	Genres         []GenreResponse `json:"genres"`
}

// NovelDetailResponse extends NovelResponse with author, description, and arc list.
type NovelDetailResponse struct {
	NovelResponse
	AuthorName  string        `json:"author_name,omitempty"`
	Description string        `json:"description,omitempty"`
	Arcs        []ArcResponse `json:"arcs"`
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
type ChapterListResponse struct {
	ID          int64  `json:"id,string"`
	ChapterNo   int    `json:"chapter_no"`
	Title       string `json:"title"`
	PriceCoins  int    `json:"price_coins"`
	PublishedAt string `json:"published_at,omitempty"`
	ArcID       *int64 `json:"arc_id,string,omitempty"`
}

// ChapterViewResponse is the wire representation of a single readable chapter.
type ChapterViewResponse struct {
	ID         int64  `json:"id,string"`
	NovelID    int64  `json:"novel_id,string"`
	ChapterNo  int    `json:"chapter_no"`
	Title      string `json:"title"`
	PriceCoins int    `json:"price_coins"`
	Locked     bool   `json:"locked"`
	BodyHTML   string `json:"body_html,omitempty"`
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
	arcs := make([]ArcResponse, 0, len(d.Arcs))
	for _, a := range d.Arcs {
		arcs = append(arcs, ArcResponse{
			ID:            a.ID,
			ArcNo:         a.ArcNo,
			Name:          a.Name,
			FromChapterNo: a.FromChapterNo,
			ToChapterNo:   a.ToChapterNo,
		})
	}
	return NovelDetailResponse{
		NovelResponse: toNovelResponse(d.Novel),
		AuthorName:    d.Novel.AuthorName,
		Description:   d.Novel.Description,
		Arcs:          arcs,
	}
}

func toChapterListResponses(items []domain.Chapter) []ChapterListResponse {
	out := make([]ChapterListResponse, 0, len(items))
	for _, c := range items {
		out = append(out, ChapterListResponse{
			ID:          c.ID,
			ChapterNo:   c.ChapterNo,
			Title:       c.Title,
			PriceCoins:  c.PriceCoins,
			PublishedAt: c.PublishedAt,
			ArcID:       c.ArcID,
		})
	}
	return out
}

func toChapterViewResponse(v domain.ChapterView) ChapterViewResponse {
	return ChapterViewResponse{
		ID:         v.Chapter.ID,
		NovelID:    v.Chapter.NovelID,
		ChapterNo:  v.Chapter.ChapterNo,
		Title:      v.Chapter.Title,
		PriceCoins: v.Chapter.PriceCoins,
		Locked:     v.Locked,
		BodyHTML:   v.BodyHTML,
	}
}
