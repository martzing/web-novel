package wallet

import "time"

// Plan is the complete set of rows one coin operation must write.
//
// Expiry is non-nil when stale bonus has to be written off first. It is
// deliberately a separate ledger row rather than a silent adjustment, so the
// running total stays reconcilable row by row:
//
//	sum(delta) == wallet_balances.balance
//	sum(bonus_delta) == wallet_balances.bonus_balance
type Plan struct {
	Expiry *LedgerEntry
	Entry  LedgerEntry
	Result Balance
}

// PlanSpend computes a debit, spending bonus coins before paid coins.
//
// Expired bonus is treated as zero and written off first, so a spend that
// arrives before the nightly expiry job still produces a correct, auditable
// ledger (U-COIN-02, I-COIN-10).
func PlanSpend(bal Balance, amount int, now time.Time) (Plan, error) {
	if amount <= 0 {
		return Plan{}, ErrInvalidAmount
	}

	plan := Plan{}
	effectiveBonus := bal.BonusBalance
	bonusExpiresAt := bal.BonusExpiresAt

	if expired(bal, now) {
		effectiveBonus = 0
		bonusExpiresAt = nil
		if bal.BonusBalance > 0 {
			plan.Expiry = &LedgerEntry{
				UserID:            bal.UserID,
				Delta:             0,
				BonusDelta:        -bal.BonusBalance,
				Kind:              KindBonusExpire,
				BalanceAfter:      bal.Balance,
				BonusBalanceAfter: 0,
				CreatedAt:         now,
			}
		}
	}

	if effectiveBonus+bal.Balance < amount {
		return Plan{}, ErrInsufficientCoins
	}

	// Bonus first: it is the balance that can expire, so spending it first is
	// what makes the expiry policy fair to the reader.
	fromBonus := min(effectiveBonus, amount)
	fromPaid := amount - fromBonus

	bonusAfter := effectiveBonus - fromBonus
	balanceAfter := bal.Balance - fromPaid
	if bonusAfter == 0 {
		bonusExpiresAt = nil
	}

	plan.Entry = LedgerEntry{
		UserID:            bal.UserID,
		Delta:             -fromPaid,
		BonusDelta:        -fromBonus,
		Kind:              KindSpendUnlock,
		BalanceAfter:      balanceAfter,
		BonusBalanceAfter: bonusAfter,
		CreatedAt:         now,
	}
	plan.Result = Balance{
		UserID:         bal.UserID,
		Balance:        balanceAfter,
		BonusBalance:   bonusAfter,
		BonusExpiresAt: bonusExpiresAt,
		UpdatedAt:      now,
	}
	return plan, nil
}

// PlanSpendPaidOnly computes a debit that may draw on purchased coins only.
//
// Bonus coins are promotional currency. An unlock may spend them because the
// promotion was priced knowing it drives reading, and the resulting payout
// obligation is bounded by chapter prices the platform itself sets. A tip is
// different: it converts free currency into an unbounded, reader-chosen cash
// payout with nothing exchanged, which is a direct subsidy and an obvious farm
// — sign up for the bonus, tip a colluding translator, request a payout.
//
// Lapsed bonus is still written off first, exactly as PlanSpend does, so the
// ledger remains a valid running total whichever planner ran.
func PlanSpendPaidOnly(bal Balance, amount int, now time.Time) (Plan, error) {
	if amount <= 0 {
		return Plan{}, ErrInvalidAmount
	}

	plan := Plan{}
	bonusAfter := bal.BonusBalance
	bonusExpiresAt := bal.BonusExpiresAt

	if expired(bal, now) {
		bonusAfter = 0
		bonusExpiresAt = nil
		if bal.BonusBalance > 0 {
			plan.Expiry = &LedgerEntry{
				UserID:            bal.UserID,
				Delta:             0,
				BonusDelta:        -bal.BonusBalance,
				Kind:              KindBonusExpire,
				BalanceAfter:      bal.Balance,
				BonusBalanceAfter: 0,
				CreatedAt:         now,
			}
		}
	}

	// A distinct error: telling a reader "not enough coins" while the wallet
	// card shows a healthy bonus balance is a guaranteed support ticket.
	if bal.Balance < amount {
		return Plan{}, ErrInsufficientPaidCoins
	}

	balanceAfter := bal.Balance - amount

	plan.Entry = LedgerEntry{
		UserID:            bal.UserID,
		Delta:             -amount,
		BonusDelta:        0,
		Kind:              KindTip,
		BalanceAfter:      balanceAfter,
		BonusBalanceAfter: bonusAfter,
		CreatedAt:         now,
	}
	plan.Result = Balance{
		UserID:         bal.UserID,
		Balance:        balanceAfter,
		BonusBalance:   bonusAfter,
		BonusExpiresAt: bonusExpiresAt,
		UpdatedAt:      now,
	}
	return plan, nil
}

