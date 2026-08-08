package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mokchan/webnovel-backend/internal/crypto/argon2id"
	domain "github.com/mokchan/webnovel-backend/internal/domain/identity"
)

// fakeRepo wires only the methods a given test exercises.
type fakeRepo struct {
	createUser        func(ctx context.Context, u domain.User, hash string) (*domain.User, error)
	getUserByID       func(ctx context.Context, id int64) (*domain.User, error)
	getUserByEmail    func(ctx context.Context, email string) (*domain.User, string, error)
	updateUser        func(ctx context.Context, id int64, patch domain.UserPatch) (*domain.User, error)
	getPrefs          func(ctx context.Context, userID int64) (*domain.Prefs, error)
	upsertPrefs       func(ctx context.Context, p domain.Prefs) (*domain.Prefs, error)
	listGenrePrefs    func(ctx context.Context, userID int64) ([]domain.GenrePref, error)
	replaceGenrePrefs func(ctx context.Context, userID int64, prefs []domain.GenrePref) error
}

func (f *fakeRepo) CreateUser(ctx context.Context, u domain.User, hash string) (*domain.User, error) {
	return f.createUser(ctx, u, hash)
}
func (f *fakeRepo) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	return f.getUserByID(ctx, id)
}
func (f *fakeRepo) GetUserByEmail(ctx context.Context, email string) (*domain.User, string, error) {
	return f.getUserByEmail(ctx, email)
}
func (f *fakeRepo) UpdateUser(ctx context.Context, id int64, patch domain.UserPatch) (*domain.User, error) {
	return f.updateUser(ctx, id, patch)
}
func (f *fakeRepo) GetPrefs(ctx context.Context, userID int64) (*domain.Prefs, error) {
	return f.getPrefs(ctx, userID)
}
func (f *fakeRepo) UpsertPrefs(ctx context.Context, p domain.Prefs) (*domain.Prefs, error) {
	return f.upsertPrefs(ctx, p)
}
func (f *fakeRepo) ListGenrePrefs(ctx context.Context, userID int64) ([]domain.GenrePref, error) {
	return f.listGenrePrefs(ctx, userID)
}
func (f *fakeRepo) ReplaceGenrePrefs(ctx context.Context, userID int64, prefs []domain.GenrePref) error {
	return f.replaceGenrePrefs(ctx, userID, prefs)
}

// fakeSessions records what the service asked the store to do.
type fakeSessions struct {
	created  []domain.Session
	revoked  int
	rotateFn func(ctx context.Context, tokenHash []byte, next domain.Session, nextHash []byte, now time.Time) (*domain.Session, error)
}

func (f *fakeSessions) Create(_ context.Context, s domain.Session, _ []byte) (*domain.Session, error) {
	f.created = append(f.created, s)
	s.ID = int64(len(f.created))
	return &s, nil
}
func (f *fakeSessions) Rotate(ctx context.Context, tokenHash []byte, next domain.Session, nextHash []byte, now time.Time) (*domain.Session, error) {
	return f.rotateFn(ctx, tokenHash, next, nextHash, now)
}
func (f *fakeSessions) RevokeFamilyByToken(context.Context, []byte, time.Time) error {
	f.revoked++
	return nil
}
func (f *fakeSessions) RevokeAllForUser(context.Context, int64, time.Time) error { return nil }
func (f *fakeSessions) DeleteExpired(context.Context, time.Time) (int64, error)  { return 0, nil }

// countingHasher records how many verifications ran, which is how the timing
// defence is asserted without measuring wall-clock time.
type countingHasher struct {
	verifyCalls int
	accept      bool
}

func (h *countingHasher) Hash(string) (string, error) { return "hashed", nil }
func (h *countingHasher) Verify(string, string) bool {
	h.verifyCalls++
	return h.accept
}

type stubIssuer struct{}

func (stubIssuer) Issue(_ int64, _ []string, now time.Time) (string, time.Time, error) {
	return "access-token", now.Add(15 * time.Minute), nil
}

func newService(repo domain.Repository, sessions domain.SessionStore, hasher domain.PasswordHasher) *Service {
	now := func() time.Time { return time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC) }
	return New(repo, sessions, hasher, stubIssuer{}, 30*24*time.Hour, now)
}

