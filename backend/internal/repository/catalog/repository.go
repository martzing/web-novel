// Package catalog is the GORM adapter for the catalog domain port.
package catalog

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	domain "github.com/mokchan/webnovel-backend/internal/domain/catalog"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

// GormRepository implements domain.Repository against PostgreSQL through GORM.
type GormRepository struct {
	db *gorm.DB
}

// New builds a repository around the shared GORM connection.
func New(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

// Compile-time port check.
var _ domain.Repository = (*GormRepository)(nil)

func (r *GormRepository) ListGenres(ctx context.Context) ([]domain.Genre, error) {
	var rows []entities.Genre
	if err := dbctx.From(ctx, r.db).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Genre, 0, len(rows))
	for _, g := range rows {
		out = append(out, toDomainGenre(g))
	}
	return out, nil
}

func (r *GormRepository) ListNovels(ctx context.Context, filter domain.NovelFilter) ([]domain.Novel, bool, error) {
	db := dbctx.From(ctx, r.db)
	query := db.Model(&entities.Novel{}).Where("status <> ?", entities.NovelHidden)

	if q := strings.TrimSpace(filter.Query); q != "" {
		// Thai has no word boundaries, so `เซียนดาบ` is a substring of the single
		// lexeme `เซียนดาบเก้าสายธาร` and plain tsvector matching misses it.
		// Blend substring matching, trigram similarity and FTS, and rank by all
		// three. escapeLike keeps % and _ from acting as user-supplied wildcards.
		like := "%" + escapeLike(q) + "%"
		query = query.Where(`(
			  title_th ILIKE ? ESCAPE '\'
			   OR title_cn ILIKE ? ESCAPE '\'
			   OR author_name ILIKE ? ESCAPE '\'
			   OR search_tsv @@ plainto_tsquery('simple', ?)
			   OR EXISTS (SELECT 1 FROM glossary_entries e
			                JOIN glossary_groups g ON g.id = e.group_id
			               WHERE g.novel_id = novels.id AND e.title_th ILIKE ? ESCAPE '\')
			)`, like, like, like, q, like)
	}
	if filter.GenreSlug != "" {
		query = query.Where(`EXISTS (
			SELECT 1 FROM novel_genres ng
			  JOIN genres g ON g.id = ng.genre_id
			 WHERE ng.novel_id = novels.id AND g.slug = ?)`, filter.GenreSlug)
	}

	switch {
	case strings.TrimSpace(filter.Query) != "":
		q := strings.TrimSpace(filter.Query)
		query = query.Order(gorm.Expr(`(
			  ts_rank(search_tsv, plainto_tsquery('simple', ?))
			+ similarity(title_th, ?) * 2.0
			+ CASE WHEN title_th ILIKE ? ESCAPE '\' THEN 1.5 ELSE 0 END
		) DESC`, q, q, "%"+escapeLike(q)+"%")).
			Order("followers_count DESC").Order("id DESC")
	case filter.Sort == domain.SortLatest:
		if filter.AfterValue != "" && filter.AfterID > 0 {
			if ts, err := time.Parse(time.RFC3339Nano, filter.AfterValue); err == nil {
				query = query.Where("(updated_at, id) < (?, ?)", ts, filter.AfterID)
			}
		}
		query = query.Order("updated_at DESC").Order("id DESC")
	default:
		if filter.AfterValue != "" && filter.AfterID > 0 {
			query = query.Where("(followers_count, id) < (?, ?)", filter.AfterValue, filter.AfterID)
		}
		query = query.Order("followers_count DESC").Order("id DESC")
	}

	// Fetch one extra row to learn whether another page exists without a COUNT.
	var novels []entities.Novel
	if err := query.Limit(filter.Limit + 1).Find(&novels).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(novels) > filter.Limit
	if hasMore {
		novels = novels[:filter.Limit]
	}

	out := make([]domain.Novel, 0, len(novels))
	byID := map[int64]int{}
	ids := make([]int64, 0, len(novels))
	for _, n := range novels {
		byID[n.ID] = len(out)
		ids = append(ids, n.ID)
		out = append(out, toDomainNovel(n))
	}
	if len(ids) == 0 {
		return out, false, nil
	}

	genres, err := r.genresForNovels(ctx, ids)
	if err != nil {
		return nil, false, err
	}
	for novelID, list := range genres {
		if idx, ok := byID[novelID]; ok {
			out[idx].Genres = list
		}
	}
	return out, hasMore, nil
}

func (r *GormRepository) GetNovelBySlug(ctx context.Context, slug string) (*domain.NovelDetail, error) {
	return r.novelDetail(ctx, "slug = ?", slug)
}

func (r *GormRepository) GetNovelByID(ctx context.Context, id int64) (*domain.NovelDetail, error) {
	return r.novelDetail(ctx, "id = ?", id)
}

