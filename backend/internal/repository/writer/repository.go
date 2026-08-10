// Package writer is the GORM adapter for the translator workspace.
package writer

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
	domain "github.com/mokchan/webnovel-backend/internal/domain/writer"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/glossaryrender"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

// GormRepository implements domain.Repository.
type GormRepository struct {
	db *gorm.DB
}

// New builds a repository around the shared GORM connection.
func New(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

var _ domain.Repository = (*GormRepository)(nil)

func (r *GormRepository) OwnsNovel(ctx context.Context, userID, novelID int64) (bool, error) {
	var count int64
	err := dbctx.From(ctx, r.db).Model(&entities.Novel{}).
		Where("id = ? AND primary_translator_id = ?", novelID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *GormRepository) OwnsChapter(ctx context.Context, userID, chapterID int64) (bool, error) {
	var count int64
	err := dbctx.From(ctx, r.db).
		Table("chapters c").
		Joins("JOIN novels n ON n.id = c.novel_id").
		Where("c.id = ? AND (c.translator_id = ? OR n.primary_translator_id = ?)", chapterID, userID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *GormRepository) CreateNovel(ctx context.Context, n domain.NovelDraft, ownerID int64) (*domain.NovelDraft, error) {
	db := dbctx.From(ctx, r.db)

	row := entities.Novel{
		Slug:                n.Slug,
		TitleTH:             n.TitleTH,
		Status:              n.Status,
		PrimaryTranslatorID: &ownerID,
		SeriesID:            n.SeriesID,
	}
	assignOptional(&row, n)

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return replaceGenres(tx, row.ID, n.GenreIDs)
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrSlugTaken
		}
		return nil, err
	}
	return r.GetNovel(ctx, row.ID)
}

func (r *GormRepository) UpdateNovel(ctx context.Context, id int64, n domain.NovelDraft) (*domain.NovelDraft, error) {
	updates := map[string]any{}
	if n.TitleTH != "" {
		updates["title_th"] = n.TitleTH
	}
	if n.TitleCN != "" {
		updates["title_cn"] = n.TitleCN
	}
	if n.AuthorName != "" {
		updates["author_name"] = n.AuthorName
	}
	if n.Description != "" {
		updates["description"] = n.Description
	}
	if n.Status != "" {
		updates["status"] = n.Status
	}
	if n.ReleaseSchedule != "" {
		updates["release_schedule"] = n.ReleaseSchedule
	}
	if n.CoverStyle != "" {
		updates["cover_style"] = n.CoverStyle
	}
	if n.CoverColor != "" {
		updates["cover_color"] = n.CoverColor
	}
	if n.CoverText != "" {
		updates["cover_text"] = n.CoverText
	}
	if n.SeriesNote != "" {
		updates["series_note"] = n.SeriesNote
	}

	// The pointer settings apply whenever they are supplied, including their
	// zero values: "free until chapter 0" and "arc sales off" are exactly the
	// edits a translator makes, and a non-zero test would silently drop them.
	if n.SourceChaptersCount != nil {
		updates["source_chapters_count"] = *n.SourceChaptersCount
	}
	if n.PricePerChapter != nil {
		updates["price_per_chapter"] = *n.PricePerChapter
	}
	if n.FreeUntilChapter != nil {
		updates["free_until_chapter"] = *n.FreeUntilChapter
	}
	if n.SellByArc != nil {
		updates["sell_by_arc"] = *n.SellByArc
	}
	if n.TipsEnabled != nil {
		updates["tips_enabled"] = *n.TipsEnabled
	}
	if n.EarlyAccessHours != nil {
		updates["early_access_hours"] = *n.EarlyAccessHours
	}
	if n.SeriesPosition != nil {
		updates["series_position"] = *n.SeriesPosition
	}
	// SeriesIDSet separates "leave the series alone" from "remove it", which a
	// nil SeriesID alone cannot express.
	if n.SeriesIDSet {
		updates["series_id"] = n.SeriesID
		if n.SeriesID == nil {
			// Leaving a series drops its reading-order slot with it, or the
			// novel rejoins a different series carrying a stale position.
			updates["series_position"] = nil
			updates["series_note"] = nil
		}
	}

	err := dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&entities.Novel{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}
		if n.GenreIDs != nil {
			return replaceGenres(tx, id, n.GenreIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetNovel(ctx, id)
}

func (r *GormRepository) GetNovel(ctx context.Context, id int64) (*domain.NovelDraft, error) {
	db := dbctx.From(ctx, r.db)

	var row entities.Novel
	if err := db.Where("id = ?", id).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	var genreIDs []int64
	err := db.Model(&entities.NovelGenre{}).
		Where("novel_id = ?", id).
		Order("genre_id").
		Pluck("genre_id", &genreIDs).Error
	if err != nil {
		return nil, err
	}

	out := toDomainNovel(row)
	out.GenreIDs = genreIDs
	return &out, nil
}

func (r *GormRepository) ListNovels(ctx context.Context, ownerID int64, p page.Page) ([]domain.NovelDraft, string, error) {
	query := dbctx.From(ctx, r.db).Model(&entities.Novel{}).
		Where("primary_translator_id = ?", ownerID)

	cursor, err := httpx.DecodeCursorFor(p.Cursor, "writer_novels")
	if err != nil {
		return nil, "", err
	}
	if len(cursor.Keys) == 1 {
		id, parseErr := strconv.ParseInt(cursor.Keys[0], 10, 64)
		if parseErr != nil {
			return nil, "", httpx.ErrBadCursor
		}
		query = query.Where("id < ?", id)
	}

	var rows []entities.Novel
	if err := query.Order("id DESC").Limit(p.Limit + 1).Find(&rows).Error; err != nil {
		return nil, "", err
	}

	next := ""
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		next = httpx.EncodeCursor(httpx.Cursor{
			Sort: "writer_novels",
			Keys: []string{strconv.FormatInt(rows[len(rows)-1].ID, 10)},
		})
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	genres, err := r.genreIDsForNovels(ctx, ids)
	if err != nil {
		return nil, "", err
	}

	out := make([]domain.NovelDraft, 0, len(rows))
	for _, row := range rows {
		draft := toDomainNovel(row)
		// Without this the works tree hands the editor a novel whose genres
		// look empty, so the chips render unselected and a save that includes
		// them replaces the real set with nothing.
		draft.GenreIDs = genres[row.ID]
		out = append(out, draft)
	}
	return out, next, nil
}

// genreIDsForNovels loads the genre links for a whole page in one query. A
// lookup per novel would make listing the works tree cost grow with the number
// of works a translator has.
func (r *GormRepository) genreIDsForNovels(ctx context.Context, novelIDs []int64) (map[int64][]int64, error) {
	out := make(map[int64][]int64, len(novelIDs))
	if len(novelIDs) == 0 {
		return out, nil
	}

	var links []entities.NovelGenre
	err := dbctx.From(ctx, r.db).
		Where("novel_id IN ?", novelIDs).
		Order("novel_id, genre_id").
		Find(&links).Error
	if err != nil {
		return nil, err
	}
	for _, link := range links {
		out[link.NovelID] = append(out[link.NovelID], link.GenreID)
	}
	return out, nil
}

func (r *GormRepository) SetCoverURL(ctx context.Context, novelID int64, url string) error {
	return dbctx.From(ctx, r.db).Model(&entities.Novel{}).
		Where("id = ?", novelID).
		Update("cover_url", url).Error
}

func (r *GormRepository) ListArcs(ctx context.Context, novelID int64) ([]domain.Arc, error) {
	var rows []entities.Arc
	err := dbctx.From(ctx, r.db).Where("novel_id = ?", novelID).Order("arc_no").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.Arc, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainArc(row))
	}
	return out, nil
}

func (r *GormRepository) CreateArc(ctx context.Context, a domain.Arc) (*domain.Arc, error) {
	row := entities.Arc{
		NovelID:       a.NovelID,
		ArcNo:         int16(a.ArcNo),
		Name:          a.Name,
		FromChapterNo: a.FromChapterNo,
		ToChapterNo:   a.ToChapterNo,
	}
	if err := dbctx.From(ctx, r.db).Create(&row).Error; err != nil {
		return nil, err
	}
	out := toDomainArc(row)
	return &out, nil
}

func (r *GormRepository) UpdateArc(ctx context.Context, id int64, a domain.Arc) (*domain.Arc, error) {
	updates := map[string]any{}
	if a.Name != "" {
		updates["name"] = a.Name
	}
	if a.FromChapterNo > 0 {
		updates["from_chapter_no"] = a.FromChapterNo
	}
	if a.ToChapterNo > 0 {
		updates["to_chapter_no"] = a.ToChapterNo
	}
	if a.ArcNo > 0 {
		updates["arc_no"] = a.ArcNo
	}

	db := dbctx.From(ctx, r.db)
	if len(updates) > 0 {
		if err := db.Model(&entities.Arc{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	var row entities.Arc
	if err := db.Where("id = ?", id).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	out := toDomainArc(row)
	return &out, nil
}

func (r *GormRepository) ArcNovelID(ctx context.Context, arcID int64) (int64, error) {
	var row entities.Arc
	if err := dbctx.From(ctx, r.db).Where("id = ?", arcID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, domain.ErrNotFound
		}
		return 0, err
	}
	return row.NovelID, nil
}

type chapterRow struct {
	entities.Chapter
	BodySource string
	BodyHTML   string
}

func (r *GormRepository) ListChapters(ctx context.Context, novelID int64, p page.Page) ([]domain.Chapter, string, error) {
	query := dbctx.From(ctx, r.db).
		Table("chapters c").
		Select("c.*, COALESCE(cb.body_source, '') AS body_source, COALESCE(cb.body_html, '') AS body_html").
		Joins("LEFT JOIN chapter_bodies cb ON cb.chapter_id = c.id").
		Where("c.novel_id = ?", novelID)

	cursor, err := httpx.DecodeCursorFor(p.Cursor, "writer_chapters")
	if err != nil {
		return nil, "", err
	}
	if len(cursor.Keys) == 1 {
		no, parseErr := strconv.Atoi(cursor.Keys[0])
		if parseErr != nil {
			return nil, "", httpx.ErrBadCursor
		}
		query = query.Where("c.chapter_no < ?", no)
	}

	var rows []chapterRow
	if err := query.Order("c.chapter_no DESC").Limit(p.Limit + 1).Scan(&rows).Error; err != nil {
		return nil, "", err
	}

	next := ""
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		next = httpx.EncodeCursor(httpx.Cursor{
			Sort: "writer_chapters",
			Keys: []string{strconv.Itoa(rows[len(rows)-1].ChapterNo)},
		})
	}

	out := make([]domain.Chapter, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainChapter(row))
	}
	return out, next, nil
}

func (r *GormRepository) GetChapter(ctx context.Context, id int64) (*domain.Chapter, error) {
	var row chapterRow
	err := dbctx.From(ctx, r.db).
		Table("chapters c").
		Select("c.*, COALESCE(cb.body_source, '') AS body_source, COALESCE(cb.body_html, '') AS body_html").
		Joins("LEFT JOIN chapter_bodies cb ON cb.chapter_id = c.id").
		Where("c.id = ?", id).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	out := toDomainChapter(row)
	return &out, nil
}

func (r *GormRepository) CreateChapter(ctx context.Context, c domain.Chapter) (*domain.Chapter, error) {
	row := entities.Chapter{
		NovelID:    c.NovelID,
		ArcID:      c.ArcID,
		ChapterNo:  c.ChapterNo,
		Title:      c.Title,
		Status:     entities.ChapterDraftStatus,
		PriceCoins: int16(c.PriceCoins),
		WordCount:  c.WordCount,
	}

	err := dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		// The chapter's own translator defaults to the novel's.
		var novel entities.Novel
		if err := tx.Where("id = ?", c.NovelID).Take(&novel).Error; err != nil {
			return err
		}
		row.TranslatorID = novel.PrimaryTranslatorID

		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Create(&entities.ChapterBody{
			ChapterID:  row.ID,
			BodyHTML:   "",
			BodySource: c.BodySource,
			Revision:   1,
		}).Error
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrChapterNoTaken
		}
		return nil, err
	}
	return r.GetChapter(ctx, row.ID)
}