// PlanCredit computes a top-up. Any surviving bonus has its expiry extended to
// the new deadline, so a reader topping up never loses bonus they just bought.
func PlanCredit(bal Balance, coins, bonusCoins int, bonusTTL time.Duration, now time.Time) (Plan, error) {
	if coins < 0 || bonusCoins < 0 || coins+bonusCoins <= 0 {
		return Plan{}, ErrInvalidAmount
	}

	plan := Plan{}
	effectiveBonus := bal.BonusBalance

	if expired(bal, now) {
		effectiveBonus = 0
		if bal.BonusBalance > 0 {
			plan.Expiry = &LedgerEntry{
				UserID:            bal.UserID,
				Delta:             0,
				BonusDelta:        -bal.BonusBalance,
				Kind:              KindBonusExpire,
				BalanceAfter:      bal.Balance,
				BonusBalanceAfter: 0,
				CreatedAt:         now,
			}
		}
	}

	balanceAfter := bal.Balance + coins
	bonusAfter := effectiveBonus + bonusCoins

	var bonusExpiresAt *time.Time
	switch {
	case bonusAfter == 0:
		bonusExpiresAt = nil
	case bonusCoins > 0 && bonusTTL > 0:
		deadline := now.Add(bonusTTL)
		bonusExpiresAt = &deadline
	default:
		bonusExpiresAt = bal.BonusExpiresAt
	}

	plan.Entry = LedgerEntry{
		UserID:            bal.UserID,
		Delta:             coins,
		BonusDelta:        bonusCoins,
		Kind:              KindTopup,
		BalanceAfter:      balanceAfter,
		BonusBalanceAfter: bonusAfter,
		CreatedAt:         now,
	}
	plan.Result = Balance{
		UserID:         bal.UserID,
		Balance:        balanceAfter,
		BonusBalance:   bonusAfter,
		BonusExpiresAt: bonusExpiresAt,
		UpdatedAt:      now,
	}
	return plan, nil
}

// PlanAdjust computes an administrative correction, which may be signed in
// either direction but may never drive a balance negative.
func PlanAdjust(bal Balance, delta, bonusDelta int, now time.Time) (Plan, error) {
	if delta == 0 && bonusDelta == 0 {
		return Plan{}, ErrInvalidAmount
	}

	balanceAfter := bal.Balance + delta
	bonusAfter := bal.BonusBalance + bonusDelta
	if balanceAfter < 0 || bonusAfter < 0 {
		return Plan{}, ErrInsufficientCoins
	}

	bonusExpiresAt := bal.BonusExpiresAt
	if bonusAfter == 0 {
		bonusExpiresAt = nil
	}

	return Plan{
		Entry: LedgerEntry{
			UserID:            bal.UserID,
			Delta:             delta,
			BonusDelta:        bonusDelta,
			Kind:              KindAdjust,
			BalanceAfter:      balanceAfter,
			BonusBalanceAfter: bonusAfter,
			CreatedAt:         now,
		},
		Result: Balance{
			UserID:         bal.UserID,
			Balance:        balanceAfter,
			BonusBalance:   bonusAfter,
			BonusExpiresAt: bonusExpiresAt,
			UpdatedAt:      now,
		},
	}, nil
}

// PlanBonusExpiry computes the write-off the nightly job performs. The second
// return value is false when there is nothing to expire, so the job can skip
// the user without writing a no-op ledger row.
func PlanBonusExpiry(bal Balance, now time.Time) (Plan, bool) {
	if !expired(bal, now) || bal.BonusBalance == 0 {
		return Plan{}, false
	}

	return Plan{
		Entry: LedgerEntry{
			UserID:            bal.UserID,
			Delta:             0,
			BonusDelta:        -bal.BonusBalance,
			Kind:              KindBonusExpire,
			BalanceAfter:      bal.Balance,
			BonusBalanceAfter: 0,
			CreatedAt:         now,
		},
		Result: Balance{
			UserID:         bal.UserID,
			Balance:        bal.Balance,
			BonusBalance:   0,
			BonusExpiresAt: nil,
			UpdatedAt:      now,
		},
	}, true
}

func expired(bal Balance, now time.Time) bool {
	return bal.BonusExpiresAt != nil && !bal.BonusExpiresAt.After(now)
}
