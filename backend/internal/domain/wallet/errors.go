package wallet

import "errors"

var (
	ErrNotFound          = errors.New("wallet: not found")
	ErrInvalidAmount     = errors.New("wallet: invalid amount")
	ErrInsufficientCoins = errors.New("wallet: insufficient coins")

	// ErrAlreadyUnlocked is returned when the reader already owns the chapter.
	// It is how a concurrent double-unlock resolves to exactly one debit.
	ErrAlreadyUnlocked = errors.New("wallet: chapter already unlocked")
	// ErrChapterNotForSale covers a free chapter, which cannot be purchased.
	ErrChapterNotForSale = errors.New("wallet: chapter is not for sale")

	ErrPurchaseNotPending = errors.New("wallet: purchase is not pending")
	ErrPackNotFound       = errors.New("wallet: coin pack not found")

	// ErrIdempotencyConflict means the key was already used for a different
	// operation, so replaying it would silently do the wrong thing.
	ErrIdempotencyConflict = errors.New("wallet: idempotency key reused for a different operation")
)
