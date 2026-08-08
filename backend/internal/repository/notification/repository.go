// Package notification is the GORM adapter for the reader inbox.
package notification

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	domain "github.com/mokchan/webnovel-backend/internal/domain/notification"
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

// FanOut inserts one row per recipient in a single statement.
//
// The new_chapter dedupe index (migration 0006) makes ON CONFLICT DO NOTHING
// swallow repeats, so republishing a chapter cannot notify a follower twice.
func (r *GormRepository) FanOut(ctx context.Context, userIDs []int64, kind string, payload map[string]any, now time.Time) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	rows := make([]entities.Notification, 0, len(userIDs))
	for _, id := range userIDs {
		rows = append(rows, entities.Notification{
			UserID:    id,
			Kind:      kind,
			Payload:   string(encoded),
			CreatedAt: now,
		})
	}

	res := dbctx.From(ctx, r.db).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(rows, 200)
	return int(res.RowsAffected), res.Error
}

func (r *GormRepository) List(ctx context.Context, userID int64, unreadOnly bool, p page.Page) ([]domain.Notification, string, error) {
	query := dbctx.From(ctx, r.db).
		Model(&entities.Notification{}).
		Where("user_id = ?", userID)
	if unreadOnly {
		query = query.Where("read_at IS NULL")
	}

	cursor, err := httpx.DecodeCursorFor(p.Cursor, "notifications")
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

	var rows []entities.Notification
	if err := query.Order("id DESC").Limit(p.Limit + 1).Find(&rows).Error; err != nil {
		return nil, "", err
	}

	next := ""
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		next = httpx.EncodeCursor(httpx.Cursor{
			Sort: "notifications",
			Keys: []string{strconv.FormatInt(rows[len(rows)-1].ID, 10)},
		})
	}

	out := make([]domain.Notification, 0, len(rows))
	for _, row := range rows {
		item := domain.Notification{
			ID:        row.ID,
			UserID:    row.UserID,
			Kind:      row.Kind,
			ReadAt:    row.ReadAt,
			CreatedAt: row.CreatedAt,
			Payload:   map[string]any{},
		}
		// A malformed payload must not take the whole inbox down.
		_ = json.Unmarshal([]byte(row.Payload), &item.Payload)
		out = append(out, item)
	}
	return out, next, nil
}

func (r *GormRepository) CountUnread(ctx context.Context, userID int64) (int, error) {
	var count int64
	err := dbctx.From(ctx, r.db).Model(&entities.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&count).Error
	return int(count), err
}

func (r *GormRepository) MarkRead(ctx context.Context, userID int64, ids []int64, now time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	return dbctx.From(ctx, r.db).Model(&entities.Notification{}).
		Where("user_id = ? AND id IN ? AND read_at IS NULL", userID, ids).
		Update("read_at", now).Error
}

func (r *GormRepository) MarkAllRead(ctx context.Context, userID int64, now time.Time) error {
	return dbctx.From(ctx, r.db).Model(&entities.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now).Error
}