// SaveChapter writes the edit, appends an autosave revision and prunes the
// history, all in one transaction so the chapter and its history cannot drift.
func (r *GormRepository) SaveChapter(ctx context.Context, c domain.Chapter, authorID int64, keepRevisions int) (*domain.Chapter, error) {
	err := dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"title":       c.Title,
			"price_coins": int16(c.PriceCoins),
			"word_count":  c.WordCount,
			"arc_id":      c.ArcID,
			"chapter_no":  c.ChapterNo,
			"updated_at":  time.Now(),
		}
		if err := tx.Model(&entities.Chapter{}).Where("id = ?", c.ID).Updates(updates).Error; err != nil {
			return err
		}

		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chapter_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"body_source", "updated_at"}),
		}).Create(&entities.ChapterBody{
			ChapterID:  c.ID,
			BodyHTML:   c.BodyHTML,
			BodySource: c.BodySource,
			Revision:   1,
		}).Error
		if err != nil {
			return err
		}

		if err := tx.Create(&entities.ChapterDraft{
			ChapterID:  c.ID,
			AuthorID:   authorID,
			BodySource: c.BodySource,
		}).Error; err != nil {
			return err
		}
		return pruneRevisions(tx, c.ID, keepRevisions)
	})
	if err != nil {
		return nil, err
	}
	return r.GetChapter(ctx, c.ID)
}

