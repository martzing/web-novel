// Package wallet is the GORM adapter for the coin economy.
package wallet

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
	domain "github.com/mokchan/webnovel-backend/internal/domain/wallet"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

// GormRepository implements domain.Repository against PostgreSQL.
type GormRepository struct {
	db *gorm.DB
}

// New builds a repository around the shared GORM connection.
func New(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

var _ domain.Repository = (*GormRepository)(nil)

func (r *GormRepository) GetBalance(ctx context.Context, userID int64) (*domain.Balance, error) {
	var row entities.WalletBalance
	err := dbctx.From(ctx, r.db).Where("user_id = ?", userID).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// A user who has never transacted has no row; an empty wallet is
			// the correct answer, not a 404.
			return &domain.Balance{UserID: userID}, nil
		}
		return nil, err
	}
	out := toDomainBalance(row)
	return &out, nil
}

func (r *GormRepository) ListLedger(ctx context.Context, userID int64, p page.Page) ([]domain.LedgerEntry, string, error) {
	query := dbctx.From(ctx, r.db).
		Model(&entities.CoinLedgerEntry{}).
		Where("user_id = ?", userID)

	// Keyset pagination on (created_at, id): stable while new rows land at the
	// head, which OFFSET would not be.
	cursor, err := httpx.DecodeCursorFor(p.Cursor, "ledger")
	if err != nil {
		return nil, "", err
	}
	if len(cursor.Keys) == 2 {
		createdAt, parseErr := time.Parse(time.RFC3339Nano, cursor.Keys[0])
		id, idErr := strconv.ParseInt(cursor.Keys[1], 10, 64)
		if parseErr != nil || idErr != nil {
			return nil, "", httpx.ErrBadCursor
		}
		query = query.Where("(created_at, id) < (?, ?)", createdAt, id)
	}

	var rows []entities.CoinLedgerEntry
	err = query.Order("created_at DESC").Order("id DESC").
		Limit(p.Limit + 1).
		Find(&rows).Error
	if err != nil {
		return nil, "", err
	}

	next := ""
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		last := rows[len(rows)-1]
		next = httpx.EncodeCursor(httpx.Cursor{
			Sort: "ledger",
			Keys: []string{last.CreatedAt.Format(time.RFC3339Nano), strconv.FormatInt(last.ID, 10)},
		})
	}

	out := make([]domain.LedgerEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainLedger(row))
	}
	return out, next, nil
}

func (r *GormRepository) ListPacks(ctx context.Context) ([]domain.Pack, error) {
	var rows []entities.CoinPack
	err := dbctx.From(ctx, r.db).
		Where("active = ?", true).
		Order("sort_no").Order("id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.Pack, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainPack(row))
	}
	return out, nil
}

func (r *GormRepository) GetPack(ctx context.Context, id int64) (*domain.Pack, error) {
	var row entities.CoinPack
	err := dbctx.From(ctx, r.db).Where("id = ? AND active = ?", id, true).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPackNotFound
		}
		return nil, err
	}
	out := toDomainPack(row)
	return &out, nil
}

// CreatePurchase is idempotent on (user_id, idempotency_key).
//
// POST /purchases writes no ledger row, so coin_ledger's unique index cannot
// dedupe it; migration 0005 adds a dedicated index for exactly this (I-COIN-01M).
func (r *GormRepository) CreatePurchase(ctx context.Context, p domain.Purchase, idempotencyKey string) (*domain.Purchase, error) {
	row := entities.Purchase{
		UserID:       p.UserID,
		PackID:       p.PackID,
		Provider:     p.Provider,
		ProviderRef:  p.ProviderRef,
		AmountSatang: p.AmountSatang,
		Currency:     p.Currency,
		Status:       entities.PurchasePending,
	}
	if idempotencyKey != "" {
		key := idempotencyKey
		row.IdempotencyKey = &key
	}

	db := dbctx.From(ctx, r.db)
	res := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(&row)
	if res.Error != nil {
		return nil, res.Error
	}

	// Nothing inserted means the key was already used: return the original row
	// so a retry is a safe replay rather than a second pending purchase.
	if res.RowsAffected == 0 && idempotencyKey != "" {
		var existing entities.Purchase
		err := db.Where("user_id = ? AND idempotency_key = ?", p.UserID, idempotencyKey).
			Take(&existing).Error
		if err != nil {
			return nil, err
		}
		out := toDomainPurchase(existing)
		return &out, nil
	}

	out := toDomainPurchase(row)
	return &out, nil
}

