package makeme

import (
	"github.com/mokchan/webnovel-backend/internal/entities"
)

// ANewWalletBalance creates a wallet balance builder. Set UserID via With.
func (m *MakeMe) ANewWalletBalance() *Builder[entities.WalletBalance] {
	m.t.Helper()
	model := &entities.WalletBalance{
		Balance:      0,
		BonusBalance: 0,
	}
	return newBuilder(m, model, func(row *entities.WalletBalance) map[string]any {
		return map[string]any{"user_id": row.UserID}
	})
}

// ANewCoinLedgerEntry creates a ledger row builder. Set UserID via With.
func (m *MakeMe) ANewCoinLedgerEntry() *Builder[entities.CoinLedgerEntry] {
	m.t.Helper()
	seq := m.next()
	model := &entities.CoinLedgerEntry{
		Delta:             0,
		BonusDelta:        0,
		Kind:              entities.LedgerTopup,
		BalanceAfter:      0,
		BonusBalanceAfter: 0,
		CreatedAt:         timeForSequence(seq),
	}
	return newBuilder(m, model, func(row *entities.CoinLedgerEntry) map[string]any {
		return map[string]any{"id": row.ID}
	})
}

// ANewCoinPack creates a coin pack builder.
func (m *MakeMe) ANewCoinPack() *Builder[entities.CoinPack] {
	m.t.Helper()
	seq := m.next()
	model := &entities.CoinPack{
		Coins:       100,
		BonusCoins:  0,
		PriceSatang: 3500,
		Currency:    "THB",
		Active:      true,
		SortNo:      int16(seq % 100),
	}
	return newBuilder(m, model, func(row *entities.CoinPack) map[string]any {
		return map[string]any{"id": row.ID}
	})
}

// ANewPurchase creates a purchase builder. Set UserID and PackID via With.
func (m *MakeMe) ANewPurchase() *Builder[entities.Purchase] {
	m.t.Helper()
	seq := m.next()
	model := &entities.Purchase{
		Provider:     "mock",
		ProviderRef:  fixtureCode("mock_", seq, 12),
		AmountSatang: 3500,
		Currency:     "THB",
		Status:       entities.PurchasePending,
	}
	return newBuilder(m, model, func(row *entities.Purchase) map[string]any {
		return map[string]any{"id": row.ID}
	})
}

// ANewChapterUnlock creates an unlock builder. Set UserID, ChapterID and
// LedgerID via With.
func (m *MakeMe) ANewChapterUnlock() *Builder[entities.ChapterUnlock] {
	m.t.Helper()
	model := &entities.ChapterUnlock{CoinsSpent: 5}
	return newBuilder(m, model, func(row *entities.ChapterUnlock) map[string]any {
		return map[string]any{"user_id": row.UserID, "chapter_id": row.ChapterID}
	})
}

// ANewWriterEarning creates an earnings builder.
func (m *MakeMe) ANewWriterEarning() *Builder[entities.WriterEarning] {
	m.t.Helper()
	model := &entities.WriterEarning{GrossCoins: 5, NetCoins: 4}
	return newBuilder(m, model, func(row *entities.WriterEarning) map[string]any {
		return map[string]any{"id": row.ID}
	})
}

// ANewPayout creates a payout-request builder. Set WriterID via With.
func (m *MakeMe) ANewPayout() *Builder[entities.Payout] {
	m.t.Helper()
	model := &entities.Payout{
		AmountSatang: 10000,
		Status:       entities.PayoutRequested,
	}
	return newBuilder(m, model, func(row *entities.Payout) map[string]any {
		return map[string]any{"id": row.ID}
	})
}
