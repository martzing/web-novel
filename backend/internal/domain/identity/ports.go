package identity

import (
	"context"
	"time"
)

// Repository is the driven port for accounts and preferences.
type Repository interface {
	// CreateUser persists a new account. Duplicate email/username are detected
	// through the unique indexes and surface as ErrEmailTaken/ErrUsernameTaken,
	// never through a racy pre-flight SELECT.
	CreateUser(ctx context.Context, u User, passwordHash string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	// GetUserByEmail returns the account and its stored password hash.
	GetUserByEmail(ctx context.Context, email string) (*User, string, error)
	UpdateUser(ctx context.Context, id int64, patch UserPatch) (*User, error)

	GetPrefs(ctx context.Context, userID int64) (*Prefs, error)
	UpsertPrefs(ctx context.Context, p Prefs) (*Prefs, error)

	ListGenrePrefs(ctx context.Context, userID int64) ([]GenrePref, error)
	ReplaceGenrePrefs(ctx context.Context, userID int64, prefs []GenrePref) error
}

// SessionStore persists refresh-token sessions. Only token hashes are stored.
type SessionStore interface {
	Create(ctx context.Context, s Session, tokenHash []byte) (*Session, error)

	// Rotate atomically consumes tokenHash and issues its successor. Presenting
	// an already-revoked token means theft: the whole family is revoked and
	// ErrTokenReused is returned.
	Rotate(ctx context.Context, tokenHash []byte, next Session, nextHash []byte, now time.Time) (*Session, error)

	RevokeFamilyByToken(ctx context.Context, tokenHash []byte, now time.Time) error
	RevokeAllForUser(ctx context.Context, userID int64, now time.Time) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(plain string) (string, error)
	// Verify reports a match. It returns false rather than an error for an
	// unparseable stored hash, so a legacy row yields 401 instead of 500.
	Verify(plain, encoded string) bool
}

// TokenIssuer mints short-lived access tokens.
type TokenIssuer interface {
	Issue(userID int64, roles []string, now time.Time) (token string, expiresAt time.Time, err error)
}