// I-AUTH-02 — an unknown email and a wrong password must be indistinguishable.
func TestLogin_SameErrorForUnknownEmailAndBadPassword(t *testing.T) {
	activeUser := &domain.User{ID: 1, Email: "reader@example.com", Status: domain.StatusActive}

	t.Run("unknown email", func(t *testing.T) {
		hasher := &countingHasher{accept: false}
		repo := &fakeRepo{
			getUserByEmail: func(context.Context, string) (*domain.User, string, error) {
				return nil, "", domain.ErrNotFound
			},
		}

		_, err := newService(repo, &fakeSessions{}, hasher).
			Login(context.Background(), "nobody@example.com", "hunter2hunter2", "")

		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("error = %v, want ErrInvalidCredentials", err)
		}
		// The dummy verification is what keeps the unknown-email branch from
		// returning measurably faster than the wrong-password branch.
		if hasher.verifyCalls != 1 {
			t.Fatalf("verify calls = %d, want 1 (dummy hash comparison)", hasher.verifyCalls)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		hasher := &countingHasher{accept: false}
		repo := &fakeRepo{
			getUserByEmail: func(context.Context, string) (*domain.User, string, error) {
				return activeUser, "stored-hash", nil
			},
		}

		_, err := newService(repo, &fakeSessions{}, hasher).
			Login(context.Background(), "reader@example.com", "wrong", "")

		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("error = %v, want ErrInvalidCredentials", err)
		}
		if hasher.verifyCalls != 1 {
			t.Fatalf("verify calls = %d, want 1", hasher.verifyCalls)
		}
	})
}

// The dummy hash must be a genuine argon2id encoding, or Verify short-circuits
// on a parse failure and the timing defence does no work at all.
func TestDummyHash_IsParseable(t *testing.T) {
	if _, err := argon2id.ParseParams(dummyHash); err != nil {
		t.Fatalf("dummy hash must be a parseable argon2id hash: %v", err)
	}
	// It must also not accidentally match a plausible password.
	if argon2id.New(argon2id.DefaultParams()).Verify("hunter2hunter2", dummyHash) {
		t.Fatal("dummy hash must not verify against a real password")
	}
}

func TestLogin_SuspendedAccountIsRejected(t *testing.T) {
	repo := &fakeRepo{
		getUserByEmail: func(context.Context, string) (*domain.User, string, error) {
			return &domain.User{ID: 1, Status: domain.StatusSuspended}, "hash", nil
		},
	}

	_, err := newService(repo, &fakeSessions{}, &countingHasher{accept: true}).
		Login(context.Background(), "reader@example.com", "right", "")

	if !errors.Is(err, domain.ErrUserSuspended) {
		t.Fatalf("error = %v, want ErrUserSuspended", err)
	}
}

func TestLogin_SuccessOpensASession(t *testing.T) {
	sessions := &fakeSessions{}
	repo := &fakeRepo{
		getUserByEmail: func(context.Context, string) (*domain.User, string, error) {
			return &domain.User{ID: 7, Status: domain.StatusActive, Roles: []string{"reader"}}, "hash", nil
		},
	}

	pair, err := newService(repo, sessions, &countingHasher{accept: true}).
		Login(context.Background(), "reader@example.com", "right", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected both tokens, got %+v", pair)
	}
	if len(sessions.created) != 1 {
		t.Fatalf("sessions created = %d, want 1", len(sessions.created))
	}
	if sessions.created[0].FamilyID == "" {
		t.Fatal("a new session must start a rotation family")
	}
	if sessions.created[0].UserID != 7 {
		t.Fatalf("session user = %d, want 7", sessions.created[0].UserID)
	}
}

