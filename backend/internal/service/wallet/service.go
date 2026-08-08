// Package wallet is the application layer for the coin economy.
package wallet

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
	domain "github.com/mokchan/webnovel-backend/internal/domain/wallet"
)

// Service orchestrates coin use cases on top of the single write path.
type Service struct {
	repo       domain.Repository
	bonusTTL   time.Duration
	feePercent int
	now        func() time.Time
}

// New wires the service.
func New(repo domain.Repository, bonusTTL time.Duration, feePercent int, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, bonusTTL: bonusTTL, feePercent: feePercent, now: now}
}

// GetBalance returns the caller's wallet.
func (s *Service) GetBalance(ctx context.Context, userID int64) (*domain.Balance, error) {
	return s.repo.GetBalance(ctx, userID)
}

// ListLedger returns a page of the caller's transaction history.
func (s *Service) ListLedger(ctx context.Context, userID int64, p page.Page) ([]domain.LedgerEntry, string, error) {
	return s.repo.ListLedger(ctx, userID, p.Normalize(20, 100))
}

// ListPacks returns the purchasable coin bundles.
func (s *Service) ListPacks(ctx context.Context) ([]domain.Pack, error) {
	return s.repo.ListPacks(ctx)
}

// CreatePurchase opens a pending mock purchase for a coin pack.
func (s *Service) CreatePurchase(ctx context.Context, userID, packID int64, idempotencyKey string) (*domain.Purchase, error) {
	pack, err := s.repo.GetPack(ctx, packID)
	if err != nil {
		return nil, err
	}

	// provider_ref is NOT NULL and UNIQUE (provider, provider_ref), so the
	// service must mint one; a real gateway would supply it.
	providerRef, err := mockProviderRef()
	if err != nil {
		return nil, err
	}

	return s.repo.CreatePurchase(ctx, domain.Purchase{
		UserID:       userID,
		PackID:       pack.ID,
		Provider:     domain.ProviderMock,
		ProviderRef:  providerRef,
		AmountSatang: pack.PriceSatang,
		Currency:     pack.Currency,
		Status:       domain.PurchasePending,
	}, namespaced("purchase", idempotencyKey))
}

// CompletePurchase credits the wallet for a pending purchase. It is the mock
// stand-in for a payment webhook and uses the same ledger helper a real one
// would.
func (s *Service) CompletePurchase(ctx context.Context, userID, purchaseID int64, idempotencyKey string) (*domain.Receipt, error) {
	purchase, err := s.repo.GetPurchase(ctx, purchaseID)
	if err != nil {
		return nil, err
	}
	if purchase.UserID != userID {
		// Do not distinguish "not yours" from "does not exist": that would let
		// a caller enumerate other people's purchases.
		return nil, domain.ErrNotFound
	}

	pack, err := s.repo.GetPack(ctx, purchase.PackID)
	if err != nil {
		return nil, err
	}

	refID := purchase.ID
	return s.repo.Apply(ctx, domain.Command{
		UserID:         userID,
		IdempotencyKey: namespaced("topup", idempotencyKey),
		Now:            s.now(),
		Op: domain.Operation{
			Kind:       domain.KindTopup,
			Coins:      pack.Coins,
			BonusCoins: pack.BonusCoins,
			BonusTTL:   s.bonusTTL,
			RefType:    "purchase",
			RefID:      &refID,
		},
		Child: domain.ChildWrite{
			Kind:       domain.ChildPurchaseComplete,
			PurchaseID: purchase.ID,
		},
	})
}

// FailPurchase marks a pending mock purchase as failed. No wallet change.
func (s *Service) FailPurchase(ctx context.Context, userID, purchaseID int64) (*domain.Purchase, error) {
	return s.repo.FailPurchase(ctx, purchaseID, userID)
}