// pruneRevisions keeps only the newest `keep` autosaves for a chapter.
func pruneRevisions(tx *gorm.DB, chapterID int64, keep int) error {
	if keep <= 0 {
		return nil
	}
	var revs []entities.ChapterDraft
	err := tx.Select("id", "saved_at").
		Where("chapter_id = ?", chapterID).
		Order("saved_at DESC").Order("id DESC").
		Find(&revs).Error
	if err != nil {
		return err
	}
	if len(revs) <= keep {
		return nil
	}

	doomed := make([]int64, 0, len(revs)-keep)
	for _, rev := range revs[keep:] {
		doomed = append(doomed, rev.ID)
	}
	return tx.Where("id IN ?", doomed).Delete(&entities.ChapterDraft{}).Error
}

func (r *GormRepository) CountRevisions(ctx context.Context, chapterID int64) (int, error) {
	var count int64
	err := dbctx.From(ctx, r.db).Model(&entities.ChapterDraft{}).
		Where("chapter_id = ?", chapterID).
		Count(&count).Error
	return int(count), err
}

// PublishChapter renders the source against the novel's glossary and stamps the
// status, all in one transaction.
//
// The glossary revision written is the one actually rendered against; the
// glossary_entries trigger bumps novels.glossary_rev per row, so a bulk edit
// landing mid-render must not leave the body marked fresher than it is.
func (r *GormRepository) PublishChapter(ctx context.Context, chapterID int64, status string, publishedAt, scheduledAt *time.Time) (*domain.Chapter, error) {
	err := dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		var chapter entities.Chapter
		if err := tx.Where("id = ?", chapterID).Take(&chapter).Error; err != nil {
			return err
		}

		var novel entities.Novel
		if err := tx.Where("id = ?", chapter.NovelID).Take(&novel).Error; err != nil {
			return err
		}

		var body entities.ChapterBody
		if err := tx.Where("chapter_id = ?", chapterID).Take(&body).Error; err != nil {
			return err
		}

		terms, err := loadTerms(tx, chapter.NovelID)
		if err != nil {
			return err
		}
		rendered := glossaryrender.Render(body.BodySource, terms)

		err = tx.Model(&entities.ChapterBody{}).
			Where("chapter_id = ?", chapterID).
			Updates(map[string]any{
				"body_html":    rendered.HTML,
				"glossary_rev": novel.GlossaryRev,
				"revision":     gorm.Expr("revision + 1"),
				"rendered_at":  time.Now(),
			}).Error
		if err != nil {
			return err
		}

		if err := replaceGlossaryRefs(tx, chapterID, rendered.EntryIDs); err != nil {
			return err
		}

		updates := map[string]any{
			"status":       status,
			"published_at": publishedAt,
			"scheduled_at": scheduledAt,
			"updated_at":   time.Now(),
		}
		// Snapshot when non-subscribers may read it. Deriving this at read time
		// would let a later settings change retroactively hide chapters that
		// readers can already see.
		if publishedAt != nil {
			updates["public_at"] = domain.PublicAt(*publishedAt, int(novel.EarlyAccessHours))
		} else {
			updates["public_at"] = nil
		}
		if err := tx.Model(&entities.Chapter{}).Where("id = ?", chapterID).Updates(updates).Error; err != nil {
			return err
		}
		return refreshChapterCount(tx, chapter.NovelID)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.GetChapter(ctx, chapterID)
}

