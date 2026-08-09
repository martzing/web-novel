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
	sale, err := s.repo.ChapterForSale(ctx, chapterID)
	if err != nil {
		return nil, err
	}
	if sale.PriceCoins <= 0 {
		return nil, domain.ErrChapterNotForSale
	}

	// A chapter inside its early-access window is reserved for auto-unlock
	// subscribers. Without this, anyone could simply pay to defeat the
	// exclusivity and the perk would mean nothing.
	if err := s.assertBuyable(ctx, userID, *sale); err != nil {
		return nil, err
	}

	refID := chapterID
	return s.repo.Apply(ctx, domain.Command{
		UserID:         userID,
		IdempotencyKey: namespaced("unlock", idempotencyKey),
		Now:            s.now(),
		Op: domain.Operation{
			Kind:    domain.KindSpendUnlock,
			Amount:  sale.PriceCoins,
			RefType: domain.RefChapterUnlock,
			RefID:   &refID,
		},
		Child: s.unlockChild(userID, *sale, sale.PriceCoins),
	})
}

// QuoteArcBundle prices the chapters of an arc the reader does not yet own.
func (s *Service) QuoteArcBundle(ctx context.Context, userID, arcID int64) (*ArcQuote, error) {
	arc, err := s.repo.ArcChaptersForSale(ctx, arcID)
	if err != nil {
		return nil, err
	}
	if !arc.SellByArc {
		return nil, domain.ErrArcNotForSale
	}

	ids := make([]int64, 0, len(arc.Chapters))
	for _, c := range arc.Chapters {
		ids = append(ids, c.ChapterID)
	}
	owned, err := s.repo.ListUnlockedChapterIDs(ctx, userID, ids)
	if err != nil {
		return nil, err
	}

	now := s.now()
	subscribed := false
	if s.hasEarlyChapters(arc.Chapters, now) {
		if subscribed, err = s.repo.IsSubscribed(ctx, userID, arc.NovelID); err != nil {
			return nil, err
		}
	}

	// The reader pays only for what they still need, and never for a chapter
	// still reserved for subscribers.
	wanted := make([]domain.ChapterSale, 0, len(arc.Chapters))
	prices := make([]int, 0, len(arc.Chapters))
	for _, c := range arc.Chapters {
		if owned[c.ChapterID] {
			continue
		}
		if !c.IsPublic(now) && !subscribed {
			continue
		}
		wanted = append(wanted, c)
		prices = append(prices, c.PriceCoins)
	}

	quote, err := domain.QuoteBundle(prices, domain.ArcBundleDiscountPercent)
	if err != nil {
		return nil, err
	}

	items := make([]domain.ChildItem, 0, len(wanted))
	for i, c := range wanted {
		item := domain.ChildItem{
			ChapterID: c.ChapterID,
			ListPrice: c.PriceCoins,
			Coins:     quote.PerChapter[i],
		}
		if c.TranslatorID != nil && *c.TranslatorID != userID {
			item.WriterID = *c.TranslatorID
			item.NetCoins = domain.NetCoins(item.Coins, s.feePercent)
		}
		items = append(items, item)
	}

	return &ArcQuote{
		ArcID:    arc.ArcID,
		NovelID:  arc.NovelID,
		ArcNo:    arc.ArcNo,
		Name:     arc.Name,
		Quote:    quote,
		Chapters: wanted,
		Items:    items,
	}, nil
}

// UnlockArc buys every chapter in the quote in one atomic operation.
func (s *Service) UnlockArc(ctx context.Context, userID, arcID int64, idempotencyKey string) (*domain.Receipt, error) {
	key := namespaced("arc_unlock", idempotencyKey)

	// A bundle must recognise a retry before it quotes. Once the first attempt
	// commits, every chapter is owned and a re-quote would raise
	// ErrArcAlreadyOwned — so a client retrying a timed-out request would get a
	// 409 instead of its receipt. Apply's own replay check never gets reached.
	if replay, err := s.repo.ReplayByKey(ctx, userID, key); err != nil {
		return nil, err
	} else if replay != nil {
		return replay, nil
	}

	quote, err := s.QuoteArcBundle(ctx, userID, arcID)
	if err != nil {
		return nil, err
	}

	refID := arcID
	return s.repo.Apply(ctx, domain.Command{
		UserID:         userID,
		IdempotencyKey: key,
		Now:            s.now(),
		Op: domain.Operation{
			Kind:    domain.KindSpendUnlock,
			Amount:  quote.Quote.Total,
			RefType: domain.RefArcBundle,
			RefID:   &refID,
		},
		Child: domain.ChildWrite{Kind: domain.ChildArcBundle, Items: quote.Items},
	})
}

