package writer

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	domain "github.com/mokchan/webnovel-backend/internal/domain/writer"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

func (r *GormRepository) ListGlossary(ctx context.Context, novelID int64) ([]domain.GlossaryGroup, error) {
	db := dbctx.From(ctx, r.db)

	var groups []entities.GlossaryGroup
	err := db.Where("novel_id = ?", novelID).Order("sort_no").Order("id").Find(&groups).Error
	if err != nil {
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
		if idx, ok := index[e.GroupID]; ok {
			out[idx].Entries = append(out[idx].Entries, toDomainEntry(e))
		}
	}
	return out, nil
}

func (r *GormRepository) CreateGlossaryGroup(ctx context.Context, g domain.GlossaryGroup) (*domain.GlossaryGroup, error) {
	row := entities.GlossaryGroup{
		NovelID: g.NovelID,
		Name:    g.Name,
		SortNo:  int16(g.SortNo),
	}
	if err := dbctx.From(ctx, r.db).Create(&row).Error; err != nil {
		return nil, err
	}
	return &domain.GlossaryGroup{
		ID:      row.ID,
		NovelID: row.NovelID,
		Name:    row.Name,
		SortNo:  int(row.SortNo),
		Entries: []domain.GlossaryEntry{},
	}, nil
}

func (r *GormRepository) CreateGlossaryEntry(ctx context.Context, e domain.GlossaryEntry) (*domain.GlossaryEntry, error) {
	row := entities.GlossaryEntry{
		GroupID: e.GroupID,
		TermKey: e.TermKey,
		TitleTH: e.TitleTH,
		Body:    e.Body,
	}
	if e.TitleCN != "" {
		v := e.TitleCN
		row.TitleCN = &v
	}
	if e.Kind != "" {
		v := e.Kind
		row.Kind = &v
	}

	if err := dbctx.From(ctx, r.db).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrInvalidInput
		}
		return nil, err
	}
	out := toDomainEntry(row)
	return &out, nil
}

// UpdateGlossaryEntry writes the edit. The glossary_entries trigger bumps
// novels.glossary_rev, which is what makes the re-render worker pick the
// chapter up.
func (r *GormRepository) UpdateGlossaryEntry(ctx context.Context, id int64, e domain.GlossaryEntry) (*domain.GlossaryEntry, error) {
	db := dbctx.From(ctx, r.db)

	updates := map[string]any{}
	if e.TitleTH != "" {
		updates["title_th"] = e.TitleTH
	}
	if e.TitleCN != "" {
		updates["title_cn"] = e.TitleCN
	}
	if e.Body != "" {
		updates["body"] = e.Body
	}
	if e.Kind != "" {
		updates["kind"] = e.Kind
	}
	if e.TermKey != "" {
		updates["term_key"] = e.TermKey
	}

	if len(updates) > 0 {
		res := db.Model(&entities.GlossaryEntry{}).Where("id = ?", id).Updates(updates)
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, domain.ErrNotFound
		}
	}

	var row entities.GlossaryEntry
	if err := db.Where("id = ?", id).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	out := toDomainEntry(row)
	return &out, nil
}

func (r *GormRepository) GlossaryEntryNovelID(ctx context.Context, entryID int64) (int64, error) {
	var row struct{ NovelID int64 }
	err := dbctx.From(ctx, r.db).
		Table("glossary_entries e").
		Select("g.novel_id").
		Joins("JOIN glossary_groups g ON g.id = e.group_id").
		Where("e.id = ?", entryID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, domain.ErrNotFound
		}
		return 0, err
	}
	return row.NovelID, nil
}

// DailyStats reads the pre-aggregated rollup tables rather than the raw event
// stream, which is what keeps the stats page fast.
func (r *GormRepository) DailyStats(ctx context.Context, novelID int64, from, to time.Time) ([]domain.DailyPoint, error) {
	type row struct {
		Day             time.Time
		Reads           int
		CoinsEarned     int
		FollowersGained int
	}
	var rows []row
	err := dbctx.From(ctx, r.db).
		Table("novel_daily_stats").
		Select("day, reads, coins_earned, followers_gained").
		Where("novel_id = ? AND day >= ? AND day < ?", novelID, from, to).
		Order("day").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.DailyPoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.DailyPoint{
			Day:         r.Day,
			Reads:       r.Reads,
			CoinsEarned: r.CoinsEarned,
			Followers:   r.FollowersGained,
		})
	}
	return out, nil
}

func (r *GormRepository) TopChapters(ctx context.Context, novelID int64, from, to time.Time, limit int) ([]domain.ChapterPerformance, error) {
	type row struct {
		ChapterID   int64
		ChapterNo   int
		Title       string
		Reads       int
		CoinsEarned int
	}
	var rows []row
	err := dbctx.From(ctx, r.db).
		Table("chapter_daily_stats s").
		Select(`s.chapter_id, c.chapter_no, c.title,
		        SUM(s.reads) AS reads, SUM(s.coins_earned) AS coins_earned`).
		Joins("JOIN chapters c ON c.id = s.chapter_id").
		Where("c.novel_id = ? AND s.day >= ? AND s.day < ?", novelID, from, to).
		Group("s.chapter_id, c.chapter_no, c.title").
		Order("reads DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.ChapterPerformance, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.ChapterPerformance{
			ChapterID:   r.ChapterID,
			ChapterNo:   r.ChapterNo,
			Title:       r.Title,
			Reads:       r.Reads,
			CoinsEarned: r.CoinsEarned,
		})
	}
	return out, nil
}

func toDomainEntry(row entities.GlossaryEntry) domain.GlossaryEntry {
	out := domain.GlossaryEntry{
		ID:      row.ID,
		GroupID: row.GroupID,
		TermKey: row.TermKey,
		TitleTH: row.TitleTH,
		Body:    row.Body,
	}
	if row.TitleCN != nil {
		out.TitleCN = *row.TitleCN
	}
	if row.Kind != nil {
		out.Kind = *row.Kind
	}
	return out
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
