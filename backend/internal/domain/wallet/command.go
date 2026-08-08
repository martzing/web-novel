package wallet

import "time"

// ChildKind selects the row written alongside the ledger entry, inside the same
// transaction.
type ChildKind int

const (
	ChildNone ChildKind = iota
	// ChildChapterUnlock writes chapter_unlocks, plus writer_earnings when the
	// chapter has a translator.
	ChildChapterUnlock
	// ChildPurchaseComplete flips a pending purchase to succeeded.
	ChildPurchaseComplete
)

// ChildWrite describes the non-ledger row this command must also write.
type ChildWrite struct {
	Kind       ChildKind
	ChapterID  int64
	PurchaseID int64
	WriterID   int64
	NetCoins   int
}

// Operation is what the command does to the balance.
type Operation struct {
	Kind LedgerKind

	// Spend
	Amount int

	// Credit
	Coins      int
	BonusCoins int
	BonusTTL   time.Duration

	// Adjust
	Delta      int
	BonusDelta int

	RefType string
	RefID   *int64
}

// Command is one coin mutation. Every coin movement in the system — top-up,
// unlock, refund, bonus grant, bonus expiry, admin adjustment — is expressed as
// a Command and applied through the single write path.
type Command struct {
	UserID int64
	// IdempotencyKey is namespaced by the service ("unlock:", "topup:", …) so a
	// client reusing one key across endpoints cannot collide.
	IdempotencyKey string
	Now            time.Time
	Op             Operation
	Child          ChildWrite
	ActorUserID    *int64
	Reason         string
}

// Receipt is the outcome of applying a Command.
type Receipt struct {
	Ledger       LedgerEntry
	ExpiryLedger *LedgerEntry
	Balance      Balance
	// Replayed reports that the idempotency key had already been used for this
	// same operation, so nothing new was written and the stored result is
	// being returned verbatim.
	Replayed bool
}
