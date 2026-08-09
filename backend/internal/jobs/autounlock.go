package jobs

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	walletdomain "github.com/mokchan/webnovel-backend/internal/domain/wallet"
	walletsvc "github.com/mokchan/webnovel-backend/internal/service/wallet"
)

// Notifier tells a reader their auto-unlock could not be paid for. It is a
// one-method port so the job does not depend on the notification context.
type Notifier interface {
	NotifyAutoUnlockFailed(ctx context.Context, userID, novelID, chapterID int64) error
}

// AutoUnlockJob debits subscribers for chapters published since they opted in.
//
// It is a *scan* for missing unlocks rather than a publish-time loop or a
// queue. Debiting inside the publish request would make publish O(subscribers)
// and hold its transaction across many wallet locks — exactly the lock-ordering
// hazard the coin design avoids. A queue would need an outbox to avoid losing
// work. Scanning makes the candidate predicate the invariant itself, so the job
// is idempotent by construction, self-heals after an incident, and never
// charges a reader who unlocked manually.
type AutoUnlockJob struct {
	DB       *gorm.DB
	Wallet   *walletsvc.Service
	Notifier Notifier
	Logger   *slog.Logger

	// Batch bounds one run. Window stops a brand-new subscriber triggering a
	// backfill of a novel's entire history. RetryAfter and MaxAttempts bound
	// the retries for a reader who was simply short of coins.
	Batch       int
	Window      time.Duration
	RetryAfter  time.Duration
	MaxAttempts int
}

func (j *AutoUnlockJob) Name() string { return "auto_unlock" }

func (j *AutoUnlockJob) Run(ctx context.Context, now time.Time) (Report, error) {
	// The advisory lock claims a batch and nothing more.
	//
	// withJobLock publishes its transaction on the context, so work done inside
	// it joins that transaction. That is fine for a small nightly job, but not
	// here: one poisoned wallet would roll back every other subscriber's debit,
	// and the job would hold a single transaction across hundreds of wallet
	// locks. So the claim commits first, and each debit below then opens its
	// own transaction on the outer context.
	var claimed []walletdomain.AutoUnlockCandidate

	acquired, err := withJobLock(ctx, j.DB, j.Name(), func(lockCtx context.Context, _ *gorm.DB) error {
		var queryErr error
		claimed, queryErr = j.Wallet.AutoUnlockCandidates(
			lockCtx,
			now.Add(-j.window()),
			now.Add(-j.retryAfter()),
			j.maxAttempts(),
			j.batch(),
		)
		return queryErr
	})
	if err != nil || !acquired {
		return Report{Name: j.Name(), Skipped: !acquired}, err
	}

	processed := 0
	for _, candidate := range claimed {
		outcome, ledgerID := j.applyOne(ctx, candidate)

		err := j.Wallet.RecordAutoUnlockAttempt(ctx, walletdomain.AutoUnlockAttempt{
			UserID:    candidate.UserID,
			ChapterID: candidate.ChapterID,
			Outcome:   outcome,
			LedgerID:  ledgerID,
			Now:       now,
		})
		if err != nil {
			j.logger().Error("record auto-unlock attempt",
				"user", candidate.UserID, "chapter", candidate.ChapterID, "err", err)
		}

		if outcome == walletdomain.AutoUnlockUnlocked {
			processed++
		}
	}
	return Report{Name: j.Name(), Processed: processed}, nil
}

// applyOne debits a single subscriber in its own transaction.
func (j *AutoUnlockJob) applyOne(
	ctx context.Context,
	c walletdomain.AutoUnlockCandidate,
) (string, *int64) {
	// A price rise past the reader's cap stops the subscription silently for
	// that chapter; they can still unlock it by hand.
	if c.MaxCoinsPerChapter > 0 && c.PriceCoins > c.MaxCoinsPerChapter {
		return walletdomain.AutoUnlockOverCap, nil
	}

	receipt, err := j.Wallet.AutoUnlockChapter(ctx, c.UserID, c.ChapterID)
	switch {
	case err == nil:
		return walletdomain.AutoUnlockUnlocked, &receipt.Ledger.ID

	case errors.Is(err, walletdomain.ErrAlreadyUnlocked):
		// They bought it manually between the scan and the debit.
		return walletdomain.AutoUnlockSkipped, nil

	case errors.Is(err, walletdomain.ErrInsufficientCoins):
		// Tell them once — the dedupe index on notifications makes a retry a
		// no-op — and leave the subscription active. Pausing it would silently
		// stop unlocking after they topped up, which is the opposite of what
		// they asked for.
		if j.Notifier != nil {
			if nerr := j.Notifier.NotifyAutoUnlockFailed(ctx, c.UserID, c.NovelID, c.ChapterID); nerr != nil {
				j.logger().Warn("notify auto-unlock failure", "user", c.UserID, "err", nerr)
			}
		}
		return walletdomain.AutoUnlockInsufficient, nil

	case errors.Is(err, walletdomain.ErrChapterNotForSale), errors.Is(err, walletdomain.ErrNotFound):
		return walletdomain.AutoUnlockSkipped, nil

	default:
		j.logger().Error("auto-unlock failed",
			"user", c.UserID, "chapter", c.ChapterID, "err", err)
		return walletdomain.AutoUnlockInsufficient, nil
	}
}

func (j *AutoUnlockJob) batch() int {
	if j.Batch <= 0 {
		return 200
	}
	return j.Batch
}

func (j *AutoUnlockJob) window() time.Duration {
	if j.Window <= 0 {
		return 7 * 24 * time.Hour
	}
	return j.Window
}

func (j *AutoUnlockJob) retryAfter() time.Duration {
	if j.RetryAfter <= 0 {
		return 6 * time.Hour
	}
	return j.RetryAfter
}

func (j *AutoUnlockJob) maxAttempts() int {
	if j.MaxAttempts <= 0 {
		return 3
	}
	return j.MaxAttempts
}

func (j *AutoUnlockJob) logger() *slog.Logger {
	if j.Logger == nil {
		return slog.Default()
	}
	return j.Logger
}