func (r *GormRepository) novelDetail(ctx context.Context, where string, arg any) (*domain.NovelDetail, error) {
	db := dbctx.From(ctx, r.db)

	var n entities.Novel
	if err := db.Where(where, arg).First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	detail := &domain.NovelDetail{
		Novel:            toDomainNovel(n),
		SellByArc:        n.SellByArc,
		TipsEnabled:      n.TipsEnabled,
		ReleaseSchedule:  n.ReleaseSchedule,
		EarlyAccessHours: int(n.EarlyAccessHours),
	}

	genres, err := r.genresForNovels(ctx, []int64{n.ID})
	if err != nil {
		return nil, err
	}
	detail.Genres = genres[n.ID]
	if detail.Genres == nil {
		detail.Genres = []domain.Genre{}
	}

	arcs, err := r.ListArcs(ctx, n.ID)
	if err != nil {
		return nil, err
	}
	detail.Arcs = arcs

	var glossaryCount int64
	err = db.Table("glossary_entries e").
		Joins("JOIN glossary_groups g ON g.id = e.group_id").
		Where("g.novel_id = ?", n.ID).
		Count(&glossaryCount).Error
	if err != nil {
		return nil, err
	}
	detail.GlossaryCount = int(glossaryCount)

	var commentsCount int64
	err = db.Table("comments c").
		Joins("JOIN chapters ch ON ch.id = c.chapter_id").
		Where("ch.novel_id = ? AND c.deleted_at IS NULL", n.ID).
		Count(&commentsCount).Error
	if err != nil {
		return nil, err
	}
	detail.CommentsCount = int(commentsCount)

	return detail, nil
}

func (r *GormRepository) ListArcs(ctx context.Context, novelID int64) ([]domain.Arc, error) {
	var arcs []entities.Arc
	err := dbctx.From(ctx, r.db).Where("novel_id = ?", novelID).Order("arc_no").Find(&arcs).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.Arc, 0, len(arcs))
	for _, a := range arcs {
		out = append(out, toDomainArc(a))
	}
	return out, nil
}

func (r *GormRepository) ListChapters(ctx context.Context, novelID int64, limit int) ([]domain.Chapter, error) {
	var chapters []entities.Chapter
	err := dbctx.From(ctx, r.db).
		Where("novel_id = ? AND status = ?", novelID, entities.ChapterPublished).
		Order("chapter_no").
		Limit(limit).
		Find(&chapters).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.Chapter, 0, len(chapters))
	for _, c := range chapters {
		out = append(out, toDomainChapter(c))
	}
	return out, nil
}

func (r *GormRepository) GetGlossary(ctx context.Context, novelID int64) ([]domain.GlossaryGroup, error) {
	db := dbctx.From(ctx, r.db)

	var groups []entities.GlossaryGroup
	if err := db.Where("novel_id = ?", novelID).Order("sort_no").Order("id").Find(&groups).Error; err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return []domain.GlossaryGroup{}, nil
	}

	ids := make([]int64, 0, len(groups))
	index := make(map[int64]int, len(groups))
	out := make([]domain.GlossaryGroup, 0, len(groups))
	for _, g := range groups {
		ids = append(ids, g.ID)
		index[g.ID] = len(out)
		out = append(out, domain.GlossaryGroup{
			ID:      g.ID,
			NovelID: g.NovelID,
			Name:    g.Name,
			SortNo:  int(g.SortNo),
			Entries: []domain.GlossaryEntry{},
		})
	}

	var entries []entities.GlossaryEntry
	if err := db.Where("group_id IN ?", ids).Order("id").Find(&entries).Error; err != nil {
		return nil, err
	}
	for _, e := range entries {
		idx, ok := index[e.GroupID]
		if !ok {
			continue
		}
		out[idx].Entries = append(out[idx].Entries, domain.GlossaryEntry{
			ID:      e.ID,
			GroupID: e.GroupID,
			TermKey: e.TermKey,
			TitleTH: e.TitleTH,
			TitleCN: derefString(e.TitleCN),
			Body:    e.Body,
			Kind:    derefString(e.Kind),
		})
	}
	return out, nil
}

