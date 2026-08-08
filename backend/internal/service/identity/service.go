// Package identity is the application layer for accounts, sessions and prefs.
package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	domain "github.com/mokchan/webnovel-backend/internal/domain/identity"
)

// dummyHash is a genuine argon2id hash of a value nobody submits. Login
// verifies against it when the email is unknown so the "no such user" and
// "wrong password" branches do comparable work and cannot be told apart by
// timing (I-AUTH-02).
//
// It must stay parseable: Verify short-circuits on a malformed hash, which
// would skip the key derivation and defeat the whole point.
const dummyHash = "$argon2id$v=19$m=65536,t=3,p=2$6rtTQfXccBA1fMqezIFZ6w$7VcO83a9aFIMG2ie+OSF4Z439vh7syIL0T85GdhPUls"

// Service orchestrates registration, login, session rotation and preferences.
type Service struct {
	repo       domain.Repository
	sessions   domain.SessionStore
	hasher     domain.PasswordHasher
	issuer     domain.TokenIssuer
	refreshTTL time.Duration
	now        func() time.Time
}

// New wires the service.
func New(
	repo domain.Repository,
	sessions domain.SessionStore,
	hasher domain.PasswordHasher,
	issuer domain.TokenIssuer,
	refreshTTL time.Duration,
	now func() time.Time,
) *Service {
	if now == nil {
		now = time.Now
	}
	if refreshTTL <= 0 {
		refreshTTL = 30 * 24 * time.Hour
	}
	return &Service{
		repo:       repo,
		sessions:   sessions,
		hasher:     hasher,
		issuer:     issuer,
		refreshTTL: refreshTTL,
		now:        now,
	}
}

// Register creates an account and opens its first session.
func (s *Service) Register(ctx context.Context, reg domain.Registration, userAgent string) (*domain.TokenPair, error) {
	reg = reg.Normalize()
	if err := reg.Validate(); err != nil {
		return nil, err
	}

	hash, err := s.hasher.Hash(reg.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, domain.User{
		Username:    reg.Username,
		Email:       reg.Email,
		DisplayName: reg.DisplayName,
		Roles:       domain.DefaultRoles(),
		Status:      domain.StatusActive,
	}, hash)
	if err != nil {
		return nil, err
	}

	// A fresh account starts with default preferences so the reader has a
	// complete settings object on first load.
	if _, err := s.repo.UpsertPrefs(ctx, domain.DefaultPrefs(user.ID)); err != nil {
		return nil, err
	}
	return s.openSession(ctx, *user, userAgent)
}

// Login authenticates by email and password.
func (s *Service) Login(ctx context.Context, email, password, userAgent string) (*domain.TokenPair, error) {
	user, hash, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if errors.Is(err, domain.ErrNotFound) || user == nil {
		// Burn comparable CPU so an unknown email is not measurably faster
		// than a wrong password, then return the same error either way.
		s.hasher.Verify(password, dummyHash)
		return nil, domain.ErrInvalidCredentials
	}
	if !s.hasher.Verify(password, hash) {
		return nil, domain.ErrInvalidCredentials
	}
	if !user.IsActive() {
		return nil, domain.ErrUserSuspended
	}
	return s.openSession(ctx, *user, userAgent)
}

// Refresh rotates a refresh token, returning a new pair.
func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent string) (*domain.TokenPair, error) {
	presented, err := decodeToken(refreshToken)
	if err != nil {
		return nil, domain.ErrInvalidRefreshToken
	}

	now := s.now()
	nextToken, nextHash, err := newRefreshToken()
	if err != nil {
		return nil, err
	}

	session, err := s.sessions.Rotate(ctx, hashToken(presented), domain.Session{
		UserAgent: userAgent,
		ExpiresAt: now.Add(s.refreshTTL),
	}, nextHash, now)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, domain.ErrUserSuspended
	}

	access, expiresAt, err := s.issuer.Issue(user.ID, user.Roles, now)
	if err != nil {
		return nil, err
	}
	return &domain.TokenPair{
		AccessToken:  access,
		RefreshToken: nextToken,
		ExpiresAt:    expiresAt,
		User:         *user,
	}, nil
}