// UnlockChapter spends coins to grant permanent access to a chapter.
func (s *Service) UnlockChapter(ctx context.Context, userID, chapterID int64, idempotencyKey string) (*domain.Receipt, error) {
	price, translatorID, err := s.repo.ChapterForUnlock(ctx, chapterID)
	if err != nil {
		return nil, err
	}
	if price <= 0 {
		return nil, domain.ErrChapterNotForSale
	}

	child := domain.ChildWrite{Kind: domain.ChildChapterUnlock, ChapterID: chapterID}
	// A translator unlocking their own chapter would otherwise pay themselves.
	if translatorID != nil && *translatorID != userID {
		child.WriterID = *translatorID
		child.NetCoins = domain.NetCoins(price, s.feePercent)
	}

	refID := chapterID
	return s.repo.Apply(ctx, domain.Command{
		UserID:         userID,
		IdempotencyKey: namespaced("unlock", idempotencyKey),
		Now:            s.now(),
		Op: domain.Operation{
			Kind:    domain.KindSpendUnlock,
			Amount:  price,
			RefType: "chapter_unlock",
			RefID:   &refID,
		},
		Child: child,
	})
}

// Adjust applies an administrative correction, recorded with the acting admin
// and a reason.
func (s *Service) Adjust(ctx context.Context, targetUserID int64, delta, bonusDelta int, reason string, actorUserID int64, idempotencyKey string) (*domain.Receipt, error) {
	return s.repo.Apply(ctx, domain.Command{
		UserID:         targetUserID,
		IdempotencyKey: namespaced("adjust", idempotencyKey),
		Now:            s.now(),
		Op: domain.Operation{
			Kind:       domain.KindAdjust,
			Delta:      delta,
			BonusDelta: bonusDelta,
			RefType:    "admin_adjust",
		},
		ActorUserID: &actorUserID,
		Reason:      reason,
	})
}

// ExpireBonus writes off a user's lapsed bonus. The idempotency key is stamped
// with the date so a re-run on the same day is a no-op.
func (s *Service) ExpireBonus(ctx context.Context, userID int64, now time.Time) (*domain.Receipt, error) {
	return s.repo.Apply(ctx, domain.Command{
		UserID:         userID,
		IdempotencyKey: fmt.Sprintf("bonus_expire:%d:%s", userID, now.Format("2006-01-02")),
		Now:            now,
		Op:             domain.Operation{Kind: domain.KindBonusExpire, RefType: "bonus_expire"},
	})
}

// ExpiryCandidates lists users whose bonus has lapsed.
func (s *Service) ExpiryCandidates(ctx context.Context, before time.Time, limit int) ([]int64, error) {
	return s.repo.ExpiryCandidates(ctx, before, limit)
}

// IsChapterUnlocked satisfies the reading context's Entitlements port.
func (s *Service) IsChapterUnlocked(ctx context.Context, userID, chapterID int64) (bool, error) {
	return s.repo.IsChapterUnlocked(ctx, userID, chapterID)
}

// ListEarnings returns a page of the translator's per-unlock earnings.
func (s *Service) ListEarnings(ctx context.Context, writerID int64, p page.Page) ([]domain.Earning, string, error) {
	return s.repo.ListEarnings(ctx, writerID, p.Normalize(20, 100))
}

// RequestPayout opens a fiat withdrawal request, capped at the unpaid balance.
func (s *Service) RequestPayout(ctx context.Context, writerID int64, amountSatang int) (*domain.Payout, error) {
	if amountSatang <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	available, err := s.repo.SumUnpaidEarnings(ctx, writerID)
	if err != nil {
		return nil, err
	}
	if amountSatang > available {
		return nil, domain.ErrInsufficientCoins
	}
	return s.repo.CreatePayout(ctx, domain.Payout{WriterID: writerID, AmountSatang: amountSatang})
}

// AvailableEarnings is the amount a writer may still withdraw.
func (s *Service) AvailableEarnings(ctx context.Context, writerID int64) (int, error) {
	return s.repo.SumUnpaidEarnings(ctx, writerID)
}

// namespaced prefixes a client key with the operation, so one key reused across
// endpoints cannot be mistaken for a replay of a different operation.
func namespaced(op, key string) string {
	if key == "" {
		return ""
	}
	return op + ":" + key
}

func mockProviderRef() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("wallet: read random: %w", err)
	}
	return "mock_" + hex.EncodeToString(buf), nil
}
