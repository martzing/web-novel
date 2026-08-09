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

	// ReplayByKey returns the receipt for an already-applied idempotency key,
	// or nil when the key is unused. Apply performs this check itself inside
	// the wallet lock; this exists for callers that must recognise a retry
	// *before* rebuilding a request, such as an arc bundle whose quote would
	// otherwise come back empty and look like a second purchase.
	ReplayByKey(ctx context.Context, userID int64, idempotencyKey string) (*Receipt, error)

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

	// ChapterForSale returns everything the unlock, bundle and tip paths need
	// about a chapter, in one read.
	ChapterForSale(ctx context.Context, chapterID int64) (*ChapterSale, error)
	// ArcChaptersForSale lists an arc's published, paid chapters. Membership is
	// resolved by chapter-number range, never by chapters.arc_id.
	ArcChaptersForSale(ctx context.Context, arcID int64) (*ArcSale, error)

	// Auto-unlock subscriptions live here because they are coin-spend policy;
	// a separate bounded context for one table would add wiring and no
	// isolation.
	GetSubscription(ctx context.Context, userID, novelID int64) (*Subscription, error)
	ListSubscriptions(ctx context.Context, userID int64) ([]Subscription, error)
	UpsertSubscription(ctx context.Context, s Subscription) (*Subscription, error)
	DeleteSubscription(ctx context.Context, userID, novelID int64) error
	IsSubscribed(ctx context.Context, userID, novelID int64) (bool, error)

	// AutoUnlockCandidates finds subscriber/chapter pairs still missing an
	// unlock. The predicate is the invariant itself, which is what makes the
	// fan-out job idempotent by construction.
	AutoUnlockCandidates(ctx context.Context, before time.Time, retryBefore time.Time, maxAttempts, limit int) ([]AutoUnlockCandidate, error)
	RecordAutoUnlockAttempt(ctx context.Context, a AutoUnlockAttempt) error

	ListEarnings(ctx context.Context, writerID int64, p page.Page) ([]Earning, string, error)
	SumUnpaidEarnings(ctx context.Context, writerID int64) (int, error)
	CreatePayout(ctx context.Context, p Payout) (*Payout, error)

	// ExpiryCandidates lists users whose bonus has lapsed, for the nightly job.
	ExpiryCandidates(ctx context.Context, before time.Time, limit int) ([]int64, error)
}
