package wallet

import (
	"context"
	"time"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
)

// Repository is the driven port for the coin economy.
type Repository interface {
	// Apply is the single coin write path. Every coin movement in the system
	// goes through it, inside one transaction that locks wallet_balances FOR
	// UPDATE, checks idempotency, writes exactly one ledger row plus any
	// bonus-expiry row, updates the balance and writes the child row.
	Apply(ctx context.Context, cmd Command) (*Receipt, error)

	GetBalance(ctx context.Context, userID int64) (*Balance, error)
	ListLedger(ctx context.Context, userID int64, p page.Page) ([]LedgerEntry, string, error)

	ListPacks(ctx context.Context) ([]Pack, error)
	GetPack(ctx context.Context, id int64) (*Pack, error)

	// CreatePurchase is idempotent on (user_id, idempotency_key): calling it
	// twice with the same key returns the same pending row.
	CreatePurchase(ctx context.Context, p Purchase, idempotencyKey string) (*Purchase, error)
	GetPurchase(ctx context.Context, id int64) (*Purchase, error)
	FailPurchase(ctx context.Context, id, userID int64) (*Purchase, error)

	IsChapterUnlocked(ctx context.Context, userID, chapterID int64) (bool, error)
	ListUnlockedChapterIDs(ctx context.Context, userID int64, chapterIDs []int64) (map[int64]bool, error)

	// ChapterForUnlock returns the price and the translator to credit.
	ChapterForUnlock(ctx context.Context, chapterID int64) (price int, translatorID *int64, err error)

	ListEarnings(ctx context.Context, writerID int64, p page.Page) ([]Earning, string, error)
	SumUnpaidEarnings(ctx context.Context, writerID int64) (int, error)
	CreatePayout(ctx context.Context, p Payout) (*Payout, error)

	// ExpiryCandidates lists users whose bonus has lapsed, for the nightly job.
	ExpiryCandidates(ctx context.Context, before time.Time, limit int) ([]int64, error)
}
