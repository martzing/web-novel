package wallet

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	domain "github.com/mokchan/webnovel-backend/internal/domain/wallet"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

// saleRow is the joined projection every sale path needs.
type saleRow struct {
	ChapterID    int64
	NovelID      int64
	ChapterNo    int
	PriceCoins   int16
	Status       string
	TranslatorID *int64
	PublicAt     *time.Time
	TipsEnabled  bool
	SellByArc    bool
}

const saleSelect = `c.id AS chapter_id, c.novel_id, c.chapter_no, c.price_coins, c.status,
	COALESCE(c.translator_id, n.primary_translator_id) AS translator_id,
	c.public_at, n.tips_enabled, n.sell_by_arc`

func (r *GormRepository) ChapterForSale(ctx context.Context, chapterID int64) (*domain.ChapterSale, error) {
	var row saleRow
	err := dbctx.From(ctx, r.db).
		Table("chapters c").
		Select(saleSelect).
		Joins("JOIN novels n ON n.id = c.novel_id").
		Where("c.id = ?", chapterID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	// An unpublished chapter cannot be bought, and saying so plainly would leak
	// that a draft exists.
	if row.Status != entities.ChapterPublished {
		return nil, domain.ErrNotFound
	}
	sale := toDomainSale(row)
	return &sale, nil
}

// ArcChaptersForSale lists an arc's published, paid chapters.
//
// Membership is resolved by chapter-number range rather than chapters.arc_id:
// arc_id is NULL for chapters created before their arc existed, so keying off
// it would silently drop chapters from the bundle and undercharge the reader.
func (r *GormRepository) ArcChaptersForSale(ctx context.Context, arcID int64) (*domain.ArcSale, error) {
	db := dbctx.From(ctx, r.db)

	var arc entities.Arc
	if err := db.Where("id = ?", arcID).Take(&arc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	var novel entities.Novel
	if err := db.Where("id = ?", arc.NovelID).Take(&novel).Error; err != nil {
		return nil, err
	}

	var rows []saleRow
	err := db.Table("chapters c").
		Select(saleSelect).
		Joins("JOIN novels n ON n.id = c.novel_id").
		Where(`c.novel_id = ? AND c.status = ? AND c.price_coins > 0
		       AND c.chapter_no BETWEEN ? AND ?`,
			arc.NovelID, entities.ChapterPublished, arc.FromChapterNo, arc.ToChapterNo).
		Order("c.chapter_no").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := &domain.ArcSale{
		ArcID:     arc.ID,
		NovelID:   arc.NovelID,
		ArcNo:     int(arc.ArcNo),
		Name:      arc.Name,
		SellByArc: novel.SellByArc,
		Chapters:  make([]domain.ChapterSale, 0, len(rows)),
	}
	for _, row := range rows {
		out.Chapters = append(out.Chapters, toDomainSale(row))
	}
	return out, nil
}

func (r *GormRepository) GetSubscription(ctx context.Context, userID, novelID int64) (*domain.Subscription, error) {
	var row entities.AutoUnlockSubscription
	err := dbctx.From(ctx, r.db).
		Where("user_id = ? AND novel_id = ?", userID, novelID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out := toDomainSubscription(row)
	return &out, nil
}

func (r *GormRepository) ListSubscriptions(ctx context.Context, userID int64) ([]domain.Subscription, error) {
	type row struct {
		UserID             int64
		NovelID            int64
		Active             bool
		MaxCoinsPerChapter int16
		TitleTH            string
		Slug               string
	}
	var rows []row
	err := dbctx.From(ctx, r.db).
		Table("auto_unlock_subscriptions s").
		Select("s.user_id, s.novel_id, s.active, s.max_coins_per_chapter, n.title_th, n.slug").
		Joins("JOIN novels n ON n.id = s.novel_id").
		Where("s.user_id = ?", userID).
		Order("s.novel_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.Subscription, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Subscription{
			UserID:             r.UserID,
			NovelID:            r.NovelID,
			Active:             r.Active,
			MaxCoinsPerChapter: int(r.MaxCoinsPerChapter),
			NovelTitleTH:       r.TitleTH,
			NovelSlug:          r.Slug,
		})
	}
	return out, nil
}

func (r *GormRepository) UpsertSubscription(ctx context.Context, s domain.Subscription) (*domain.Subscription, error) {
	row := entities.AutoUnlockSubscription{
		UserID:             s.UserID,
		NovelID:            s.NovelID,
		Active:             s.Active,
		MaxCoinsPerChapter: int16(s.MaxCoinsPerChapter),
	}
	err := dbctx.From(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "novel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"active", "max_coins_per_chapter", "updated_at"}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	out := toDomainSubscription(row)
	return &out, nil
}

func (r *GormRepository) DeleteSubscription(ctx context.Context, userID, novelID int64) error {
	return dbctx.From(ctx, r.db).
		Where("user_id = ? AND novel_id = ?", userID, novelID).
		Delete(&entities.AutoUnlockSubscription{}).Error
}

func (r *GormRepository) IsSubscribed(ctx context.Context, userID, novelID int64) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	var count int64
	err := dbctx.From(ctx, r.db).Model(&entities.AutoUnlockSubscription{}).
		Where("user_id = ? AND novel_id = ? AND active", userID, novelID).
		Count(&count).Error
	return count > 0, err
}

// AutoUnlockCandidates finds subscriber/chapter pairs that still lack an
// unlock row.
//
// The `NOT EXISTS chapter_unlocks` predicate is the invariant the fan-out
// maintains, so the query is idempotent by construction: it self-heals after an
// incident and never charges a reader who unlocked manually. `published_at >=
// s.created_at` stops a brand-new subscriber triggering a backfill of the
// novel's entire history.
func (r *GormRepository) AutoUnlockCandidates(
	ctx context.Context,
	publishedAfter time.Time,
	retryBefore time.Time,
	maxAttempts, limit int,
) ([]domain.AutoUnlockCandidate, error) {
	type row struct {
		UserID             int64
		ChapterID          int64
		NovelID            int64
		PriceCoins         int16
		TranslatorID       *int64
		MaxCoinsPerChapter int16
	}

	var rows []row
	err := dbctx.From(ctx, r.db).
		Table("auto_unlock_subscriptions s").
		Select(`s.user_id, c.id AS chapter_id, c.novel_id, c.price_coins,
		        COALESCE(c.translator_id, n.primary_translator_id) AS translator_id,
		        s.max_coins_per_chapter`).
		Joins("JOIN novels n ON n.id = s.novel_id").
		Joins(`JOIN chapters c ON c.novel_id = s.novel_id
		       AND c.status = ? AND c.price_coins > 0
		       AND c.published_at >= s.created_at
		       AND c.published_at >= ?`, entities.ChapterPublished, publishedAfter).
		Where(`s.active
		       AND NOT EXISTS (SELECT 1 FROM chapter_unlocks u
		                        WHERE u.user_id = s.user_id AND u.chapter_id = c.id)
		       AND NOT EXISTS (SELECT 1 FROM auto_unlock_attempts a
		                        WHERE a.user_id = s.user_id AND a.chapter_id = c.id
		                          AND (a.outcome <> ? OR a.attempts >= ? OR a.attempted_at > ?))`,
			entities.AutoUnlockInsufficient, maxAttempts, retryBefore).
		Order("c.published_at, s.user_id").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.AutoUnlockCandidate, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.AutoUnlockCandidate{
			UserID:             r.UserID,
			ChapterID:          r.ChapterID,
			NovelID:            r.NovelID,
			PriceCoins:         int(r.PriceCoins),
			TranslatorID:       r.TranslatorID,
			MaxCoinsPerChapter: int(r.MaxCoinsPerChapter),
		})
	}
	return out, nil
}

