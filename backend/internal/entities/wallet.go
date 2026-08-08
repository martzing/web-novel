package entities

import "time"

// WalletBalance matches `wallet_balances`. This row is the serialization point
// for every coin mutation: the single write path locks it FOR UPDATE first.
type WalletBalance struct {
	UserID         int64      `gorm:"primaryKey;column:user_id"`
	Balance        int        `gorm:"column:balance;not null;default:0"`
	BonusBalance   int        `gorm:"column:bonus_balance;not null;default:0"`
	BonusExpiresAt *time.Time `gorm:"column:bonus_expires_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (WalletBalance) TableName() string { return "wallet_balances" }

// Ledger kinds, matching the CHECK constraint on coin_ledger.kind.
const (
	LedgerTopup       = "topup"
	LedgerSpendUnlock = "spend_unlock"
	LedgerRefund      = "refund"
	LedgerBonusGrant  = "bonus_grant"
	LedgerBonusExpire = "bonus_expire"
	LedgerAdjust      = "adjust"
)

// CoinLedgerEntry matches `coin_ledger`, an append-only audit log.
//
// IdempotencyKey and RefID must stay pointers: the table has
// UNIQUE (user_id, idempotency_key), and Postgres treats NULLs as distinct.
// A non-pointer string would write ” for every key-less row (each bonus_expire),
// and the second one per user would violate the constraint.
type CoinLedgerEntry struct {
	ID                int64     `gorm:"primaryKey;column:id"`
	UserID            int64     `gorm:"column:user_id;not null;index"`
	Delta             int       `gorm:"column:delta;not null"`
	BonusDelta        int       `gorm:"column:bonus_delta;not null;default:0"`
	Kind              string    `gorm:"column:kind;not null"`
	RefType           *string   `gorm:"column:ref_type"`
	RefID             *int64    `gorm:"column:ref_id"`
	BalanceAfter      int       `gorm:"column:balance_after;not null"`
	BonusBalanceAfter int       `gorm:"column:bonus_balance_after;not null"`
	Reason            *string   `gorm:"column:reason"`
	ActorUserID       *int64    `gorm:"column:actor_user_id"`
	IdempotencyKey    *string   `gorm:"column:idempotency_key"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (CoinLedgerEntry) TableName() string { return "coin_ledger" }

// CoinPack matches `coin_packs`.
type CoinPack struct {
	ID          int64  `gorm:"primaryKey;column:id"`
	Coins       int    `gorm:"column:coins;not null"`
	BonusCoins  int    `gorm:"column:bonus_coins;not null;default:0"`
	PriceSatang int    `gorm:"column:price_satang;not null"`
	Currency    string `gorm:"column:currency;not null;default:THB"`
	IsBestValue bool   `gorm:"column:is_best_value;not null;default:false"`
	Active      bool   `gorm:"column:active;not null;default:true"`
	SortNo      int16  `gorm:"column:sort_no;not null;default:0"`
}

func (CoinPack) TableName() string { return "coin_packs" }

// Purchase statuses.
const (
	PurchasePending   = "pending"
	PurchaseSucceeded = "succeeded"
	PurchaseFailed    = "failed"
	PurchaseRefunded  = "refunded"
)

// Purchase matches `purchases`. IdempotencyKey comes from migration 0005 and is
// what dedupes POST /purchases, which writes no ledger row of its own.
type Purchase struct {
	ID             int64      `gorm:"primaryKey;column:id"`
	UserID         int64      `gorm:"column:user_id;not null;index"`
	PackID         int64      `gorm:"column:pack_id;not null"`
	Provider       string     `gorm:"column:provider;not null"`
	ProviderRef    string     `gorm:"column:provider_ref;not null"`
	AmountSatang   int        `gorm:"column:amount_satang;not null"`
	Currency       string     `gorm:"column:currency;not null"`
	Status         string     `gorm:"column:status;not null"`
	LedgerID       *int64     `gorm:"column:ledger_id"`
	IdempotencyKey *string    `gorm:"column:idempotency_key"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
	CompletedAt    *time.Time `gorm:"column:completed_at"`
}

func (Purchase) TableName() string { return "purchases" }

// ChapterUnlock matches `chapter_unlocks`. Its composite PK is what makes a
// concurrent double-unlock resolve to exactly one debit.
type ChapterUnlock struct {
	UserID     int64     `gorm:"primaryKey;column:user_id"`
	ChapterID  int64     `gorm:"primaryKey;column:chapter_id"`
	CoinsSpent int16     `gorm:"column:coins_spent;not null"`
	LedgerID   int64     `gorm:"column:ledger_id;not null"`
	UnlockedAt time.Time `gorm:"column:unlocked_at;autoCreateTime"`
}

func (ChapterUnlock) TableName() string { return "chapter_unlocks" }

// WriterEarning matches `writer_earnings`.
type WriterEarning struct {
	ID             int64     `gorm:"primaryKey;column:id"`
	WriterID       int64     `gorm:"column:writer_id;not null;index"`
	ChapterID      int64     `gorm:"column:chapter_id;not null"`
	UnlockLedgerID int64     `gorm:"column:unlock_ledger_id;not null"`
	GrossCoins     int       `gorm:"column:gross_coins;not null"`
	NetCoins       int       `gorm:"column:net_coins;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (WriterEarning) TableName() string { return "writer_earnings" }

// Payout statuses.
const (
	PayoutRequested = "requested"
	PayoutApproved  = "approved"
	PayoutPaid      = "paid"
	PayoutRejected  = "rejected"
)

// Payout matches `payouts`.
type Payout struct {
	ID           int64      `gorm:"primaryKey;column:id"`
	WriterID     int64      `gorm:"column:writer_id;not null;index"`
	AmountSatang int        `gorm:"column:amount_satang;not null"`
	Status       string     `gorm:"column:status;not null"`
	RequestedAt  time.Time  `gorm:"column:requested_at;autoCreateTime"`
	PaidAt       *time.Time `gorm:"column:paid_at"`
}

func (Payout) TableName() string { return "payouts" }
