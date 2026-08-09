// Package reading is the GORM adapter for the reading domain port.
package reading

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	domain "github.com/mokchan/webnovel-backend/internal/domain/reading"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

// GormRepository implements domain.Repository against PostgreSQL.
type GormRepository struct {
	db *gorm.DB
}

// New builds a repository around the shared GORM connection.
func New(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

var _ domain.Repository = (*GormRepository)(nil)

// chapterRow is the joined projection the reader needs in a single query.
type chapterRow struct {
	ID           int64
	NovelID      int64
	NovelSlug    string
	NovelTitleTH string
	ArcID        *int64
	ArcNo        *int16
	ArcName      *string
	ChapterNo    int
	Title        string
	Status       string
	PriceCoins   int16
	WordCount    int
	TranslatorID *int64
	PublishedAt  *time.Time
	PublicAt     *time.Time
	TipsEnabled  bool
}

func (r *GormRepository) GetChapter(ctx context.Context, id int64, includeUnpublished bool) (*domain.Chapter, error) {
	query := dbctx.From(ctx, r.db).
		Table("chapters c").
		Select(`c.id, c.novel_id, n.slug AS novel_slug, n.title_th AS novel_title_th,
		        c.arc_id, a.arc_no, a.name AS arc_name,
		        c.chapter_no, c.title, c.status, c.price_coins, c.word_count,
		        COALESCE(c.translator_id, n.primary_translator_id) AS translator_id,
		        c.published_at, c.public_at, n.tips_enabled`).
		Joins("JOIN novels n ON n.id = c.novel_id").
		Joins("LEFT JOIN arcs a ON a.id = c.arc_id").
		Where("c.id = ?", id)

	if !includeUnpublished {
		query = query.Where("c.status = ?", entities.ChapterPublished)
	}

	var row chapterRow
	if err := query.Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toDomainChapter(row), nil
}

func (r *GormRepository) GetBody(ctx context.Context, chapterID int64) (string, error) {
	var body entities.ChapterBody
	err := dbctx.From(ctx, r.db).Where("chapter_id = ?", chapterID).First(&body).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return body.BodyHTML, nil
}

func (r *GormRepository) NeighbourIDs(ctx context.Context, novelID int64, chapterNo int) (*int64, *int64, error) {
	db := dbctx.From(ctx, r.db)

	var prev, next *int64
	find := func(order string, cmp string) (*int64, error) {
		var row entities.Chapter
		err := db.Where("novel_id = ? AND status = ? AND chapter_no "+cmp+" ?",
			novelID, entities.ChapterPublished, chapterNo).
			Order("chapter_no " + order).
			Limit(1).
			Take(&row).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, err
		}
		id := row.ID
		return &id, nil
	}

	prev, err := find("DESC", "<")
	if err != nil {
		return nil, nil, err
	}
	next, err = find("ASC", ">")
	if err != nil {
		return nil, nil, err
	}
	return prev, next, nil
}

func (r *GormRepository) GetProgress(ctx context.Context, userID, novelID int64) (*domain.Progress, error) {
	var row entities.ReadingProgress
	err := dbctx.From(ctx, r.db).
		Where("user_id = ? AND novel_id = ?", userID, novelID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	p := toDomainProgress(row)
	return &p, nil
}

func (r *GormRepository) UpsertProgress(ctx context.Context, p domain.Progress) (*domain.Progress, error) {
	row := entities.ReadingProgress{
		UserID:        p.UserID,
		NovelID:       p.NovelID,
		LastChapterID: p.LastChapterID,
		LastChapterNo: p.LastChapterNo,
		ParaAnchor:    p.ParaAnchor,
		Pct:           p.Pct,
		UpdatedAt:     p.UpdatedAt,
	}
	err := dbctx.From(ctx, r.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "novel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"last_chapter_id", "last_chapter_no", "para_anchor", "pct", "updated_at",
		}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	out := toDomainProgress(row)
	return &out, nil
}

func (r *GormRepository) ListProgress(ctx context.Context, userID int64, novelIDs []int64) (map[int64]domain.Progress, error) {
	out := make(map[int64]domain.Progress, len(novelIDs))
	if len(novelIDs) == 0 {
		return out, nil
	}
	var rows []entities.ReadingProgress
	err := dbctx.From(ctx, r.db).
		Where("user_id = ? AND novel_id IN ?", userID, novelIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.NovelID] = toDomainProgress(row)
	}
	return out, nil
}

func (r *GormRepository) InsertReadEvent(ctx context.Context, e domain.ReadEvent) error {
	row := entities.ChapterReadEvent{
		ChapterID:  e.ChapterID,
		UserID:     e.UserID,
		SessionID:  e.SessionID,
		Completed:  e.Completed,
		OccurredAt: e.OccurredAt,
	}
	return dbctx.From(ctx, r.db).Create(&row).Error
}

func toDomainChapter(row chapterRow) *domain.Chapter {
	out := &domain.Chapter{
		ID:           row.ID,
		NovelID:      row.NovelID,
		NovelSlug:    row.NovelSlug,
		NovelTitleTH: row.NovelTitleTH,
		ArcID:        row.ArcID,
		ChapterNo:    row.ChapterNo,
		Title:        row.Title,
		Status:       row.Status,
		PriceCoins:   int(row.PriceCoins),
		WordCount:    row.WordCount,
		TranslatorID: row.TranslatorID,
		TipsEnabled:  row.TipsEnabled,
	}
	if row.ArcNo != nil {
		out.ArcNo = int(*row.ArcNo)
	}
	if row.ArcName != nil {
		out.ArcName = *row.ArcName
	}
	if row.PublishedAt != nil {
		out.PublishedAt = row.PublishedAt.Format(time.RFC3339)
	}
	out.PublishedAtTime = row.PublishedAt
	out.PublicAtTime = row.PublicAt
	return out
}

func toDomainProgress(row entities.ReadingProgress) domain.Progress {
	return domain.Progress{
		UserID:        row.UserID,
		NovelID:       row.NovelID,
		LastChapterID: row.LastChapterID,
		LastChapterNo: row.LastChapterNo,
		ParaAnchor:    row.ParaAnchor,
		Pct:           row.Pct,
		UpdatedAt:     row.UpdatedAt,
	}
}
