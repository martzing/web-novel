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
)

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