func (r *GormRepository) WeeklyRanking(ctx context.Context, limit int) ([]domain.RankedNovel, error) {
	db := dbctx.From(ctx, r.db)

	type rankRow struct {
		NovelID int64
		Rank    int
		Score   float64
	}
	var ranks []rankRow
	err := db.Table("ranking_snapshots").
		Select("novel_id, rank, score").
		Where("period = (SELECT MAX(period) FROM ranking_snapshots)").
		Order("rank").
		Limit(limit).
		Scan(&ranks).Error
	if err != nil {
		return nil, err
	}

	// No snapshot yet (fresh install, or before the first Monday job run):
	// fall back to live popularity so the home page is never empty.
	if len(ranks) == 0 {
		novels, _, err := r.ListNovels(ctx, domain.NovelFilter{Sort: domain.SortPopular, Limit: limit})
		if err != nil {
			return nil, err
		}
		out := make([]domain.RankedNovel, 0, len(novels))
		for i, n := range novels {
			out = append(out, domain.RankedNovel{Novel: n, Rank: i + 1, Score: float64(n.FollowersCount)})
		}
		return out, nil
	}

	ids := make([]int64, 0, len(ranks))
	for _, r := range ranks {
		ids = append(ids, r.NovelID)
	}
	// A novel hidden after the weekly snapshot was taken must drop out of the
	// chart too, or ซ่อนจากหน้าร้าน leaves it on the busiest page on the site.
	var novels []entities.Novel
	err = db.Where("id IN ? AND status <> ?", ids, entities.NovelHidden).Find(&novels).Error
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]entities.Novel, len(novels))
	for _, n := range novels {
		byID[n.ID] = n
	}
	genres, err := r.genresForNovels(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := make([]domain.RankedNovel, 0, len(ranks))
	for _, rr := range ranks {
		n, ok := byID[rr.NovelID]
		if !ok {
			continue
		}
		item := domain.RankedNovel{Novel: toDomainNovel(n), Rank: rr.Rank, Score: rr.Score}
		if g := genres[n.ID]; g != nil {
			item.Genres = g
		}
		out = append(out, item)
	}
	return out, nil
}

// genresForNovels batch-loads genres for a set of novels, avoiding N+1.
func (r *GormRepository) genresForNovels(ctx context.Context, novelIDs []int64) (map[int64][]domain.Genre, error) {
	type row struct {
		NovelID int64
		ID      int64
		Slug    string
		NameTH  string
	}
	var joined []row
	err := dbctx.From(ctx, r.db).
		Table("novel_genres ng").
		Select("ng.novel_id, g.id, g.slug, g.name_th").
		Joins("JOIN genres g ON g.id = ng.genre_id").
		Where("ng.novel_id IN ?", novelIDs).
		Order("g.id").
		Scan(&joined).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int64][]domain.Genre, len(novelIDs))
	for _, j := range joined {
		out[j.NovelID] = append(out[j.NovelID], domain.Genre{ID: j.ID, Slug: j.Slug, NameTH: j.NameTH})
	}
	return out, nil
}

// escapeLike neutralises the LIKE metacharacters in user input so a query of
// "%" matches a literal percent sign instead of every row.
func escapeLike(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}

func toDomainGenre(g entities.Genre) domain.Genre {
	return domain.Genre{ID: g.ID, Slug: g.Slug, NameTH: g.NameTH}
}

func toDomainNovel(n entities.Novel) domain.Novel {
	return domain.Novel{
		ID:             n.ID,
		Slug:           n.Slug,
		TitleTH:        n.TitleTH,
		TitleCN:        derefString(n.TitleCN),
		AuthorName:     derefString(n.AuthorName),
		Description:    derefString(n.Description),
		CoverURL:       derefString(n.CoverURL),
		Status:         n.Status,
		RatingAvg:      n.RatingAvg,
		RatingCount:    n.RatingCount,
		FollowersCount: n.FollowersCount,
		ChaptersCount:  n.ChaptersCount,
		Genres:         []domain.Genre{},
		UpdatedAt:      n.UpdatedAt.Format(time.RFC3339Nano),

		SourceChaptersCount: n.SourceChaptersCount,
		CoverStyle:          n.CoverStyle,
		CoverColor:          derefString(n.CoverColor),
		CoverText:           derefString(n.CoverText),
		SeriesID:            n.SeriesID,
		PrimaryTranslatorID: n.PrimaryTranslatorID,
	}
}

func toDomainArc(a entities.Arc) domain.Arc {
	return domain.Arc{
		ID:            a.ID,
		NovelID:       a.NovelID,
		ArcNo:         int(a.ArcNo),
		Name:          a.Name,
		FromChapterNo: a.FromChapterNo,
		ToChapterNo:   a.ToChapterNo,
	}
}

func toDomainChapter(c entities.Chapter) domain.Chapter {
	item := domain.Chapter{
		ID:         c.ID,
		NovelID:    c.NovelID,
		ArcID:      c.ArcID,
		ChapterNo:  c.ChapterNo,
		Title:      c.Title,
		PriceCoins: int(c.PriceCoins),
		WordCount:  c.WordCount,
		Unlocked:   c.PriceCoins == 0,
	}
	if c.PublishedAt != nil {
		item.PublishedAt = c.PublishedAt.Format(time.RFC3339)
	}
	return item
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
