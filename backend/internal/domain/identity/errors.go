package identity

import "errors"

var (
	ErrNotFound           = errors.New("identity: not found")
	ErrEmailTaken         = errors.New("identity: email already registered")
	ErrUsernameTaken      = errors.New("identity: username already taken")
	ErrInvalidUsername    = errors.New("identity: invalid username")
	ErrInvalidEmail       = errors.New("identity: invalid email")
	ErrInvalidDisplayName = errors.New("identity: invalid display name")
	ErrWeakPassword       = errors.New("identity: password too short")
	ErrInvalidPrefs       = errors.New("identity: invalid preferences")

	// ErrInvalidCredentials is returned for both an unknown email and a wrong
	// password, so the response never reveals whether an account exists.
	ErrInvalidCredentials = errors.New("identity: invalid credentials")
	ErrUserSuspended      = errors.New("identity: account is not active")

	ErrInvalidRefreshToken = errors.New("identity: invalid refresh token")
	// ErrTokenReused means an already-revoked refresh token was replayed. The
	// whole family is revoked in response.
	ErrTokenReused = errors.New("identity: refresh token reused")
)
