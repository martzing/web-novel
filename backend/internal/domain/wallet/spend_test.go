package wallet

import (
	"errors"
	"testing"
	"time"
)

var now = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

func at(offset time.Duration) *time.Time {
	t := now.Add(offset)
	return &t
}

// U-COIN-01 — bonus coins are always spent before paid coins.
func TestPlanSpend_UsesBonusBeforePaid(t *testing.T) {
	tests := []struct {
		name           string
		balance        Balance
		amount         int
		wantDelta      int // paid portion, negative
		wantBonusDelta int // bonus portion, negative
		wantBalance    int
		wantBonus      int
	}{
		{
			name:           "bonus alone covers the spend",
			balance:        Balance{Balance: 100, BonusBalance: 50, BonusExpiresAt: at(time.Hour)},
			amount:         30,
			wantDelta:      0,
			wantBonusDelta: -30,
			wantBalance:    100,
			wantBonus:      20,
		},
		{
			name:           "spend splits across bonus then paid",
			balance:        Balance{Balance: 100, BonusBalance: 50, BonusExpiresAt: at(time.Hour)},
			amount:         70,
			wantDelta:      -20,
			wantBonusDelta: -50,
			wantBalance:    80,
			wantBonus:      0,
		},
		{
			name:           "no bonus means paid only",
			balance:        Balance{Balance: 100},
			amount:         25,
			wantDelta:      -25,
			wantBonusDelta: 0,
			wantBalance:    75,
			wantBonus:      0,
		},
		{
			name:           "bonus exactly covers the spend",
			balance:        Balance{Balance: 10, BonusBalance: 40, BonusExpiresAt: at(time.Hour)},
			amount:         40,
			wantDelta:      0,
			wantBonusDelta: -40,
			wantBalance:    10,
			wantBonus:      0,
		},
		{
			name:           "spend drains both balances exactly",
			balance:        Balance{Balance: 10, BonusBalance: 5, BonusExpiresAt: at(time.Hour)},
			amount:         15,
			wantDelta:      -10,
			wantBonusDelta: -5,
			wantBalance:    0,
			wantBonus:      0,
		},
		{
			name:           "bonus with no expiry is still spent first",
			balance:        Balance{Balance: 100, BonusBalance: 20},
			amount:         30,
			wantDelta:      -10,
			wantBonusDelta: -20,
			wantBalance:    90,
			wantBonus:      0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := PlanSpend(tc.balance, tc.amount, now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if plan.Expiry != nil {
				t.Fatalf("did not expect an expiry row, got %+v", plan.Expiry)
			}
			if plan.Entry.Delta != tc.wantDelta {
				t.Fatalf("delta = %d, want %d", plan.Entry.Delta, tc.wantDelta)
			}
			if plan.Entry.BonusDelta != tc.wantBonusDelta {
				t.Fatalf("bonus_delta = %d, want %d", plan.Entry.BonusDelta, tc.wantBonusDelta)
			}
			if plan.Entry.Kind != KindSpendUnlock {
				t.Fatalf("kind = %q, want %q", plan.Entry.Kind, KindSpendUnlock)
			}
			if plan.Entry.BalanceAfter != tc.wantBalance || plan.Result.Balance != tc.wantBalance {
				t.Fatalf("balance after = %d/%d, want %d",
					plan.Entry.BalanceAfter, plan.Result.Balance, tc.wantBalance)
			}
			if plan.Entry.BonusBalanceAfter != tc.wantBonus || plan.Result.BonusBalance != tc.wantBonus {
				t.Fatalf("bonus after = %d/%d, want %d",
					plan.Entry.BonusBalanceAfter, plan.Result.BonusBalance, tc.wantBonus)
			}

			// The invariant the PRD's reconciliation target depends on: the
			// deltas must exactly reconcile the before and after states.
			if tc.balance.Balance+plan.Entry.Delta != plan.Result.Balance {
				t.Fatalf("delta does not reconcile the paid balance")
			}
			if tc.balance.BonusBalance+plan.Entry.BonusDelta != plan.Result.BonusBalance {
				t.Fatalf("bonus_delta does not reconcile the bonus balance")
			}
		})
	}
}

