// Package library is the GORM adapter for the reader's shelf.
package library

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	domain "github.com/mokchan/webnovel-backend/internal/domain/library"
	"github.com/mokchan/webnovel-backend/internal/domain/page"
	"github.com/mokchan/webnovel-backend/internal/entities"
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

type shelfRow struct {
	UserID              int64
	NovelID             int64
	Status              string
	AddedAt             string
	Slug                string
	TitleTH             string
	TitleCN             *string
	CoverURL            *string
	ChaptersCount       int
	SourceChaptersCount int
	LastChapterNo       *int
	LastChapterID       *int64
	Pct                 *float64
	CoverStyle          string
	CoverColor          *string
	CoverText           *string
}

func (r *GormRepository) ListShelf(ctx context.Context, userID int64, tab string, p page.Page) ([]domain.EntryWithNovel, string, error) {
	query := dbctx.From(ctx, r.db).
		Table("library_entries le").
		Select(`le.user_id, le.novel_id, le.status, le.added_at,
		        n.slug, n.title_th, n.title_cn, n.cover_url, n.chapters_count,
		        n.source_chapters_count, n.cover_style, n.cover_color, n.cover_text,
		        rp.last_chapter_no, rp.last_chapter_id, rp.pct`).
		Joins("JOIN novels n ON n.id = le.novel_id").
		Joins("LEFT JOIN reading_progress rp ON rp.user_id = le.user_id AND rp.novel_id = le.novel_id").
		Where("le.user_id = ?", userID)

	if tab != "" {
		query = query.Where("le.status = ?", tab)
	}

	cursor, err := httpx.DecodeCursorFor(p.Cursor, "shelf")
	if err != nil {
		return nil, "", err
	}
	if len(cursor.Keys) == 1 {
		novelID, parseErr := strconv.ParseInt(cursor.Keys[0], 10, 64)
		if parseErr != nil {
			return nil, "", httpx.ErrBadCursor
		}
		query = query.Where("le.novel_id > ?", novelID)
	}

	var rows []shelfRow
	if err := query.Order("le.novel_id").Limit(p.Limit + 1).Scan(&rows).Error; err != nil {
		return nil, "", err
	}

	next := ""
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		next = httpx.EncodeCursor(httpx.Cursor{
			Sort: "shelf",
			Keys: []string{strconv.FormatInt(rows[len(rows)-1].NovelID, 10)},
		})
	}

	out := make([]domain.EntryWithNovel, 0, len(rows))
	for _, row := range rows {
		item := domain.EntryWithNovel{
			Entry: domain.Entry{
				UserID:  row.UserID,
				NovelID: row.NovelID,
				Status:  row.Status,
			},
			Slug:                row.Slug,
			TitleTH:             row.TitleTH,
			ChaptersCount:       row.ChaptersCount,
			SourceChaptersCount: row.SourceChaptersCount,
			LastChapterNo:       row.LastChapterNo,
			LastChapterID:       row.LastChapterID,
			CoverStyle:          row.CoverStyle,
		}
		if row.CoverColor != nil {
			item.CoverColor = *row.CoverColor
		}
		if row.CoverText != nil {
			item.CoverText = *row.CoverText
		}
		if row.TitleCN != nil {
			item.TitleCN = *row.TitleCN
		}
		if row.CoverURL != nil {
			item.CoverURL = *row.CoverURL
		}
		if row.Pct != nil {
			item.Pct = *row.Pct
		}
		out = append(out, item)
	}
	return out, next, nil
}

func (r *GormRepository) CountShelf(ctx context.Context, userID int64) (domain.Counts, error) {
	var rows []struct {
		Status string
		Total  int
	}
	err := dbctx.From(ctx, r.db).
		Table("library_entries").
		Select("status, COUNT(*) AS total").
		Where("user_id = ?", userID).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return domain.Counts{}, err
	}

	var counts domain.Counts
	for _, row := range rows {
		switch row.Status {
		case domain.StatusReading:
			counts.Reading = row.Total
		case domain.StatusSaved:
			counts.Saved = row.Total
		case domain.StatusDone:
			counts.Done = row.Total
		}
	}
	return counts, nil
}

