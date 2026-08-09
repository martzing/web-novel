// Package wallet is the domain layer for the coin economy: balances, the
// append-only ledger, coin packs, purchases, chapter unlocks, writer earnings
// and payouts.
//
// These live in one bounded context because they all commit through a single
// transaction; splitting them would break the atomicity guarantee the PRD's
// zero-discrepancy target depends on.
package wallet

import "time"

// LedgerKind mirrors the CHECK constraint on coin_ledger.kind.
type LedgerKind string

const (
	KindTopup       LedgerKind = "topup"
	KindSpendUnlock LedgerKind = "spend_unlock"
	KindRefund      LedgerKind = "refund"
	KindBonusGrant  LedgerKind = "bonus_grant"
	KindBonusExpire LedgerKind = "bonus_expire"
	KindAdjust      LedgerKind = "adjust"
	// KindTip is a reader tipping a translator. It is a distinct kind rather
	// than a spend_unlock so the reader's history does not label it
	// "ปลดล็อกบท", and so the writer-stats derivations can tell the two apart.
	KindTip LedgerKind = "tip"
)

// Ref types, which distinguish operations that share a ledger kind. They are
// part of the idempotency target, so an unlock and an arc bundle that happen to
// share an id are never mistaken for each other.
const (
	RefChapterUnlock = "chapter_unlock"
	RefArcBundle     = "arc_bundle"
	RefAutoUnlock    = "auto_unlock"
	RefChapterTip    = "chapter_tip"
	RefPurchase      = "purchase"
	RefAdminAdjust   = "admin_adjust"
	RefBonusExpire   = "bonus_expire"
)

// Tip bounds. The maximum exists because the ledger is append-only and the
// money is immediately committed to a payout: an accidental extra zero on a
// phone keypad is unrecoverable, and a cap bounds the damage a stolen session
// can do.
const (
	MinTipCoins = 1
	MaxTipCoins = 1000
)

// ValidateTip checks a tip amount.
func ValidateTip(coins int) error {
	if coins < MinTipCoins || coins > MaxTipCoins {
		return ErrInvalidAmount
	}
	return nil
}

// Balance is a user's wallet state.
type Balance struct {
	UserID         int64
	Balance        int
	BonusBalance   int
	BonusExpiresAt *time.Time
	UpdatedAt      time.Time
}

// Total is the spendable amount, ignoring expiry.
func (b Balance) Total() int { return b.Balance + b.BonusBalance }

// EffectiveBonus is the bonus balance actually usable at `now`. Expired bonus
// counts as zero even before the nightly job has written it off.
func (b Balance) EffectiveBonus(now time.Time) int {
	if b.BonusExpiresAt != nil && !b.BonusExpiresAt.After(now) {
		return 0
	}
	return b.BonusBalance
}

// LedgerEntry is one append-only audit row.
type LedgerEntry struct {
	ID                int64
	UserID            int64
	Delta             int
	BonusDelta        int
	Kind              LedgerKind
	RefType           string
	RefID             *int64
	BalanceAfter      int
	BonusBalanceAfter int
	Reason            string
	ActorUserID       *int64
	IdempotencyKey    string
	CreatedAt         time.Time
}

// Pack is a purchasable coin bundle.
type Pack struct {
	ID          int64
	Coins       int
	BonusCoins  int
	PriceSatang int
	Currency    string
	IsBestValue bool
	SortNo      int
}

// Purchase is a top-up attempt. In phases 1–4 the only provider is "mock".
type Purchase struct {
	ID           int64
	UserID       int64
	PackID       int64
	Provider     string
	ProviderRef  string
	AmountSatang int
	Currency     string
	Status       string
	LedgerID     *int64
	CreatedAt    time.Time
	CompletedAt  *time.Time
}

// Purchase statuses.
const (
	PurchasePending   = "pending"
	PurchaseSucceeded = "succeeded"
	PurchaseFailed    = "failed"
	PurchaseRefunded  = "refunded"
)

// ProviderMock is the only payment provider wired in phases 1–4.
const ProviderMock = "mock"

// ChapterSale is everything the unlock, bundle and tip paths need to know
// about a chapter, read in one go.
type ChapterSale struct {
	ChapterID    int64
	NovelID      int64
	ChapterNo    int
	PriceCoins   int
	TranslatorID *int64
	// PublicAt gates the sale: before it, only auto-unlock subscribers may buy.
	// nil means the chapter is public immediately.
	PublicAt    *time.Time
	TipsEnabled bool
	SellByArc   bool
}

// IsPublic reports whether the chapter has left its early-access window.
func (c ChapterSale) IsPublic(now time.Time) bool {
	return c.PublicAt == nil || !c.PublicAt.After(now)
}

// ArcSale is an arc's purchasable contents.
type ArcSale struct {
	ArcID     int64
	NovelID   int64
	ArcNo     int
	Name      string
	SellByArc bool
	Chapters  []ChapterSale
}

// Subscription is a reader's auto-unlock opt-in for one novel, which also
// grants them the early-access window.
type Subscription struct {
	UserID  int64
	NovelID int64
	Active  bool
	// MaxCoinsPerChapter is 0 for "no cap".
	MaxCoinsPerChapter int
	NovelTitleTH       string
	NovelSlug          string
}

// AutoUnlockCandidate is one subscriber/chapter pair the fan-out job may debit.
type AutoUnlockCandidate struct {
	UserID             int64
	ChapterID          int64
	NovelID            int64
	PriceCoins         int
	TranslatorID       *int64
	MaxCoinsPerChapter int
}

// AutoUnlockAttempt records what the fan-out decided for one pair.
type AutoUnlockAttempt struct {
	UserID    int64
	ChapterID int64
	Outcome   string
	LedgerID  *int64
	Now       time.Time
}

// Auto-unlock fan-out outcomes.
const (
	AutoUnlockUnlocked     = "unlocked"
	AutoUnlockInsufficient = "insufficient"
	AutoUnlockOverCap      = "over_cap"
	AutoUnlockSkipped      = "skipped"
)

// Unlock records that a reader owns a chapter.
type Unlock struct {
	UserID     int64
	ChapterID  int64
	CoinsSpent int
	LedgerID   int64
	UnlockedAt time.Time
}

// Earning credits a translator for one unlock.
type Earning struct {
	ID             int64
	WriterID       int64
	ChapterID      int64
	UnlockLedgerID int64
	GrossCoins     int
	NetCoins       int
	CreatedAt      time.Time
}

// Payout is a fiat withdrawal request.
type Payout struct {
	ID           int64
	WriterID     int64
	AmountSatang int
	Status       string
	RequestedAt  time.Time
	PaidAt       *time.Time
}

// Payout statuses.
const (
	PayoutRequested = "requested"
	PayoutApproved  = "approved"
	PayoutPaid      = "paid"
	PayoutRejected  = "rejected"
)

// NetCoins applies the platform fee to a gross amount, rounding the platform's
// share down so the translator is never short-changed by rounding.
func NetCoins(gross, feePercent int) int {
	if feePercent <= 0 {
		return gross
	}
	if feePercent >= 100 {
		return 0
	}
	fee := gross * feePercent / 100
	return gross - fee
}