func (r *GormRepository) UnpublishChapter(ctx context.Context, chapterID int64) (*domain.Chapter, error) {
	err := dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		var chapter entities.Chapter
		if err := tx.Where("id = ?", chapterID).Take(&chapter).Error; err != nil {
			return err
		}
		err := tx.Model(&entities.Chapter{}).Where("id = ?", chapterID).
			Updates(map[string]any{
				"status":       entities.ChapterDraftStatus,
				"published_at": nil,
				"scheduled_at": nil,
				"public_at":    nil,
			}).Error
		if err != nil {
			return err
		}
		return refreshChapterCount(tx, chapter.NovelID)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.GetChapter(ctx, chapterID)
}

func (r *GormRepository) NextSlugSeq(ctx context.Context) (int64, error) {
	var seq int64
	err := dbctx.From(ctx, r.db).Raw("SELECT COALESCE(MAX(id), 0) + 1 FROM novels").Scan(&seq).Error
	return seq, err
}

func loadTerms(tx *gorm.DB, novelID int64) ([]glossaryrender.Term, error) {
	type termRow struct {
		ID      int64
		TermKey string
		TitleTH string
	}
	var rows []termRow
	err := tx.Table("glossary_entries e").
		Select("e.id, e.term_key, e.title_th").
		Joins("JOIN glossary_groups g ON g.id = e.group_id").
		Where("g.novel_id = ?", novelID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]glossaryrender.Term, 0, len(rows))
	for _, row := range rows {
		out = append(out, glossaryrender.Term{EntryID: row.ID, Key: row.TermKey, TitleTH: row.TitleTH})
	}
	return out, nil
}

