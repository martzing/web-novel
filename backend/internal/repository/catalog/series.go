package catalog

import (
	"context"
	"errors"
	"strconv"

	"gorm.io/gorm"

	domain "github.com/mokchan/webnovel-backend/internal/domain/catalog"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

// GetSeries resolves a series by numeric id or slug and loads its books in
// reading order.
func (r *GormRepository) GetSeries(ctx context.Context, idOrSlug string) (*domain.SeriesDetail, error) {
	db := dbctx.From(ctx, r.db)

	var row entities.Series
	query := db.Model(&entities.Series{})
	if id, err := strconv.ParseInt(idOrSlug, 10, 64); err == nil && id > 0 {
		query = query.Where("id = ?", id)
	} else {
		query = query.Where("slug = ?", idOrSlug)
	}
	if err := query.Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	var novels []entities.Novel
	err := db.
		Where("series_id = ? AND status <> ?", row.ID, entities.NovelHidden).
		Order("series_position, id").
		Find(&novels).Error
	if err != nil {
		return nil, err
	}

	out := &domain.SeriesDetail{
		ID:          row.ID,
		Slug:        row.Slug,
		Title:       row.Title,
		Description: derefString(row.Description),
		CoverURL:    derefString(row.CoverURL),
		Books:       make([]domain.SeriesEntry, 0, len(novels)),
	}

	ids := make([]int64, 0, len(novels))
	for _, n := range novels {
		ids = append(ids, n.ID)
	}
	genres, err := r.genresForNovels(ctx, ids)
	if err != nil {
		return nil, err
	}

	for _, n := range novels {
		book := domain.SeriesEntry{
			Novel: toDomainNovel(n),
			Note:  derefString(n.SeriesNote),
		}
		if n.SeriesPosition != nil {
			book.Position = int(*n.SeriesPosition)
		}
		if g := genres[n.ID]; g != nil {
			book.Genres = g
		}
		out.Books = append(out.Books, book)
	}
	return out, nil
}

// RelatedNovels reads both stored directions of novel_relations.
//
// Relations are written once, from the declaring novel's point of view. Reading
// the reverse direction too — with the inverse kind — is what makes a link
// declared on ภาคต่อ show up on ปฐมบท without the translator entering it twice.
func (r *GormRepository) RelatedNovels(ctx context.Context, novelID int64) ([]domain.RelatedNovel, error) {
	db := dbctx.From(ctx, r.db)

	type row struct {
		entities.Novel
		Kind   string
		Note   *string
		SortNo int16
	}

	selectCols := `n.*, rel.kind, rel.note, rel.sort_no`

	var forward []row
	err := db.Table("novel_relations rel").
		Select(selectCols).
		Joins("JOIN novels n ON n.id = rel.related_novel_id").
		Where("rel.novel_id = ? AND n.status <> ?", novelID, entities.NovelHidden).
		Order("rel.sort_no, n.id").
		Scan(&forward).Error
	if err != nil {
		return nil, err
	}

	var backward []row
	err = db.Table("novel_relations rel").
		Select(selectCols).
		Joins("JOIN novels n ON n.id = rel.novel_id").
		Where("rel.related_novel_id = ? AND n.status <> ?", novelID, entities.NovelHidden).
		Order("rel.sort_no, n.id").
		Scan(&backward).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.RelatedNovel, 0, len(forward)+len(backward))
	for _, x := range forward {
		out = append(out, toRelatedNovel(x.Novel, x.Kind, derefString(x.Note), int(x.SortNo)))
	}
	for _, x := range backward {
		kind := domain.InverseRelationKind(x.Kind)
		out = append(out, toRelatedNovel(x.Novel, kind, derefString(x.Note), int(x.SortNo)))
	}
	return out, nil
}

func toRelatedNovel(n entities.Novel, kind, note string, sortNo int) domain.RelatedNovel {
	return domain.RelatedNovel{
		Novel:     toDomainNovel(n),
		Kind:      kind,
		KindLabel: domain.RelationKindLabelTH(kind),
		Note:      note,
		SortNo:    sortNo,
	}
}
