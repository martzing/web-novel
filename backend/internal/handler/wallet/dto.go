package wallet

import (
	"time"

	domain "github.com/mokchan/webnovel-backend/internal/domain/wallet"
)

// BalanceResponse is the wallet card payload.
type BalanceResponse struct {
	Balance        int    `json:"balance"`
	BonusBalance   int    `json:"bonus_balance"`
	BonusExpiresAt string `json:"bonus_expires_at,omitempty"`
	Total          int    `json:"total"`
}

// LedgerEntryResponse is one transaction-history row.
type LedgerEntryResponse struct {
	ID                int64  `json:"id,string"`
	Delta             int    `json:"delta"`
	BonusDelta        int    `json:"bonus_delta"`
	Kind              string `json:"kind"`
	RefType           string `json:"ref_type,omitempty"`
	RefID             *int64 `json:"ref_id,string,omitempty"`
	BalanceAfter      int    `json:"balance_after"`
	BonusBalanceAfter int    `json:"bonus_balance_after"`
	Reason            string `json:"reason,omitempty"`
	CreatedAt         string `json:"created_at"`
}

// PackResponse is a purchasable coin bundle.
type PackResponse struct {
	ID          int64  `json:"id,string"`
	Coins       int    `json:"coins"`
	BonusCoins  int    `json:"bonus_coins"`
	PriceSatang int    `json:"price_satang"`
	Currency    string `json:"currency"`
	IsBestValue bool   `json:"is_best_value"`
}

// PurchaseResponse is returned when a mock purchase is opened.
type PurchaseResponse struct {
	PurchaseID      int64  `json:"purchase_id,string"`
	Status          string `json:"status"`
	AmountSatang    int    `json:"amount_satang"`
	Currency        string `json:"currency"`
	Provider        string `json:"provider"`
	MockCheckoutURL string `json:"mock_checkout_url"`
}

// ReceiptResponse is the outcome of a coin mutation.
type ReceiptResponse struct {
	LedgerID          int64 `json:"ledger_id,string"`
	CoinsSpent        int   `json:"coins_spent"`
	Delta             int   `json:"delta"`
	BonusDelta        int   `json:"bonus_delta"`
	BalanceAfter      int   `json:"balance_after"`
	BonusBalanceAfter int   `json:"bonus_balance_after"`
	Replayed          bool  `json:"replayed"`
}

// EarningResponse is one translator earning row.
type EarningResponse struct {
	ID         int64  `json:"id,string"`
	ChapterID  int64  `json:"chapter_id,string"`
	GrossCoins int    `json:"gross_coins"`
	NetCoins   int    `json:"net_coins"`
	CreatedAt  string `json:"created_at"`
}

// PayoutResponse is a fiat withdrawal request.
type PayoutResponse struct {
	ID           int64  `json:"id,string"`
	AmountSatang int    `json:"amount_satang"`
	Status       string `json:"status"`
	RequestedAt  string `json:"requested_at"`
}

type createPurchaseRequest struct {
	PackID int64 `json:"pack_id,string"`
}

type walletAdjustRequest struct {
	UserID     int64  `json:"user_id,string"`
	Delta      int    `json:"delta"`
	BonusDelta int    `json:"bonus_delta"`
	Reason     string `json:"reason"`
}

type payoutRequest struct {
	AmountSatang int `json:"amount_satang"`
}

func toBalanceResponse(b domain.Balance) BalanceResponse {
	out := BalanceResponse{
		Balance:      b.Balance,
		BonusBalance: b.BonusBalance,
		Total:        b.Total(),
	}
	if b.BonusExpiresAt != nil {
		out.BonusExpiresAt = b.BonusExpiresAt.UTC().Format(time.RFC3339)
	}
	return out
}

func toLedgerResponses(entries []domain.LedgerEntry) []LedgerEntryResponse {
	out := make([]LedgerEntryResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, LedgerEntryResponse{
			ID:                e.ID,
			Delta:             e.Delta,
			BonusDelta:        e.BonusDelta,
			Kind:              string(e.Kind),
			RefType:           e.RefType,
			RefID:             e.RefID,
			BalanceAfter:      e.BalanceAfter,
			BonusBalanceAfter: e.BonusBalanceAfter,
			Reason:            e.Reason,
			CreatedAt:         e.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func toPackResponses(packs []domain.Pack) []PackResponse {
	out := make([]PackResponse, 0, len(packs))
	for _, p := range packs {
		out = append(out, PackResponse{
			ID:          p.ID,
			Coins:       p.Coins,
			BonusCoins:  p.BonusCoins,
			PriceSatang: p.PriceSatang,
			Currency:    p.Currency,
			IsBestValue: p.IsBestValue,
		})
	}
	return out
}

func toReceiptResponse(r domain.Receipt) ReceiptResponse {
	// CoinsSpent describes a debit, so a credit reports zero rather than a
	// negative "spend".
	spent := max(-(r.Ledger.Delta + r.Ledger.BonusDelta), 0)

	return ReceiptResponse{
		LedgerID:          r.Ledger.ID,
		CoinsSpent:        spent,
		Delta:             r.Ledger.Delta,
		BonusDelta:        r.Ledger.BonusDelta,
		BalanceAfter:      r.Balance.Balance,
		BonusBalanceAfter: r.Balance.BonusBalance,
		Replayed:          r.Replayed,
	}
}

func toEarningResponses(earnings []domain.Earning) []EarningResponse {
	out := make([]EarningResponse, 0, len(earnings))
	for _, e := range earnings {
		out = append(out, EarningResponse{
			ID:         e.ID,
			ChapterID:  e.ChapterID,
			GrossCoins: e.GrossCoins,
			NetCoins:   e.NetCoins,
			CreatedAt:  e.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func toPayoutResponse(p domain.Payout) PayoutResponse {
	return PayoutResponse{
		ID:           p.ID,
		AmountSatang: p.AmountSatang,
		Status:       p.Status,
		RequestedAt:  p.RequestedAt.UTC().Format(time.RFC3339),
	}
}