func replaceGlossaryRefs(tx *gorm.DB, chapterID int64, entryIDs []int64) error {
	if err := tx.Where("chapter_id = ?", chapterID).Delete(&entities.ChapterGlossaryRef{}).Error; err != nil {
		return err
	}
	if len(entryIDs) == 0 {
		return nil
	}
	refs := make([]entities.ChapterGlossaryRef, 0, len(entryIDs))
	for _, id := range entryIDs {
		refs = append(refs, entities.ChapterGlossaryRef{ChapterID: chapterID, EntryID: id})
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&refs).Error
}

// refreshChapterCount keeps novels.chapters_count in step with reality rather
// than incrementing it, so republishing cannot inflate the number.
func refreshChapterCount(tx *gorm.DB, novelID int64) error {
	return tx.Exec(`
		UPDATE novels
		   SET chapters_count = (SELECT COUNT(*) FROM chapters
		                          WHERE novel_id = ? AND status = 'published')
		 WHERE id = ?`, novelID, novelID).Error
}

func replaceGenres(tx *gorm.DB, novelID int64, genreIDs []int64) error {
	if err := tx.Where("novel_id = ?", novelID).Delete(&entities.NovelGenre{}).Error; err != nil {
		return err
	}
	if len(genreIDs) == 0 {
		return nil
	}
	links := make([]entities.NovelGenre, 0, len(genreIDs))
	for _, id := range genreIDs {
		links = append(links, entities.NovelGenre{NovelID: novelID, GenreID: id})
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&links).Error
}