func (r *GormRepository) GetPurchase(ctx context.Context, id int64) (*domain.Purchase, error) {
	var row entities.Purchase
	err := dbctx.From(ctx, r.db).Where("id = ?", id).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	out := toDomainPurchase(row)
	return &out, nil
}

func (r *GormRepository) FailPurchase(ctx context.Context, id, userID int64) (*domain.Purchase, error) {
	db := dbctx.From(ctx, r.db)

	res := db.Model(&entities.Purchase{}).
		Where("id = ? AND user_id = ? AND status = ?", id, userID, entities.PurchasePending).
		Update("status", entities.PurchaseFailed)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		// Either it does not exist, is not the caller's, or is already settled.
		purchase, err := r.GetPurchase(ctx, id)
		if err != nil {
			return nil, err
		}
		if purchase.UserID != userID {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrPurchaseNotPending
	}
	return r.GetPurchase(ctx, id)
}

func (r *GormRepository) IsChapterUnlocked(ctx context.Context, userID, chapterID int64) (bool, error) {
	var count int64
	err := dbctx.From(ctx, r.db).Model(&entities.ChapterUnlock{}).
		Where("user_id = ? AND chapter_id = ?", userID, chapterID).
		Count(&count).Error
	return count > 0, err
}

// ListUnlockedChapterIDs resolves ownership for a whole table of contents in
// one query rather than one per row.
func (r *GormRepository) ListUnlockedChapterIDs(ctx context.Context, userID int64, chapterIDs []int64) (map[int64]bool, error) {
	out := make(map[int64]bool, len(chapterIDs))
	if len(chapterIDs) == 0 {
		return out, nil
	}

	var ids []int64
	err := dbctx.From(ctx, r.db).Model(&entities.ChapterUnlock{}).
		Where("user_id = ? AND chapter_id IN ?", userID, chapterIDs).
		Pluck("chapter_id", &ids).Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

func (r *GormRepository) ListEarnings(ctx context.Context, writerID int64, p page.Page) ([]domain.Earning, string, error) {
	query := dbctx.From(ctx, r.db).
		Model(&entities.WriterEarning{}).
		Where("writer_id = ?", writerID)

	cursor, err := httpx.DecodeCursorFor(p.Cursor, "earnings")
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

	var rows []entities.WriterEarning
	if err := query.Order("id DESC").Limit(p.Limit + 1).Find(&rows).Error; err != nil {
		return nil, "", err
	}

	next := ""
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		next = httpx.EncodeCursor(httpx.Cursor{
			Sort: "earnings",
			Keys: []string{strconv.FormatInt(rows[len(rows)-1].ID, 10)},
		})
	}

	out := make([]domain.Earning, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Earning{
			ID:             row.ID,
			WriterID:       row.WriterID,
			ChapterID:      row.ChapterID,
			UnlockLedgerID: row.UnlockLedgerID,
			GrossCoins:     row.GrossCoins,
			NetCoins:       row.NetCoins,
			CreatedAt:      row.CreatedAt,
		})
	}
	return out, next, nil
}