// TipChapter sends coins to a chapter's translator.
//
// Tips draw on purchased coins only — see wallet.PlanSpendPaidOnly — and they
// credit writer_earnings rather than the translator's spendable wallet, so the
// command still touches exactly one wallet row.
func (s *Service) TipChapter(ctx context.Context, userID, chapterID int64, coins int, idempotencyKey string) (*domain.Receipt, error) {
	amount := coins
	if err := domain.ValidateTip(amount); err != nil {
		return nil, err
	}

	sale, err := s.repo.ChapterForSale(ctx, chapterID)
	if err != nil {
		return nil, err
	}
	if !sale.TipsEnabled {
		return nil, domain.ErrTipsDisabled
	}
	if sale.TranslatorID == nil {
		return nil, domain.ErrNotFound
	}
	if *sale.TranslatorID == userID {
		return nil, domain.ErrCannotTipSelf
	}

	refID := chapterID
	return s.repo.Apply(ctx, domain.Command{
		UserID:         userID,
		IdempotencyKey: namespaced("tip", idempotencyKey),
		Now:            s.now(),
		Op: domain.Operation{
			Kind:    domain.KindTip,
			Amount:  amount,
			RefType: domain.RefChapterTip,
			RefID:   &refID,
		},
		Child: domain.ChildWrite{
			Kind:      domain.ChildTip,
			ChapterID: chapterID,
			WriterID:  *sale.TranslatorID,
			NetCoins:  domain.NetCoins(amount, s.feePercent),
		},
	})
}

// assertBuyable rejects a sale of a chapter still inside its early-access
// window to a reader who is not subscribed.
func (s *Service) assertBuyable(ctx context.Context, userID int64, sale domain.ChapterSale) error {
	if sale.IsPublic(s.now()) {
		return nil
	}
	subscribed, err := s.repo.IsSubscribed(ctx, userID, sale.NovelID)
	if err != nil {
		return err
	}
	if !subscribed {
		return domain.ErrEarlyAccessOnly
	}
	return nil
}

func (s *Service) hasEarlyChapters(chapters []domain.ChapterSale, now time.Time) bool {
	for _, c := range chapters {
		if !c.IsPublic(now) {
			return true
		}
	}
	return false
}

// unlockChild builds the child write for a single-chapter unlock. A translator
// unlocking their own chapter is not credited, so they cannot pay themselves.
func (s *Service) unlockChild(userID int64, sale domain.ChapterSale, coins int) domain.ChildWrite {
	child := domain.ChildWrite{Kind: domain.ChildChapterUnlock, ChapterID: sale.ChapterID}
	if sale.TranslatorID != nil && *sale.TranslatorID != userID {
		child.WriterID = *sale.TranslatorID
		child.NetCoins = domain.NetCoins(coins, s.feePercent)
	}
	return child
}

// ArcQuote is what the reader is shown before buying a bundle, and what the
// purchase then applies.
type ArcQuote struct {
	ArcID    int64
	NovelID  int64
	ArcNo    int
	Name     string
	Quote    domain.BundleQuote
	Chapters []domain.ChapterSale
	Items    []domain.ChildItem
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

// ListSubscriptions returns the reader's auto-unlock opt-ins.
func (s *Service) ListSubscriptions(ctx context.Context, userID int64) ([]domain.Subscription, error) {
	return s.repo.ListSubscriptions(ctx, userID)
}

// SetSubscription turns auto-unlock on or off for one novel.
func (s *Service) SetSubscription(ctx context.Context, userID, novelID int64, active bool, maxCoins int) (*domain.Subscription, error) {
	if maxCoins < 0 {
		return nil, domain.ErrInvalidAmount
	}
	return s.repo.UpsertSubscription(ctx, domain.Subscription{
		UserID:             userID,
		NovelID:            novelID,
		Active:             active,
		MaxCoinsPerChapter: maxCoins,
	})
}

// RemoveSubscription cancels auto-unlock. Chapters already unlocked stay
// unlocked — the reader paid for them.
func (s *Service) RemoveSubscription(ctx context.Context, userID, novelID int64) error {
	return s.repo.DeleteSubscription(ctx, userID, novelID)
}

// IsSubscribed satisfies the reading and catalog contexts' Subscriptions port.
func (s *Service) IsSubscribed(ctx context.Context, userID, novelID int64) (bool, error) {
	return s.repo.IsSubscribed(ctx, userID, novelID)
}

// AutoUnlockCandidates lists subscriber/chapter pairs the fan-out may debit.
func (s *Service) AutoUnlockCandidates(ctx context.Context, publishedAfter, retryBefore time.Time, maxAttempts, limit int) ([]domain.AutoUnlockCandidate, error) {
	return s.repo.AutoUnlockCandidates(ctx, publishedAfter, retryBefore, maxAttempts, limit)
}

// RecordAutoUnlockAttempt stores what the fan-out decided for one pair.
func (s *Service) RecordAutoUnlockAttempt(ctx context.Context, a domain.AutoUnlockAttempt) error {
	return s.repo.RecordAutoUnlockAttempt(ctx, a)
}

// AutoUnlockChapter debits a subscriber for a newly published chapter.
//
// The idempotency key is derived from the pair rather than supplied, so a
// repeated fan-out — or two replicas claiming the same candidate — replays
// instead of charging twice.
func (s *Service) AutoUnlockChapter(ctx context.Context, userID, chapterID int64) (*domain.Receipt, error) {
	sale, err := s.repo.ChapterForSale(ctx, chapterID)
	if err != nil {
		return nil, err
	}
	if sale.PriceCoins <= 0 {
		return nil, domain.ErrChapterNotForSale
	}

	refID := chapterID
	return s.repo.Apply(ctx, domain.Command{
		UserID:         userID,
		IdempotencyKey: fmt.Sprintf("autounlock:%d:%d", userID, chapterID),
		Now:            s.now(),
		Op: domain.Operation{
			Kind:    domain.KindSpendUnlock,
			Amount:  sale.PriceCoins,
			RefType: domain.RefAutoUnlock,
			RefID:   &refID,
		},
		Child: s.unlockChild(userID, *sale, sale.PriceCoins),
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