func assignOptional(row *entities.Novel, n domain.NovelDraft) {
	if n.TitleCN != "" {
		v := n.TitleCN
		row.TitleCN = &v
	}
	if n.AuthorName != "" {
		v := n.AuthorName
		row.AuthorName = &v
	}
	if n.Description != "" {
		v := n.Description
		row.Description = &v
	}
	if n.CoverURL != "" {
		v := n.CoverURL
		row.CoverURL = &v
	}
	if n.CoverStyle != "" {
		row.CoverStyle = n.CoverStyle
	}
	if n.CoverColor != "" {
		v := n.CoverColor
		row.CoverColor = &v
	}
	if n.CoverText != "" {
		v := n.CoverText
		row.CoverText = &v
	}
	if n.ReleaseSchedule != "" {
		row.ReleaseSchedule = n.ReleaseSchedule
	}
	if n.SourceChaptersCount != nil {
		row.SourceChaptersCount = *n.SourceChaptersCount
	}
	if n.PricePerChapter != nil {
		row.PricePerChapter = int16(*n.PricePerChapter)
	}
	if n.FreeUntilChapter != nil {
		row.FreeUntilChapter = *n.FreeUntilChapter
	}
	if n.SellByArc != nil {
		row.SellByArc = *n.SellByArc
	}
	if n.TipsEnabled != nil {
		row.TipsEnabled = *n.TipsEnabled
	}
	if n.EarlyAccessHours != nil {
		row.EarlyAccessHours = int16(*n.EarlyAccessHours)
	}
}

func toDomainNovel(row entities.Novel) domain.NovelDraft {
	out := domain.NovelDraft{
		ID:       row.ID,
		Slug:     row.Slug,
		TitleTH:  row.TitleTH,
		Status:   row.Status,
		SeriesID: row.SeriesID,
		GenreIDs: []int64{},
	}
	if row.TitleCN != nil {
		out.TitleCN = *row.TitleCN
	}
	if row.AuthorName != nil {
		out.AuthorName = *row.AuthorName
	}
	if row.Description != nil {
		out.Description = *row.Description
	}
	if row.CoverURL != nil {
		out.CoverURL = *row.CoverURL
	}

	sourceCount := row.SourceChaptersCount
	price := int(row.PricePerChapter)
	freeUntil := row.FreeUntilChapter
	sellByArc := row.SellByArc
	tips := row.TipsEnabled
	earlyHours := int(row.EarlyAccessHours)

	out.SourceChaptersCount = &sourceCount
	out.PricePerChapter = &price
	out.FreeUntilChapter = &freeUntil
	out.SellByArc = &sellByArc
	out.TipsEnabled = &tips
	out.EarlyAccessHours = &earlyHours
	out.ReleaseSchedule = row.ReleaseSchedule
	out.CoverStyle = row.CoverStyle
	out.CoverColor = derefString(row.CoverColor)
	out.CoverText = derefString(row.CoverText)
	out.SeriesNote = derefString(row.SeriesNote)
	out.ChaptersCount = row.ChaptersCount
	out.Title = row.TitleTH
	if row.SeriesPosition != nil {
		pos := int(*row.SeriesPosition)
		out.SeriesPosition = &pos
	}
	return out
}

func toDomainArc(row entities.Arc) domain.Arc {
	return domain.Arc{
		ID:            row.ID,
		NovelID:       row.NovelID,
		ArcNo:         int(row.ArcNo),
		Name:          row.Name,
		FromChapterNo: row.FromChapterNo,
		ToChapterNo:   row.ToChapterNo,
	}
}

func toDomainChapter(row chapterRow) domain.Chapter {
	return domain.Chapter{
		ID:          row.ID,
		NovelID:     row.NovelID,
		ArcID:       row.ArcID,
		ChapterNo:   row.ChapterNo,
		Title:       row.Title,
		BodySource:  row.BodySource,
		BodyHTML:    row.BodyHTML,
		PriceCoins:  int(row.PriceCoins),
		WordCount:   row.WordCount,
		Status:      row.Status,
		ScheduledAt: row.ScheduledAt,
		PublishedAt: row.PublishedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
