package writer

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	domain "github.com/mokchan/webnovel-backend/internal/domain/writer"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

func (r *GormRepository) ListSeries(ctx context.Context, ownerID int64) ([]domain.Series, error) {
	db := dbctx.From(ctx, r.db)

	var rows []entities.Series
	if err := db.Where("owner_user_id = ?", ownerID).Order("title").Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []domain.Series{}, nil
	}

	// One grouped count rather than a query per series: the work tree renders
	// every series at once.
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	type countRow struct {
		SeriesID int64
		N        int
	}
	var counts []countRow
	err := db.Model(&entities.Novel{}).
		Select("series_id, COUNT(*) AS n").
		Where("series_id IN ?", ids).
		Group("series_id").
		Scan(&counts).Error
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]int, len(counts))
	for _, c := range counts {
		byID[c.SeriesID] = c.N
	}

	out := make([]domain.Series, 0, len(rows))
	for _, row := range rows {
		s := toDomainSeries(row)
		s.BookCount = byID[row.ID]
		out = append(out, s)
	}
	return out, nil
}

func (r *GormRepository) GetSeries(ctx context.Context, id int64) (*domain.Series, error) {
	var row entities.Series
	if err := dbctx.From(ctx, r.db).Where("id = ?", id).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	out := toDomainSeries(row)
	return &out, nil
}

func (r *GormRepository) CreateSeries(ctx context.Context, s domain.Series) (*domain.Series, error) {
	row := entities.Series{
		Slug:        s.Slug,
		Title:       s.Title,
		OwnerUserID: s.OwnerUserID,
	}
	assignSeriesOptional(&row, s)

	if err := dbctx.From(ctx, r.db).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrSlugTaken
		}
		return nil, err
	}
	out := toDomainSeries(row)
	return &out, nil
}

func (r *GormRepository) UpdateSeries(ctx context.Context, id int64, s domain.Series) (*domain.Series, error) {
	updates := map[string]any{}
	if s.Title != "" {
		updates["title"] = s.Title
	}
	if s.Description != "" {
		updates["description"] = s.Description
	}
	if s.CoverURL != "" {
		updates["cover_url"] = s.CoverURL
	}

	if len(updates) > 0 {
		err := dbctx.From(ctx, r.db).Model(&entities.Series{}).
			Where("id = ?", id).Updates(updates).Error
		if err != nil {
			return nil, err
		}
	}
	return r.GetSeries(ctx, id)
}

// DeleteSeries removes the series and detaches its books.
//
// The novels themselves survive: a series is a grouping, and deleting one must
// never take a translator's work with it. novels.series_id is ON DELETE SET
// NULL, but the position and note are ours to clear, or a novel later added to
// a different series would arrive carrying a stale reading-order note.
func (r *GormRepository) DeleteSeries(ctx context.Context, id int64) error {
	return dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&entities.Novel{}).
			Where("series_id = ?", id).
			Updates(map[string]any{
				"series_id":       nil,
				"series_position": 0,
				"series_note":     nil,
			}).Error
		if err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&entities.Series{}).Error
	})
}

func (r *GormRepository) SeriesBooks(ctx context.Context, seriesID int64) ([]domain.SeriesBook, error) {
	var rows []entities.Novel
	err := dbctx.From(ctx, r.db).
		Where("series_id = ?", seriesID).
		// Position first, id as the tiebreak so a series whose order has never
		// been set still lists deterministically.
		Order("series_position, id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.SeriesBook, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.SeriesBook{
			NovelID:             row.ID,
			Position:            derefInt16(row.SeriesPosition),
			Note:                derefString(row.SeriesNote),
			Slug:                row.Slug,
			TitleTH:             row.TitleTH,
			CoverURL:            derefString(row.CoverURL),
			CoverStyle:          row.CoverStyle,
			CoverColor:          derefString(row.CoverColor),
			CoverText:           derefString(row.CoverText),
			Status:              row.Status,
			ChaptersCount:       row.ChaptersCount,
			SourceChaptersCount: row.SourceChaptersCount,
		})
	}
	return out, nil
}

// SetSeriesOrder renumbers a series' books.
//
// Two passes inside one transaction, not one UPDATE. novels_series_position is
// a *partial* unique index, which Postgres cannot defer, so it is enforced row
// by row as a statement runs. Permuting 1,2,3 into 2,3,1 therefore collides
// halfway through even though the final state is perfectly valid. Negating
// every position first moves the whole series out of the way — negatives can
// never clash with the 1..n about to be assigned — and the second pass writes
// the final order.
//
// A CASE expression rather than a row-per-update loop, so a partial application
// can never leave duplicate or missing slots visible on the public series page.
func (r *GormRepository) SetSeriesOrder(ctx context.Context, seriesID int64, positions map[int64]int) error {
	if len(positions) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(positions))
	args := make([]any, 0, len(positions)*2)
	// The literals need an explicit ::smallint: series_position is SMALLINT and
	// a bare parameter infers INTEGER, which Postgres refuses to assign.
	var sql strings.Builder
	sql.WriteString("CASE id")
	for id, pos := range positions {
		ids = append(ids, id)
		sql.WriteString(" WHEN ? THEN ?::smallint")
		args = append(args, id, pos)
	}
	// Anything the caller did not name keeps the position it came in with,
	// undoing the negation from the first pass.
	sql.WriteString(" ELSE -series_position END")

	return dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&entities.Novel{}).
			Where("series_id = ? AND series_position IS NOT NULL", seriesID).
			Update("series_position", gorm.Expr("-series_position")).Error
		if err != nil {
			return err
		}

		return tx.Model(&entities.Novel{}).
			Where("series_id = ?", seriesID).
			Update("series_position", gorm.Expr(sql.String(), args...)).Error
	})
}

