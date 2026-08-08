package catalog
// Package catalog is the GORM adapter for the catalog domain port.
package catalog

import (
	"context"
	"errors"

	"gorm.io/gorm"

	domain "github.com/mokchan/webnovel-backend/internal/domain/catalog"
	"github.com/mokchan/webnovel-backend/internal/entities"
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
	if err := r.db.WithContext(ctx).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Genre, 0, len(rows))
	for _, g := range rows {
		out = append(out, toDomainGenre(g))
	}
	return out, nil
}

func (r *GormRepository) ListNovels(ctx context.Context, filter domain.NovelFilter) ([]domain.Novel, error) {
	query := r.db.WithContext(ctx).Model(&entities.Novel{})
	if filter.Query != "" {
		like := "%" + filter.Query + "%"
		query = query.Where("title_th ILIKE ? OR title_cn ILIKE ?", like, like)
	}
	if filter.GenreSlug != "" {
		query = query.Where(`EXISTS (
			SELECT 1 FROM novel_genres ng
			  JOIN genres g ON g.id = ng.genre_id
			 WHERE ng.novel_id = novels.id AND g.slug = ?)`, filter.GenreSlug)
	}
	if filter.Sort == "latest" {
		query = query.Order("updated_at DESC")
	} else {
		query = query.Order("followers_count DESC").Order("rating_avg DESC")
	}

	var novels []entities.Novel
	if err := query.Limit(filter.Limit).Find(&novels).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Novel, 0, len(novels))
	byID := map[int64]int{}
	ids := make([]int64, 0, len(novels))
	for _, n := range novels {
		item := toDomainNovel(n)
		byID[n.ID] = len(out)
		ids = append(ids, n.ID)
		out = append(out, item)
	}
	if len(ids) == 0 {
		return out, nil
	}

	type row struct {
		NovelID int64
		ID      int64
		Slug    string
		NameTH  string
	}
	var joined []row
	err := r.db.WithContext(ctx).
		Table("novel_genres ng").
		Select("ng.novel_id, g.id, g.slug, g.name_th").
		Joins("JOIN genres g ON g.id = ng.genre_id").
		Where("ng.novel_id IN ?", ids).
		Order("g.id").
		Scan(&joined).Error
	if err != nil {
		return nil, err
	}
	for _, j := range joined {
		if idx, ok := byID[j.NovelID]; ok {
			out[idx].Genres = append(out[idx].Genres, domain.Genre{ID: j.ID, Slug: j.Slug, NameTH: j.NameTH})
		}
	}
	return out, nil
}

func (r *GormRepository) GetNovelBySlug(ctx context.Context, slug string) (*domain.NovelDetail, error) {
	var n entities.Novel
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&n).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	detail := &domain.NovelDetail{Novel: toDomainNovel(n)}

	var genres []entities.Genre
	err = r.db.WithContext(ctx).
		Table("genres").
		Select("genres.id, genres.slug, genres.name_th").
		Joins("JOIN novel_genres ng ON ng.genre_id = genres.id").
		Where("ng.novel_id = ?", n.ID).
		Order("genres.id").
		Scan(&genres).Error
	if err != nil {
		return nil, err
	}
	detail.Genres = make([]domain.Genre, 0, len(genres))
	for _, g := range genres {
		detail.Genres = append(detail.Genres, toDomainGenre(g))
	}

	var arcs []entities.Arc
	if err := r.db.WithContext(ctx).Where("novel_id = ?", n.ID).Order("arc_no").Find(&arcs).Error; err != nil {
		return nil, err
	}
	detail.Arcs = make([]domain.Arc, 0, len(arcs))
	for _, a := range arcs {
		detail.Arcs = append(detail.Arcs, toDomainArc(a))
	}
	return detail, nil
}

func (r *GormRepository) ListChapters(ctx context.Context, novelID int64, limit int) ([]domain.Chapter, error) {
	var chapters []entities.Chapter
	err := r.db.WithContext(ctx).
		Where("novel_id = ? AND status = ?", novelID, "published").
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

func (r *GormRepository) GetChapter(ctx context.Context, id int64) (*domain.Chapter, error) {
	var c entities.Chapter
	err := r.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, "published").
		First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	dc := toDomainChapter(c)
	return &dc, nil
}

func (r *GormRepository) GetChapterBody(ctx context.Context, chapterID int64) (string, error) {
	var body entities.ChapterBody
	err := r.db.WithContext(ctx).Where("chapter_id = ?", chapterID).First(&body).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return body.BodyHTML, nil
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
	}
	if c.PublishedAt != nil {
		item.PublishedAt = c.PublishedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return item
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