func (r *GormRepository) UpsertShelf(ctx context.Context, e domain.Entry) (*domain.Entry, error) {
	row := entities.LibraryEntry{
		UserID:  e.UserID,
		NovelID: e.NovelID,
		Status:  e.Status,
		AddedAt: e.AddedAt,
	}
	// Moving between tabs updates the status but keeps the original added_at,
	// so the shelf ordering does not jump when a reader marks a novel done.
	err := dbctx.From(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "novel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status"}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	return &domain.Entry{
		UserID:  row.UserID,
		NovelID: row.NovelID,
		Status:  row.Status,
		AddedAt: row.AddedAt,
	}, nil
}

func (r *GormRepository) RemoveShelf(ctx context.Context, userID, novelID int64) error {
	res := dbctx.From(ctx, r.db).
		Where("user_id = ? AND novel_id = ?", userID, novelID).
		Delete(&entities.LibraryEntry{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *GormRepository) ListBookmarks(ctx context.Context, userID int64, novelID *int64, p page.Page) ([]domain.Bookmark, string, error) {
	query := dbctx.From(ctx, r.db).
		Table("bookmarks b").
		Select("b.*, c.chapter_no, c.title").
		Joins("JOIN chapters c ON c.id = b.chapter_id").
		Where("b.user_id = ?", userID)

	if novelID != nil {
		query = query.Where("b.novel_id = ?", *novelID)
	}

	cursor, err := httpx.DecodeCursorFor(p.Cursor, "bookmarks")
	if err != nil {
		return nil, "", err
	}
	if len(cursor.Keys) == 1 {
		id, parseErr := strconv.ParseInt(cursor.Keys[0], 10, 64)
		if parseErr != nil {
			return nil, "", httpx.ErrBadCursor
		}
		query = query.Where("b.id < ?", id)
	}

	type bookmarkRow struct {
		entities.Bookmark
		ChapterNo int
		Title     string
	}
	var rows []bookmarkRow
	if err := query.Order("b.id DESC").Limit(p.Limit + 1).Scan(&rows).Error; err != nil {
		return nil, "", err
	}

	next := ""
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		next = httpx.EncodeCursor(httpx.Cursor{
			Sort: "bookmarks",
			Keys: []string{strconv.FormatInt(rows[len(rows)-1].ID, 10)},
		})
	}

	out := make([]domain.Bookmark, 0, len(rows))
	for _, row := range rows {
		item := domain.Bookmark{
			ID:         row.ID,
			UserID:     row.UserID,
			NovelID:    row.NovelID,
			ChapterID:  row.ChapterID,
			ParaAnchor: row.ParaAnchor,
			Excerpt:    row.Excerpt,
			CreatedAt:  row.CreatedAt,
			ChapterNo:  row.ChapterNo,
			Title:      row.Title,
		}
		if row.Note != nil {
			item.Note = *row.Note
		}
		out = append(out, item)
	}
	return out, next, nil
}

func (r *GormRepository) CreateBookmark(ctx context.Context, b domain.Bookmark) (*domain.Bookmark, error) {
	row := entities.Bookmark{
		UserID:     b.UserID,
		NovelID:    b.NovelID,
		ChapterID:  b.ChapterID,
		ParaAnchor: b.ParaAnchor,
		Excerpt:    b.Excerpt,
		CreatedAt:  b.CreatedAt,
	}
	if b.Note != "" {
		note := b.Note
		row.Note = &note
	}
	if err := dbctx.From(ctx, r.db).Create(&row).Error; err != nil {
		return nil, err
	}
	b.ID = row.ID
	return &b, nil
}

// DeleteBookmark reads before deleting so a bookmark that exists but belongs to
// somebody else is a 403 rather than an indistinguishable 404. A conditional
// DELETE returning zero rows cannot tell the two cases apart.
func (r *GormRepository) DeleteBookmark(ctx context.Context, id, userID int64) error {
	db := dbctx.From(ctx, r.db)

	var row entities.Bookmark
	if err := db.Where("id = ?", id).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	if row.UserID != userID {
		return domain.ErrForbidden
	}
	return db.Where("id = ?", id).Delete(&entities.Bookmark{}).Error
}

// Follow inserts the row and bumps the denormalised counter in one transaction,
// only when the insert actually happened, so a repeat follow cannot inflate it.
func (r *GormRepository) Follow(ctx context.Context, userID, novelID int64) error {
	return dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&entities.Follow{UserID: userID, NovelID: novelID})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		return tx.Model(&entities.Novel{}).
			Where("id = ?", novelID).
			UpdateColumn("followers_count", gorm.Expr("followers_count + 1")).Error
	})
}