func TestRegister_RejectsInvalidInputBeforeTouchingTheRepository(t *testing.T) {
	tests := []struct {
		name    string
		reg     domain.Registration
		wantErr error
	}{
		{"short username", domain.Registration{Username: "ab", Email: "a@b.co", Password: "hunter2hunter2"}, domain.ErrInvalidUsername},
		{"illegal username character", domain.Registration{Username: "bad user!", Email: "a@b.co", Password: "hunter2hunter2"}, domain.ErrInvalidUsername},
		{"bad email", domain.Registration{Username: "reader", Email: "not-an-email", Password: "hunter2hunter2"}, domain.ErrInvalidEmail},
		{"short password", domain.Registration{Username: "reader", Email: "a@b.co", Password: "short"}, domain.ErrWeakPassword},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{
				createUser: func(context.Context, domain.User, string) (*domain.User, error) {
					t.Fatal("repository must not be reached for invalid input")
					return nil, nil
				},
			}
			_, err := newService(repo, &fakeSessions{}, &countingHasher{}).
				Register(context.Background(), tc.reg, "")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// A Thai passphrase is 3 bytes per rune; counting bytes would wrongly accept a
// 3-character password.
func TestRegister_PasswordLengthCountsRunesNotBytes(t *testing.T) {
	reg := domain.Registration{
		Username: "reader",
		Email:    "reader@example.com",
		Password: "รหัสผ่าน", // 8 runes, 24 bytes
	}
	if err := reg.Normalize().Validate(); err != nil {
		t.Fatalf("an 8-rune Thai password must be accepted, got %v", err)
	}

	short := reg
	short.Password = "สั้น" // 4 runes, 12 bytes
	if err := short.Normalize().Validate(); !errors.Is(err, domain.ErrWeakPassword) {
		t.Fatalf("a 4-rune password must be rejected, got %v", err)
	}
}

func TestRegister_SeedsDefaultPrefs(t *testing.T) {
	var savedPrefs *domain.Prefs
	repo := &fakeRepo{
		createUser: func(_ context.Context, u domain.User, hash string) (*domain.User, error) {
			if hash == "" {
				t.Fatal("password must be hashed before persisting")
			}
			u.ID = 11
			return &u, nil
		},
		upsertPrefs: func(_ context.Context, p domain.Prefs) (*domain.Prefs, error) {
			savedPrefs = &p
			return &p, nil
		},
	}

	_, err := newService(repo, &fakeSessions{}, &countingHasher{}).Register(context.Background(),
		domain.Registration{Username: "reader", Email: "reader@example.com", Password: "hunter2hunter2"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if savedPrefs == nil {
		t.Fatal("a new account must be seeded with default preferences")
	}
	if *savedPrefs != domain.DefaultPrefs(11) {
		t.Fatalf("prefs = %+v, want the documented defaults", *savedPrefs)
	}
}

func TestRegister_DefaultsDisplayNameToUsername(t *testing.T) {
	var created domain.User
	repo := &fakeRepo{
		createUser: func(_ context.Context, u domain.User, _ string) (*domain.User, error) {
			created = u
			u.ID = 1
			return &u, nil
		},
		upsertPrefs: func(_ context.Context, p domain.Prefs) (*domain.Prefs, error) { return &p, nil },
	}

	_, err := newService(repo, &fakeSessions{}, &countingHasher{}).Register(context.Background(),
		domain.Registration{Username: "reader", Email: "  Reader@Example.COM  ", Password: "hunter2hunter2"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.DisplayName != "reader" {
		t.Fatalf("display name = %q, want the username", created.DisplayName)
	}
	// Email is lower-cased and trimmed so the citext unique index behaves
	// predictably regardless of how the client typed it.
	if created.Email != "reader@example.com" {
		t.Fatalf("email = %q, want it normalised", created.Email)
	}
}

func TestGetPrefs_FallsBackToDefaults(t *testing.T) {
	repo := &fakeRepo{
		getPrefs: func(context.Context, int64) (*domain.Prefs, error) { return nil, nil },
	}

	prefs, err := newService(repo, &fakeSessions{}, &countingHasher{}).GetPrefs(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *prefs != domain.DefaultPrefs(5) {
		t.Fatalf("prefs = %+v, want the documented defaults", *prefs)
	}
}

func TestSetGenrePrefs_DropsDuplicatesAndNormalisesWeights(t *testing.T) {
	var got []domain.GenrePref
	repo := &fakeRepo{
		replaceGenrePrefs: func(_ context.Context, _ int64, prefs []domain.GenrePref) error {
			got = prefs
			return nil
		},
	}

	err := newService(repo, &fakeSessions{}, &countingHasher{}).SetGenrePrefs(context.Background(), 1, []domain.GenrePref{
		{GenreID: 3, Weight: 2},
		{GenreID: 3, Weight: 9}, // duplicate, dropped
		{GenreID: 4, Weight: 0}, // normalised to 1
		{GenreID: 0, Weight: 5}, // invalid, dropped
		{GenreID: -1},           // invalid, dropped
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []domain.GenrePref{{GenreID: 3, Weight: 2}, {GenreID: 4, Weight: 1}}
	if len(got) != len(want) {
		t.Fatalf("prefs = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("prefs = %+v, want %+v", got, want)
		}
	}
}

func TestLogout_MalformedTokenIsANoOp(t *testing.T) {
	sessions := &fakeSessions{}

	// Reporting a parse failure would let a caller probe which tokens exist.
	if err := newService(&fakeRepo{}, sessions, &countingHasher{}).Logout(context.Background(), "!!not-base64!!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessions.revoked != 0 {
		t.Fatal("a malformed token must not reach the session store")
	}
}