// RecordAutoUnlockAttempt upserts the outcome, bumping the retry counter so
// backoff terminates.
func (r *GormRepository) RecordAutoUnlockAttempt(ctx context.Context, a domain.AutoUnlockAttempt) error {
	row := entities.AutoUnlockAttempt{
		UserID:      a.UserID,
		ChapterID:   a.ChapterID,
		Outcome:     a.Outcome,
		Attempts:    1,
		LedgerID:    a.LedgerID,
		AttemptedAt: a.Now,
	}
	return dbctx.From(ctx, r.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "chapter_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"outcome":      a.Outcome,
			"ledger_id":    a.LedgerID,
			"attempted_at": a.Now,
			"attempts":     gorm.Expr("auto_unlock_attempts.attempts + 1"),
		}),
	}).Create(&row).Error
}

func toDomainSale(row saleRow) domain.ChapterSale {
	return domain.ChapterSale{
		ChapterID:    row.ChapterID,
		NovelID:      row.NovelID,
		ChapterNo:    row.ChapterNo,
		PriceCoins:   int(row.PriceCoins),
		TranslatorID: row.TranslatorID,
		PublicAt:     row.PublicAt,
		TipsEnabled:  row.TipsEnabled,
		SellByArc:    row.SellByArc,
	}
}

func toDomainSubscription(row entities.AutoUnlockSubscription) domain.Subscription {
	return domain.Subscription{
		UserID:             row.UserID,
		NovelID:            row.NovelID,
		Active:             row.Active,
		MaxCoinsPerChapter: int(row.MaxCoinsPerChapter),
	}
}