// U-COIN-02 / I-COIN-10 — a spend arriving after the bonus expired but before
// the nightly job ran writes the write-off first, then the spend, so the ledger
// stays a valid running total.
func TestPlanSpend_ExpiredBonusEmitsExpiryEntryBeforeSpend(t *testing.T) {
	balance := Balance{UserID: 7, Balance: 100, BonusBalance: 50, BonusExpiresAt: at(-time.Hour)}

	plan, err := PlanSpend(balance, 30, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Expiry == nil {
		t.Fatal("expected a bonus_expire entry to be planned before the spend")
	}
	if plan.Expiry.Kind != KindBonusExpire {
		t.Fatalf("expiry kind = %q, want %q", plan.Expiry.Kind, KindBonusExpire)
	}
	if plan.Expiry.BonusDelta != -50 || plan.Expiry.Delta != 0 {
		t.Fatalf("expiry entry = delta %d bonus %d, want 0/-50", plan.Expiry.Delta, plan.Expiry.BonusDelta)
	}

	// The expiry row's after-state is computed before the spend's, so the two
	// rows form a consistent running total.
	if plan.Expiry.BalanceAfter != 100 || plan.Expiry.BonusBalanceAfter != 0 {
		t.Fatalf("expiry after-state = %d/%d, want 100/0",
			plan.Expiry.BalanceAfter, plan.Expiry.BonusBalanceAfter)
	}

	// The spend itself cannot touch the expired bonus.
	if plan.Entry.BonusDelta != 0 {
		t.Fatalf("bonus_delta = %d, want 0: expired bonus is unspendable", plan.Entry.BonusDelta)
	}
	if plan.Entry.Delta != -30 {
		t.Fatalf("delta = %d, want -30", plan.Entry.Delta)
	}
	if plan.Result.Balance != 70 || plan.Result.BonusBalance != 0 {
		t.Fatalf("result = %d/%d, want 70/0", plan.Result.Balance, plan.Result.BonusBalance)
	}
	if plan.Result.BonusExpiresAt != nil {
		t.Fatal("a zero bonus balance must clear the expiry timestamp")
	}
}

func TestPlanSpend_ExpiredBonusWithZeroBalanceWritesNoExpiryRow(t *testing.T) {
	balance := Balance{Balance: 100, BonusBalance: 0, BonusExpiresAt: at(-time.Hour)}

	plan, err := PlanSpend(balance, 10, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Expiry != nil {
		t.Fatal("nothing to expire: a no-op ledger row must not be written")
	}
}

// The boundary case: expiry is inclusive, so bonus is unusable at exactly its
// deadline.
func TestPlanSpend_BonusExpiresAtTheDeadlineItself(t *testing.T) {
	exactly := now
	balance := Balance{Balance: 100, BonusBalance: 50, BonusExpiresAt: &exactly}

	plan, err := PlanSpend(balance, 10, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Expiry == nil {
		t.Fatal("bonus expiring exactly now must be written off")
	}
	if plan.Entry.BonusDelta != 0 {
		t.Fatalf("bonus_delta = %d, want 0", plan.Entry.BonusDelta)
	}
}

// I-COIN-03 — an unaffordable spend produces no plan at all, so no ledger row
// is ever written.
func TestPlanSpend_InsufficientCoins(t *testing.T) {
	tests := []struct {
		name    string
		balance Balance
		amount  int
	}{
		{"empty wallet", Balance{}, 1},
		{"one coin short", Balance{Balance: 4}, 5},
		{"bonus plus paid still short", Balance{Balance: 3, BonusBalance: 1, BonusExpiresAt: at(time.Hour)}, 5},
		{"expired bonus cannot make up the difference", Balance{Balance: 3, BonusBalance: 90, BonusExpiresAt: at(-time.Hour)}, 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PlanSpend(tc.balance, tc.amount, now)
			if !errors.Is(err, ErrInsufficientCoins) {
				t.Fatalf("error = %v, want ErrInsufficientCoins", err)
			}
		})
	}
}

func TestPlanSpend_RejectsNonPositiveAmount(t *testing.T) {
	for _, amount := range []int{0, -1, -100} {
		if _, err := PlanSpend(Balance{Balance: 100}, amount, now); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("amount %d: error = %v, want ErrInvalidAmount", amount, err)
		}
	}
}

func TestPlanCredit_SetsBonusExpiry(t *testing.T) {
	const ttl = 30 * 24 * time.Hour

	t.Run("a bonus grant starts the expiry clock", func(t *testing.T) {
		plan, err := PlanCredit(Balance{UserID: 1}, 600, 100, ttl, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Entry.Delta != 600 || plan.Entry.BonusDelta != 100 {
			t.Fatalf("entry = %d/%d, want 600/100", plan.Entry.Delta, plan.Entry.BonusDelta)
		}
		if plan.Entry.Kind != KindTopup {
			t.Fatalf("kind = %q, want %q", plan.Entry.Kind, KindTopup)
		}
		if plan.Result.BonusExpiresAt == nil || !plan.Result.BonusExpiresAt.Equal(now.Add(ttl)) {
			t.Fatalf("bonus expiry = %v, want %v", plan.Result.BonusExpiresAt, now.Add(ttl))
		}
	})

	t.Run("a bonus-free top-up leaves the existing expiry alone", func(t *testing.T) {
		existing := at(48 * time.Hour)
		plan, err := PlanCredit(Balance{Balance: 10, BonusBalance: 5, BonusExpiresAt: existing}, 60, 0, ttl, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Result.BonusExpiresAt == nil || !plan.Result.BonusExpiresAt.Equal(*existing) {
			t.Fatalf("bonus expiry = %v, want it unchanged at %v", plan.Result.BonusExpiresAt, existing)
		}
	})

	t.Run("topping up onto expired bonus writes it off first", func(t *testing.T) {
		plan, err := PlanCredit(Balance{Balance: 10, BonusBalance: 40, BonusExpiresAt: at(-time.Hour)}, 60, 10, ttl, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Expiry == nil || plan.Expiry.BonusDelta != -40 {
			t.Fatalf("expected the stale 40-coin bonus to be written off, got %+v", plan.Expiry)
		}
		// The expired 40 must not survive into the new balance.
		if plan.Result.BonusBalance != 10 {
			t.Fatalf("bonus after = %d, want 10", plan.Result.BonusBalance)
		}
		if plan.Result.Balance != 70 {
			t.Fatalf("balance after = %d, want 70", plan.Result.Balance)
		}
	})

	t.Run("no coins at all is rejected", func(t *testing.T) {
		if _, err := PlanCredit(Balance{}, 0, 0, ttl, now); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("error = %v, want ErrInvalidAmount", err)
		}
	})

	t.Run("negative amounts are rejected", func(t *testing.T) {
		if _, err := PlanCredit(Balance{}, -5, 10, ttl, now); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("error = %v, want ErrInvalidAmount", err)
		}
	})
}

func TestPlanAdjust(t *testing.T) {
	t.Run("a credit adjustment", func(t *testing.T) {
		plan, err := PlanAdjust(Balance{Balance: 10}, 25, 0, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Entry.Kind != KindAdjust {
			t.Fatalf("kind = %q, want %q", plan.Entry.Kind, KindAdjust)
		}
		if plan.Result.Balance != 35 {
			t.Fatalf("balance = %d, want 35", plan.Result.Balance)
		}
	})

	t.Run("a debit adjustment", func(t *testing.T) {
		plan, err := PlanAdjust(Balance{Balance: 40}, -15, 0, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Result.Balance != 25 {
			t.Fatalf("balance = %d, want 25", plan.Result.Balance)
		}
	})

	// The wallet_balances CHECK forbids a negative balance; catching it here
	// turns a constraint violation into a clean domain error.
	t.Run("an adjustment may not drive a balance negative", func(t *testing.T) {
		if _, err := PlanAdjust(Balance{Balance: 10}, -11, 0, now); !errors.Is(err, ErrInsufficientCoins) {
			t.Fatalf("error = %v, want ErrInsufficientCoins", err)
		}
		if _, err := PlanAdjust(Balance{BonusBalance: 5}, 0, -6, now); !errors.Is(err, ErrInsufficientCoins) {
			t.Fatalf("error = %v, want ErrInsufficientCoins", err)
		}
	})

	t.Run("a zero adjustment is rejected", func(t *testing.T) {
		if _, err := PlanAdjust(Balance{Balance: 10}, 0, 0, now); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("error = %v, want ErrInvalidAmount", err)
		}
	})
}

// I-COIN-09 — the nightly job's core.
func TestPlanBonusExpiry(t *testing.T) {
	tests := []struct {
		name    string
		balance Balance
		wantOK  bool
	}{
		{"expired bonus is written off", Balance{Balance: 10, BonusBalance: 50, BonusExpiresAt: at(-24 * time.Hour)}, true},
		{"bonus expiring exactly now is written off", Balance{BonusBalance: 5, BonusExpiresAt: &now}, true},
		{"a live bonus is untouched", Balance{BonusBalance: 50, BonusExpiresAt: at(24 * time.Hour)}, false},
		{"a bonus with no deadline never expires", Balance{BonusBalance: 50}, false},
		{"nothing to expire", Balance{Balance: 10, BonusExpiresAt: at(-24 * time.Hour)}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan, ok := PlanBonusExpiry(tc.balance, now)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if plan.Entry.Kind != KindBonusExpire {
				t.Fatalf("kind = %q, want %q", plan.Entry.Kind, KindBonusExpire)
			}
			if plan.Entry.BonusDelta != -tc.balance.BonusBalance {
				t.Fatalf("bonus_delta = %d, want %d", plan.Entry.BonusDelta, -tc.balance.BonusBalance)
			}
			// Expiry never touches purchased coins.
			if plan.Entry.Delta != 0 || plan.Result.Balance != tc.balance.Balance {
				t.Fatal("bonus expiry must leave the paid balance untouched")
			}
			if plan.Result.BonusBalance != 0 || plan.Result.BonusExpiresAt != nil {
				t.Fatalf("result = %+v, want a zeroed bonus with no deadline", plan.Result)
			}
		})
	}
}

func TestBalance_EffectiveBonus(t *testing.T) {
	tests := []struct {
		name    string
		balance Balance
		want    int
	}{
		{"live bonus", Balance{BonusBalance: 50, BonusExpiresAt: at(time.Hour)}, 50},
		{"expired bonus", Balance{BonusBalance: 50, BonusExpiresAt: at(-time.Hour)}, 0},
		{"bonus expiring exactly now", Balance{BonusBalance: 50, BonusExpiresAt: &now}, 0},
		{"no deadline", Balance{BonusBalance: 50}, 50},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.balance.EffectiveBonus(now); got != tc.want {
				t.Fatalf("EffectiveBonus = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestNetCoins(t *testing.T) {
	tests := []struct {
		name       string
		gross      int
		feePercent int
		want       int
	}{
		{"thirty percent platform fee", 100, 30, 70},
		{"no fee", 100, 0, 100},
		{"negative fee is treated as none", 100, -5, 100},
		{"full fee leaves nothing", 100, 100, 0},
		{"over-100 fee is clamped", 100, 150, 0},
		// The platform's share rounds down, so the translator keeps the
		// remainder rather than losing it.
		{"rounding favours the translator", 5, 30, 4},
		{"zero gross", 0, 30, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NetCoins(tc.gross, tc.feePercent); got != tc.want {
				t.Fatalf("NetCoins(%d, %d) = %d, want %d", tc.gross, tc.feePercent, got, tc.want)
			}
		})
	}
}