func (r *GormRepository) SetSeriesNote(ctx context.Context, novelID int64, note string) error {
	var value any
	if note != "" {
		value = note
	}
	return dbctx.From(ctx, r.db).Model(&entities.Novel{}).
		Where("id = ?", novelID).
		Update("series_note", value).Error
}

// ListRelations returns every relation involving the novel.
//
// Relations are stored once, directional. Reading both directions here is what
// lets a translator declare "B is the sequel of A" from either novel's editor
// and see it from both — with the inverse kind applied to the mirrored side, so
// A does not claim to be its own sequel.
func (r *GormRepository) ListRelations(ctx context.Context, novelID int64) ([]domain.Relation, error) {
	var forward []relationRow
	err := dbctx.From(ctx, r.db).
		Table("novel_relations rel").
		Select(`rel.novel_id, rel.related_novel_id, rel.kind, rel.note, rel.sort_no,
		        n.slug, n.title_th, n.cover_url, n.cover_style, n.cover_color,
		        n.cover_text, n.status`).
		Joins("JOIN novels n ON n.id = rel.related_novel_id").
		Where("rel.novel_id = ?", novelID).
		Order("rel.sort_no, rel.related_novel_id").
		Scan(&forward).Error
	if err != nil {
		return nil, err
	}

	var inverse []relationRow
	err = dbctx.From(ctx, r.db).
		Table("novel_relations rel").
		Select(`rel.novel_id, rel.related_novel_id, rel.kind, rel.note, rel.sort_no,
		        n.slug, n.title_th, n.cover_url, n.cover_style, n.cover_color,
		        n.cover_text, n.status`).
		Joins("JOIN novels n ON n.id = rel.novel_id").
		Where("rel.related_novel_id = ?", novelID).
		Order("rel.sort_no, rel.novel_id").
		Scan(&inverse).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.Relation, 0, len(forward)+len(inverse))
	for _, x := range forward {
		out = append(out, toDomainRelation(novelID, x.RelatedNovelID, x.Kind, x, false))
	}
	for _, x := range inverse {
		out = append(out, toDomainRelation(novelID, x.NovelID, domain.InverseRelationKind(x.Kind), x, true))
	}
	return out, nil
}

func (r *GormRepository) UpsertRelation(ctx context.Context, rel domain.Relation) (*domain.Relation, error) {
	row := entities.NovelRelation{
		NovelID:        rel.NovelID,
		RelatedNovelID: rel.RelatedNovelID,
		Kind:           rel.Kind,
		SortNo:         int16(rel.SortNo),
	}
	if rel.Note != "" {
		note := rel.Note
		row.Note = &note
	}

	err := dbctx.From(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "novel_id"}, {Name: "related_novel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"kind", "note", "sort_no"}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}

	out := rel
	return &out, nil
}

func (r *GormRepository) DeleteRelation(ctx context.Context, novelID, relatedNovelID int64) error {
	// Delete either direction: the translator sees a mirrored relation on the
	// other novel's editor and expects "ปลด" there to remove the same link.
	return dbctx.From(ctx, r.db).
		Where("(novel_id = ? AND related_novel_id = ?) OR (novel_id = ? AND related_novel_id = ?)",
			novelID, relatedNovelID, relatedNovelID, novelID).
		Delete(&entities.NovelRelation{}).Error
}

// relationRow is one novel_relations row joined to the novel on its far side.
// Which side is "far" depends on the direction being read, so the mapper takes
// the resolved id and kind rather than deriving them.
type relationRow struct {
	NovelID        int64
	RelatedNovelID int64
	Kind           string
	Note           *string
	SortNo         int16
	Slug           string
	TitleTH        string
	CoverURL       *string
	CoverStyle     string
	CoverColor     *string
	CoverText      *string
	Status         string
}

func toDomainRelation(novelID, relatedID int64, kind string, row relationRow, mirrored bool) domain.Relation {
	return domain.Relation{
		NovelID:        novelID,
		RelatedNovelID: relatedID,
		Kind:           kind,
		Note:           derefString(row.Note),
		SortNo:         int(row.SortNo),
		// A mirrored relation is shown but not editable from this side; the
		// handler uses Mirrored to hide the unlink control on the wrong novel.
		Mirrored:          mirrored,
		RelatedSlug:       row.Slug,
		RelatedTitleTH:    row.TitleTH,
		RelatedCoverURL:   derefString(row.CoverURL),
		RelatedCoverStyle: row.CoverStyle,
		RelatedCoverColor: derefString(row.CoverColor),
		RelatedCoverText:  derefString(row.CoverText),
		RelatedStatus:     row.Status,
	}
}

func toDomainSeries(row entities.Series) domain.Series {
	return domain.Series{
		ID:          row.ID,
		OwnerUserID: row.OwnerUserID,
		Slug:        row.Slug,
		Title:       row.Title,
		Description: derefString(row.Description),
		CoverURL:    derefString(row.CoverURL),
	}
}

func assignSeriesOptional(row *entities.Series, s domain.Series) {
	if s.Description != "" {
		v := s.Description
		row.Description = &v
	}
	if s.CoverURL != "" {
		v := s.CoverURL
		row.CoverURL = &v
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt16(v *int16) int {
	if v == nil {
		return 0
	}
	return int(*v)
}
