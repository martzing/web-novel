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
	// ChildArcBundle writes one chapter_unlocks row per item, plus one
	// writer_earnings row per credited chapter — chapters in an arc can have
	// different translators, so a single earning row could not name the payees.
	// All of them reference the one ledger row this command creates.
	ChildArcBundle
	// ChildTip writes only writer_earnings: a tip buys no access.
	ChildTip
)

// ChildItem is one chapter inside a bundle.
type ChildItem struct {
	ChapterID int64
	// ListPrice is the price the quote was taken at. Apply re-reads the
	// chapter inside the wallet lock and refuses the command when the price has
	// moved, so a stale quote can never buy at yesterday's price.
	ListPrice int
	// Coins is this chapter's share of the discounted total. It becomes both
	// chapter_unlocks.coins_spent and writer_earnings.gross_coins.
	Coins int
	// WriterID is 0 when nobody is credited — a chapter with no translator, or
	// one whose translator is the buyer.
	WriterID int64
	NetCoins int
}

// ChildWrite describes the non-ledger rows this command must also write.
type ChildWrite struct {
	Kind       ChildKind
	ChapterID  int64
	PurchaseID int64
	WriterID   int64
	NetCoins   int
	// Items is used by ChildArcBundle only.
	Items []ChildItem
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
