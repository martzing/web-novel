package wallet_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/mokchan/webnovel-backend/internal/domain/wallet"
	"github.com/mokchan/webnovel-backend/internal/entities"
	walletrepo "github.com/mokchan/webnovel-backend/internal/repository/wallet"
	"github.com/mokchan/webnovel-backend/test/makeme"
)

// The service namespaces idempotency keys per operation, so in practice two
// different operations never present the same stored key. These tests pin the
// layer beneath that convention: Apply itself must decide replay-versus-conflict
// from the target the key was used against, so the guarantee survives a caller
// that derives its own key — the auto-unlock job already does.
func TestApply_SameKeyDifferentRefTypeConflicts(t *testing.T) {
	m := makeme.New(t)
	repo := walletrepo.New(m.DB)
	ctx := context.Background()

	reader := m.ANewUser().Please()
	fund(t, m, reader.ID, 500)

	// ref_id 42 under two different ref_types is two different targets. Before
	// sameTarget compared ref_type, these compared equal and the second call
	// replayed the first receipt — the reader kept the coins and got nothing.
	ref := int64(42)
	first, err := repo.Apply(ctx, spendCommand(reader.ID, "k", domain.RefChapterUnlock, &ref, 10))
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}

	_, err = repo.Apply(ctx, spendCommand(reader.ID, "k", domain.RefArcBundle, &ref, 10))
	if !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("err = %v, want ErrIdempotencyConflict", err)
	}

	// And the same target still replays rather than double-charging.
	replay, err := repo.Apply(ctx, spendCommand(reader.ID, "k", domain.RefChapterUnlock, &ref, 10))
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !replay.Replayed || replay.Ledger.ID != first.Ledger.ID {
		t.Fatalf("replay = %+v, want the first receipt back", replay)
	}
}

func TestReplayByKey_ReportsAppliedKeysAndIgnoresUnusedOnes(t *testing.T) {
	m := makeme.New(t)
	repo := walletrepo.New(m.DB)
	ctx := context.Background()

	reader := m.ANewUser().Please()
	fund(t, m, reader.ID, 500)

	ref := int64(7)
	applied, err := repo.Apply(ctx, spendCommand(reader.ID, "used", domain.RefChapterUnlock, &ref, 10))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	found, err := repo.ReplayByKey(ctx, reader.ID, "used")
	if err != nil {
		t.Fatalf("replay by key: %v", err)
	}
	if found == nil || found.Ledger.ID != applied.Ledger.ID || !found.Replayed {
		t.Fatalf("ReplayByKey = %+v, want the applied receipt marked as a replay", found)
	}

	missing, err := repo.ReplayByKey(ctx, reader.ID, "never-used")
	if err != nil {
		t.Fatalf("replay by key: %v", err)
	}
	if missing != nil {
		t.Fatalf("ReplayByKey = %+v, want nil for an unused key", missing)
	}

	// A key is scoped to its owner: another reader's key is not a replay.
	other := m.ANewUser().Please()
	crossed, err := repo.ReplayByKey(ctx, other.ID, "used")
	if err != nil {
		t.Fatalf("replay by key: %v", err)
	}
	if crossed != nil {
		t.Fatal("one reader's idempotency key replayed for another reader")
	}
}

func spendCommand(userID int64, key, refType string, refID *int64, amount int) domain.Command {
	return domain.Command{
		UserID:         userID,
		IdempotencyKey: key,
		Now:            time.Now(),
		Op: domain.Operation{
			Kind:    domain.KindSpendUnlock,
			Amount:  amount,
			RefType: refType,
			RefID:   refID,
		},
	}
}

// fund seeds a balance and the ledger row that backs it, so the
// sum(delta) == balance invariant holds from the start.
func fund(t *testing.T, m *makeme.MakeMe, userID int64, coins int) {
	t.Helper()
	m.ANewWalletBalance().With(func(w *entities.WalletBalance) {
		w.UserID = userID
		w.Balance = coins
	}).Please()
	m.ANewCoinLedgerEntry().With(func(e *entities.CoinLedgerEntry) {
		e.UserID = userID
		e.Kind = entities.LedgerTopup
		e.Delta = coins
		e.BalanceAfter = coins
	}).Please()
}