// SumUnpaidEarnings totals net earnings minus everything already requested or
// paid out, which is the amount a writer may still withdraw.
func (r *GormRepository) SumUnpaidEarnings(ctx context.Context, writerID int64) (int, error) {
	db := dbctx.From(ctx, r.db)

	var earned struct{ Total int }
	err := db.Model(&entities.WriterEarning{}).
		Select("COALESCE(SUM(net_coins), 0) AS total").
		Where("writer_id = ?", writerID).
		Take(&earned).Error
	if err != nil {
		return 0, err
	}

	var requested struct{ Total int }
	err = db.Model(&entities.Payout{}).
		Select("COALESCE(SUM(amount_satang), 0) AS total").
		Where("writer_id = ? AND status IN ?", writerID,
			[]string{entities.PayoutRequested, entities.PayoutApproved, entities.PayoutPaid}).
		Take(&requested).Error
	if err != nil {
		return 0, err
	}

	return max(earned.Total-requested.Total, 0), nil
}

func (r *GormRepository) CreatePayout(ctx context.Context, p domain.Payout) (*domain.Payout, error) {
	row := entities.Payout{
		WriterID:     p.WriterID,
		AmountSatang: p.AmountSatang,
		Status:       entities.PayoutRequested,
	}
	if err := dbctx.From(ctx, r.db).Create(&row).Error; err != nil {
		return nil, err
	}
	out := domain.Payout{
		ID:           row.ID,
		WriterID:     row.WriterID,
		AmountSatang: row.AmountSatang,
		Status:       row.Status,
		RequestedAt:  row.RequestedAt,
	}
	return &out, nil
}

// ExpiryCandidates uses the wallet_bonus_expiry_idx partial index.
func (r *GormRepository) ExpiryCandidates(ctx context.Context, before time.Time, limit int) ([]int64, error) {
	var ids []int64
	err := dbctx.From(ctx, r.db).Model(&entities.WalletBalance{}).
		Where("bonus_balance > 0 AND bonus_expires_at IS NOT NULL AND bonus_expires_at <= ?", before).
		Order("user_id").
		Limit(limit).
		Pluck("user_id", &ids).Error
	return ids, err
}

func toDomainBalance(row entities.WalletBalance) domain.Balance {
	return domain.Balance{
		UserID:         row.UserID,
		Balance:        row.Balance,
		BonusBalance:   row.BonusBalance,
		BonusExpiresAt: row.BonusExpiresAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func toDomainLedger(row entities.CoinLedgerEntry) domain.LedgerEntry {
	out := domain.LedgerEntry{
		ID:                row.ID,
		UserID:            row.UserID,
		Delta:             row.Delta,
		BonusDelta:        row.BonusDelta,
		Kind:              domain.LedgerKind(row.Kind),
		RefID:             row.RefID,
		BalanceAfter:      row.BalanceAfter,
		BonusBalanceAfter: row.BonusBalanceAfter,
		ActorUserID:       row.ActorUserID,
		CreatedAt:         row.CreatedAt,
	}
	if row.RefType != nil {
		out.RefType = *row.RefType
	}
	if row.Reason != nil {
		out.Reason = *row.Reason
	}
	if row.IdempotencyKey != nil {
		out.IdempotencyKey = *row.IdempotencyKey
	}
	return out
}

func toDomainPack(row entities.CoinPack) domain.Pack {
	return domain.Pack{
		ID:          row.ID,
		Coins:       row.Coins,
		BonusCoins:  row.BonusCoins,
		PriceSatang: row.PriceSatang,
		Currency:    row.Currency,
		IsBestValue: row.IsBestValue,
		SortNo:      int(row.SortNo),
	}
}

func toDomainPurchase(row entities.Purchase) domain.Purchase {
	return domain.Purchase{
		ID:           row.ID,
		UserID:       row.UserID,
		PackID:       row.PackID,
		Provider:     row.Provider,
		ProviderRef:  row.ProviderRef,
		AmountSatang: row.AmountSatang,
		Currency:     row.Currency,
		Status:       row.Status,
		LedgerID:     row.LedgerID,
		CreatedAt:    row.CreatedAt,
		CompletedAt:  row.CompletedAt,
	}
}
