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
	// LedgerTip is a reader tipping a translator. It is a distinct kind rather
	// than a spend_unlock so the reader's history does not label it
	// "ปลดล็อกบท" and so writer-stats derivations can tell the two apart.
	LedgerTip = "tip"
)

// Ledger ref_type values, which distinguish operations sharing a kind.
const (
	RefChapterUnlock = "chapter_unlock"
	RefArcBundle     = "arc_bundle"
	RefAutoUnlock    = "auto_unlock"
	RefChapterTip    = "chapter_tip"
	RefPurchase      = "purchase"
	RefAdminAdjust   = "admin_adjust"
	RefBonusExpire   = "bonus_expire"
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

// Earning kinds, matching the CHECK on writer_earnings.kind.
const (
	EarningUnlock = "unlock"
	EarningTip    = "tip"
)

// WriterEarning matches `writer_earnings`.
//
// UnlockLedgerID names the coin_ledger row that produced the earning: a
// chapter unlock, an arc bundle, or a tip. An arc bundle writes one row per
// chapter, all sharing a single ledger id, because chapters in one arc can
// have different translators.
type WriterEarning struct {
	ID             int64     `gorm:"primaryKey;column:id"`
	WriterID       int64     `gorm:"column:writer_id;not null;index"`
	ChapterID      int64     `gorm:"column:chapter_id;not null"`
	UnlockLedgerID int64     `gorm:"column:unlock_ledger_id;not null"`
	GrossCoins     int       `gorm:"column:gross_coins;not null"`
	NetCoins       int       `gorm:"column:net_coins;not null"`
	Kind           string    `gorm:"column:kind;not null;default:unlock"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (WriterEarning) TableName() string { return "writer_earnings" }

// AutoUnlockSubscription matches `auto_unlock_subscriptions` — a reader opting
// in to have new chapters of a novel unlocked automatically on publish, which
// also grants them the early-access window.
type AutoUnlockSubscription struct {
	UserID  int64 `gorm:"primaryKey;column:user_id"`
	NovelID int64 `gorm:"primaryKey;column:novel_id"`
	Active  bool  `gorm:"column:active;not null;default:true"`
	// MaxCoinsPerChapter is 0 for "no cap"; it protects a reader from a
	// mid-series price rise.
	MaxCoinsPerChapter int16     `gorm:"column:max_coins_per_chapter;not null;default:0"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (AutoUnlockSubscription) TableName() string { return "auto_unlock_subscriptions" }

// Auto-unlock fan-out outcomes.
const (
	AutoUnlockUnlocked     = "unlocked"
	AutoUnlockInsufficient = "insufficient"
	AutoUnlockOverCap      = "over_cap"
	AutoUnlockSkipped      = "skipped"
)

// AutoUnlockAttempt matches `auto_unlock_attempts`.
//
// It is a log and a backoff record, never the source of truth: the fan-out
// job's candidate query keys off the *absence* of a chapter_unlocks row, so
// deleting a row here simply lets the job retry.
type AutoUnlockAttempt struct {
	UserID      int64     `gorm:"primaryKey;column:user_id"`
	ChapterID   int64     `gorm:"primaryKey;column:chapter_id"`
	Outcome     string    `gorm:"column:outcome;not null"`
	Attempts    int16     `gorm:"column:attempts;not null;default:1"`
	LedgerID    *int64    `gorm:"column:ledger_id"`
	AttemptedAt time.Time `gorm:"column:attempted_at;autoCreateTime"`
}

func (AutoUnlockAttempt) TableName() string { return "auto_unlock_attempts" }

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