func (r *GormRepository) Unfollow(ctx context.Context, userID, novelID int64) error {
	return dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("user_id = ? AND novel_id = ?", userID, novelID).
			Delete(&entities.Follow{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		// GREATEST guards the counter against drifting negative if it was ever
		// seeded out of step with the follows table.
		return tx.Model(&entities.Novel{}).
			Where("id = ?", novelID).
			UpdateColumn("followers_count", gorm.Expr("GREATEST(followers_count - 1, 0)")).Error
	})
}

func (r *GormRepository) IsFollowing(ctx context.Context, userID, novelID int64) (bool, error) {
	var count int64
	err := dbctx.From(ctx, r.db).Model(&entities.Follow{}).
		Where("user_id = ? AND novel_id = ?", userID, novelID).
		Count(&count).Error
	return count > 0, err
}

// FollowMany follows every novel in one statement and returns how many rows it
// actually created.
//
// `ON CONFLICT ... RETURNING novel_id` is what keeps followers_count exact:
// Postgres reports only the rows it really inserted, so two readers racing on
// the same series increment each counter once between them rather than twice.
// Counting the request instead would drift the moment a book was already
// followed.
func (r *GormRepository) FollowMany(ctx context.Context, userID int64, novelIDs []int64) (int, error) {
	if len(novelIDs) == 0 {
		return 0, nil
	}
	var inserted int
	err := dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		values, args := followValues(userID, novelIDs)
		var ids []int64
		sql := `INSERT INTO follows (user_id, novel_id) VALUES ` + values +
			` ON CONFLICT (user_id, novel_id) DO NOTHING RETURNING novel_id`
		if err := tx.Raw(sql, args...).Scan(&ids).Error; err != nil {
			return err
		}
		inserted = len(ids)
		if inserted == 0 {
			return nil
		}
		return tx.Model(&entities.Novel{}).
			Where("id IN ?", ids).
			UpdateColumn("followers_count", gorm.Expr("followers_count + 1")).Error
	})
	return inserted, err
}

// UnfollowMany drops the caller's follows on these novels, returning how many
// rows it removed. Same reasoning as FollowMany: the counter follows the rows
// Postgres deleted, never the size of the request.
func (r *GormRepository) UnfollowMany(ctx context.Context, userID int64, novelIDs []int64) (int, error) {
	if len(novelIDs) == 0 {
		return 0, nil
	}
	var removed int
	err := dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		var ids []int64
		err := tx.Raw(
			`DELETE FROM follows WHERE user_id = ? AND novel_id IN ? RETURNING novel_id`,
			userID, novelIDs,
		).Scan(&ids).Error
		if err != nil {
			return err
		}
		removed = len(ids)
		if removed == 0 {
			return nil
		}
		// GREATEST guards the counter against drifting negative if it was ever
		// seeded out of step with the follows table.
		return tx.Model(&entities.Novel{}).
			Where("id IN ?", ids).
			UpdateColumn("followers_count", gorm.Expr("GREATEST(followers_count - 1, 0)")).Error
	})
	return removed, err
}

// CountFollowing reports how many of these novels the reader already follows,
// which is what turns a series into none / partial / all.
func (r *GormRepository) CountFollowing(ctx context.Context, userID int64, novelIDs []int64) (int, error) {
	if len(novelIDs) == 0 {
		return 0, nil
	}
	var n int64
	err := dbctx.From(ctx, r.db).Model(&entities.Follow{}).
		Where("user_id = ? AND novel_id IN ?", userID, novelIDs).
		Count(&n).Error
	return int(n), err
}

// followValues builds the VALUES list for FollowMany. A series holds a handful
// of books, so an inline list is cheaper to read than array marshalling and
// stays driver-agnostic.
func followValues(userID int64, novelIDs []int64) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, len(novelIDs)*2)
	for i, id := range novelIDs {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?)")
		args = append(args, userID, id)
	}
	return sb.String(), args
}

func (r *GormRepository) ListFollowerIDs(ctx context.Context, novelID, afterID int64, limit int) ([]int64, error) {
	var ids []int64
	err := dbctx.From(ctx, r.db).Model(&entities.Follow{}).
		Where("novel_id = ? AND user_id > ?", novelID, afterID).
		Order("user_id").
		Limit(limit).
		Pluck("user_id", &ids).Error
	return ids, err
}