// Logout revokes the presented refresh token's whole family.
//
// The access token stays valid until it expires; killing it instantly would
// require a shared denylist, which this deployment deliberately avoids. The
// client is expected to discard it.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	presented, err := decodeToken(refreshToken)
	if err != nil {
		// Nothing to revoke, and reporting the parse failure would let a caller
		// probe which tokens exist.
		return nil
	}
	return s.sessions.RevokeFamilyByToken(ctx, hashToken(presented), s.now())
}

// Me returns the current account.
func (s *Service) Me(ctx context.Context, userID int64) (*domain.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// UpdateMe patches the mutable profile fields.
func (s *Service) UpdateMe(ctx context.Context, userID int64, patch domain.UserPatch) (*domain.User, error) {
	return s.repo.UpdateUser(ctx, userID, patch)
}

// GetPrefs returns saved preferences, falling back to the defaults when the
// user has never saved any.
func (s *Service) GetPrefs(ctx context.Context, userID int64) (*domain.Prefs, error) {
	prefs, err := s.repo.GetPrefs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		defaults := domain.DefaultPrefs(userID)
		return &defaults, nil
	}
	return prefs, nil
}

// SetPrefs validates and stores reader preferences.
func (s *Service) SetPrefs(ctx context.Context, p domain.Prefs) (*domain.Prefs, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	p.UpdatedAt = s.now()
	return s.repo.UpsertPrefs(ctx, p)
}

// ListGenrePrefs returns the reader's onboarding taste weights.
func (s *Service) ListGenrePrefs(ctx context.Context, userID int64) ([]domain.GenrePref, error) {
	return s.repo.ListGenrePrefs(ctx, userID)
}

// SetGenrePrefs replaces the reader's taste weights wholesale.
func (s *Service) SetGenrePrefs(ctx context.Context, userID int64, prefs []domain.GenrePref) error {
	cleaned := make([]domain.GenrePref, 0, len(prefs))
	seen := make(map[int64]bool, len(prefs))
	for _, p := range prefs {
		if p.GenreID <= 0 || seen[p.GenreID] {
			continue
		}
		seen[p.GenreID] = true
		if p.Weight <= 0 {
			p.Weight = 1
		}
		cleaned = append(cleaned, p)
	}
	return s.repo.ReplaceGenrePrefs(ctx, userID, cleaned)
}

// SweepExpiredSessions deletes refresh tokens that expired before the cutoff.
func (s *Service) SweepExpiredSessions(ctx context.Context, before time.Time) (int64, error) {
	return s.sessions.DeleteExpired(ctx, before)
}

func (s *Service) openSession(ctx context.Context, user domain.User, userAgent string) (*domain.TokenPair, error) {
	now := s.now()

	access, expiresAt, err := s.issuer.Issue(user.ID, user.Roles, now)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshHash, err := newRefreshToken()
	if err != nil {
		return nil, err
	}
	familyID, err := newFamilyID()
	if err != nil {
		return nil, err
	}
	_, err = s.sessions.Create(ctx, domain.Session{
		UserID:    user.ID,
		FamilyID:  familyID,
		UserAgent: userAgent,
		ExpiresAt: now.Add(s.refreshTTL),
	}, refreshHash)
	if err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  access,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         user,
	}, nil
}

// newRefreshToken returns the opaque token handed to the client and the SHA-256
// digest stored in the database.
func newRefreshToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), hashToken(raw), nil
}

func decodeToken(token string) ([]byte, error) {
	if token == "" {
		return nil, domain.ErrInvalidRefreshToken
	}
	return base64.RawURLEncoding.DecodeString(token)
}

func hashToken(raw []byte) []byte {
	sum := sha256.Sum256(raw)
	return sum[:]
}

// newFamilyID returns a random RFC 4122 version-4 UUID string, matching the
// refresh_tokens.family_id column type without pulling in a UUID dependency.
func newFamilyID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	hexed := hex.EncodeToString(buf)
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexed[0:8], hexed[8:12], hexed[12:16], hexed[16:20], hexed[20:32]), nil
}
