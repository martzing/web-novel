package wallet

import "errors"

var (
	ErrNotFound          = errors.New("wallet: not found")
	ErrInvalidAmount     = errors.New("wallet: invalid amount")
	ErrInsufficientCoins = errors.New("wallet: insufficient coins")

	// ErrInsufficientPaidCoins is distinct from ErrInsufficientCoins so a
	// reader whose bonus balance cannot pay for a tip gets an accurate message
	// rather than being told they have no coins while the UI shows some.
	ErrInsufficientPaidCoins = errors.New("wallet: insufficient purchased coins")

	// ErrTipsDisabled covers a novel whose translator has not opted in.
	ErrTipsDisabled = errors.New("wallet: tips are not enabled for this novel")
	// ErrCannotTipSelf mirrors the rule that a translator unlocking their own
	// chapter earns nothing.
	ErrCannotTipSelf = errors.New("wallet: cannot tip your own chapter")

	// ErrAlreadyUnlocked is returned when the reader already owns the chapter.
	// It is how a concurrent double-unlock resolves to exactly one debit.
	ErrAlreadyUnlocked = errors.New("wallet: chapter already unlocked")
	// ErrChapterNotForSale covers a free chapter, which cannot be purchased.
	ErrChapterNotForSale = errors.New("wallet: chapter is not for sale")

	// ErrArcNotForSale means the novel has not enabled arc bundles, or the arc
	// holds nothing paid to sell.
	ErrArcNotForSale = errors.New("wallet: arc is not sold as a bundle")
	// ErrArcAlreadyOwned means every purchasable chapter in the arc is already
	// unlocked.
	ErrArcAlreadyOwned = errors.New("wallet: every chapter in the arc is already unlocked")
	// ErrBundleStale means the arc's contents or prices moved between quoting
	// and applying, so the quote can no longer be honoured.
	ErrBundleStale = errors.New("wallet: arc bundle quote is stale")

	// ErrEarlyAccessOnly means the chapter is still inside its early-access
	// window and only auto-unlock subscribers may read or buy it.
	ErrEarlyAccessOnly = errors.New("wallet: chapter is in early access")

	ErrPurchaseNotPending = errors.New("wallet: purchase is not pending")
	ErrPackNotFound       = errors.New("wallet: coin pack not found")

	// ErrIdempotencyConflict means the key was already used for a different
	// operation, so replaying it would silently do the wrong thing.
	ErrIdempotencyConflict = errors.New("wallet: idempotency key reused for a different operation")
)
