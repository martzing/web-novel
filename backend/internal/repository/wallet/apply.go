package wallet

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	domain "github.com/mokchan/webnovel-backend/internal/domain/wallet"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

// Apply is the single coin write path.
//
// The whole operation runs in one transaction whose first act is to lock the
// caller's wallet_balances row FOR UPDATE. That row is the only lock any coin
// operation takes first, so there is exactly one lock-acquisition order
// (wallet_balances -> purchases -> child insert) and deadlock is structurally
// impossible under concurrent unlocks.
//
// Ordering inside the lock matters:
//
//  1. ensure the balance row exists, then lock it;
//  2. replay check — an existing ledger row for this idempotency key returns
//     the stored result and writes nothing (I-COIN-07);
//  3. child preconditions — an existing unlock short-circuits before any debit,
//     which is what makes a concurrent double-click cost exactly one debit
//     (I-COIN-02);
//  4. plan, then write the bonus-expiry row before the main row so the ledger
//     remains a valid running total;
//  5. update the balance and write the child row referencing the new ledger id.
func (r *GormRepository) Apply(ctx context.Context, cmd domain.Command) (*domain.Receipt, error) {
	var receipt *domain.Receipt

	err := dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		// A brand-new user has no wallet row, and there must be one to lock.
		if err := ensureBalanceRow(tx, cmd.UserID); err != nil {
			return err
		}

		var current entities.WalletBalance
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", cmd.UserID).
			Take(&current).Error
		if err != nil {
			return err
		}

		if cmd.IdempotencyKey != "" {
			replay, err := findReplay(tx, cmd)
			if err != nil {
				return err
			}
			if replay != nil {
				receipt = replay
				return nil
			}
		}

		if err := checkChildPreconditions(tx, cmd); err != nil {
			return err
		}

		balance := toDomainBalance(current)
		plan, err := planFor(balance, cmd)
		if err != nil {
			return err
		}

		// The write-off goes in first so balance_after on each row reads as a
		// consistent running total.
		var expiryEntry *domain.LedgerEntry
		if plan.Expiry != nil {
			row := toLedgerRow(cmd, *plan.Expiry, "")
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			entry := toDomainLedger(row)
			expiryEntry = &entry
		}

		main := toLedgerRow(cmd, plan.Entry, cmd.IdempotencyKey)
		if err := tx.Create(&main).Error; err != nil {
			// Two requests can race past the replay check only if they were
			// never serialized by the same wallet lock; the unique index is the
			// backstop.
			if isUniqueViolation(err) {
				replay, rerr := findReplay(tx, cmd)
				if rerr != nil {
					return rerr
				}
				if replay != nil {
					receipt = replay
					return nil
				}
			}
			return err
		}

		err = tx.Model(&entities.WalletBalance{}).
			Where("user_id = ?", cmd.UserID).
			Updates(map[string]any{
				"balance":          plan.Result.Balance,
				"bonus_balance":    plan.Result.BonusBalance,
				"bonus_expires_at": plan.Result.BonusExpiresAt,
				"updated_at":       cmd.Now,
			}).Error
		if err != nil {
			return err
		}

		if err := writeChild(tx, cmd, main.ID, plan.Entry); err != nil {
			return err
		}

		receipt = &domain.Receipt{
			Ledger:       toDomainLedger(main),
			ExpiryLedger: expiryEntry,
			Balance:      plan.Result,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func ensureBalanceRow(tx *gorm.DB, userID int64) error {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&entities.WalletBalance{UserID: userID}).Error
}

// findReplay returns the receipt for an already-applied idempotency key, or nil
// when the key is new. A key reused for a *different* target is a conflict, not
// a replay: silently returning the old result would confirm an operation the
// caller did not request.
func findReplay(tx *gorm.DB, cmd domain.Command) (*domain.Receipt, error) {
	var existing entities.CoinLedgerEntry
	err := tx.Where("user_id = ? AND idempotency_key = ?", cmd.UserID, cmd.IdempotencyKey).
		Take(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if !sameTarget(existing, cmd) {
		return nil, domain.ErrIdempotencyConflict
	}

	var balance entities.WalletBalance
	if err := tx.Where("user_id = ?", cmd.UserID).Take(&balance).Error; err != nil {
		return nil, err
	}
	return &domain.Receipt{
		Ledger:   toDomainLedger(existing),
		Balance:  toDomainBalance(balance),
		Replayed: true,
	}, nil
}

func sameTarget(existing entities.CoinLedgerEntry, cmd domain.Command) bool {
	if existing.Kind != string(cmd.Op.Kind) {
		return false
	}
	if (existing.RefID == nil) != (cmd.Op.RefID == nil) {
		return false
	}
	if existing.RefID != nil && cmd.Op.RefID != nil && *existing.RefID != *cmd.Op.RefID {
		return false
	}
	return true
}

func checkChildPreconditions(tx *gorm.DB, cmd domain.Command) error {
	switch cmd.Child.Kind {
	case domain.ChildChapterUnlock:
		var count int64
		err := tx.Model(&entities.ChapterUnlock{}).
			Where("user_id = ? AND chapter_id = ?", cmd.UserID, cmd.Child.ChapterID).
			Count(&count).Error
		if err != nil {
			return err
		}
		if count > 0 {
			return domain.ErrAlreadyUnlocked
		}

	case domain.ChildPurchaseComplete:
		var purchase entities.Purchase
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", cmd.Child.PurchaseID).
			Take(&purchase).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if purchase.UserID != cmd.UserID {
			return domain.ErrNotFound
		}
		if purchase.Status != entities.PurchasePending {
			return domain.ErrPurchaseNotPending
		}

	case domain.ChildNone:
	}
	return nil
}

func planFor(balance domain.Balance, cmd domain.Command) (domain.Plan, error) {
	switch cmd.Op.Kind {
	case domain.KindSpendUnlock:
		return domain.PlanSpend(balance, cmd.Op.Amount, cmd.Now)
	case domain.KindTopup, domain.KindBonusGrant, domain.KindRefund:
		plan, err := domain.PlanCredit(balance, cmd.Op.Coins, cmd.Op.BonusCoins, cmd.Op.BonusTTL, cmd.Now)
		if err != nil {
			return domain.Plan{}, err
		}
		plan.Entry.Kind = cmd.Op.Kind
		return plan, nil
	case domain.KindAdjust:
		return domain.PlanAdjust(balance, cmd.Op.Delta, cmd.Op.BonusDelta, cmd.Now)
	case domain.KindBonusExpire:
		plan, ok := domain.PlanBonusExpiry(balance, cmd.Now)
		if !ok {
			return domain.Plan{}, domain.ErrInvalidAmount
		}
		return plan, nil
	default:
		return domain.Plan{}, domain.ErrInvalidAmount
	}
}

func writeChild(tx *gorm.DB, cmd domain.Command, ledgerID int64, entry domain.LedgerEntry) error {
	switch cmd.Child.Kind {
	case domain.ChildChapterUnlock:
		spent := -(entry.Delta + entry.BonusDelta)
		unlock := entities.ChapterUnlock{
			UserID:     cmd.UserID,
			ChapterID:  cmd.Child.ChapterID,
			CoinsSpent: int16(spent),
			LedgerID:   ledgerID,
			UnlockedAt: cmd.Now,
		}
		if err := tx.Create(&unlock).Error; err != nil {
			if isUniqueViolation(err) {
				return domain.ErrAlreadyUnlocked
			}
			return err
		}
		if cmd.Child.WriterID != 0 {
			earning := entities.WriterEarning{
				WriterID:       cmd.Child.WriterID,
				ChapterID:      cmd.Child.ChapterID,
				UnlockLedgerID: ledgerID,
				GrossCoins:     spent,
				NetCoins:       cmd.Child.NetCoins,
				CreatedAt:      cmd.Now,
			}
			if err := tx.Create(&earning).Error; err != nil {
				return err
			}
		}

	case domain.ChildPurchaseComplete:
		res := tx.Model(&entities.Purchase{}).
			Where("id = ? AND status = ?", cmd.Child.PurchaseID, entities.PurchasePending).
			Updates(map[string]any{
				"status":       entities.PurchaseSucceeded,
				"ledger_id":    ledgerID,
				"completed_at": cmd.Now,
			})
		if res.Error != nil {
			return res.Error
		}
		// The row was locked and checked above, so losing the update here means
		// an invariant broke rather than a benign race.
		if res.RowsAffected != 1 {
			return domain.ErrPurchaseNotPending
		}

	case domain.ChildNone:
	}
	return nil
}

func toLedgerRow(cmd domain.Command, entry domain.LedgerEntry, idempotencyKey string) entities.CoinLedgerEntry {
	row := entities.CoinLedgerEntry{
		UserID:            cmd.UserID,
		Delta:             entry.Delta,
		BonusDelta:        entry.BonusDelta,
		Kind:              string(entry.Kind),
		RefID:             cmd.Op.RefID,
		BalanceAfter:      entry.BalanceAfter,
		BonusBalanceAfter: entry.BonusBalanceAfter,
		ActorUserID:       cmd.ActorUserID,
		CreatedAt:         cmd.Now,
	}
	if cmd.Op.RefType != "" {
		refType := cmd.Op.RefType
		row.RefType = &refType
	}
	if cmd.Reason != "" {
		reason := cmd.Reason
		row.Reason = &reason
	}
	// Left NULL when absent: the table is UNIQUE (user_id, idempotency_key) and
	// Postgres treats NULLs as distinct, so every key-less row (each
	// bonus_expire) can coexist. An empty string would collide on the second.
	if idempotencyKey != "" {
		key := idempotencyKey
		row.IdempotencyKey = &key
	}
	return row
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
