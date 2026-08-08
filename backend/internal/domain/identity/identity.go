// Package identity is the domain layer for accounts, sessions and reader
// preferences. It is framework-free.
package identity

import (
	"net/mail"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mokchan/webnovel-backend/internal/domain/roles"
)

// User is an account.
type User struct {
	ID          int64
	Username    string
	Email       string
	DisplayName string
	AvatarURL   string
	Roles       []string
	Status      string
	CreatedAt   time.Time
}

// Account statuses.
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusDeleted   = "deleted"
)

// HasRole reports whether the user carries the given role.
func (u User) HasRole(role string) bool { return slices.Contains(u.Roles, role) }

// IsActive reports whether the account may sign in.
func (u User) IsActive() bool { return u.Status == StatusActive }

// UserPatch carries the mutable profile fields; nil means "leave unchanged".
type UserPatch struct {
	DisplayName *string
	AvatarURL   *string
}

// Registration is a signup request.
type Registration struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
}

// MinPasswordLen is the shortest password accepted at signup.
const MinPasswordLen = 8

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,32}$`)

// Normalize trims surrounding whitespace and defaults the display name.
func (r Registration) Normalize() Registration {
	r.Username = strings.TrimSpace(r.Username)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	if r.DisplayName == "" {
		r.DisplayName = r.Username
	}
	return r
}

// Validate checks a signup request without touching the database.
func (r Registration) Validate() error {
	if !usernamePattern.MatchString(r.Username) {
		return ErrInvalidUsername
	}
	if _, err := mail.ParseAddress(r.Email); err != nil {
		return ErrInvalidEmail
	}
	// Count runes, not bytes: a Thai passphrase is 3 bytes per character and a
	// byte-length check would reject legitimate short passphrases.
	if utf8.RuneCountInString(r.Password) < MinPasswordLen {
		return ErrWeakPassword
	}
	if utf8.RuneCountInString(r.DisplayName) > 64 {
		return ErrInvalidDisplayName
	}
	return nil
}

// Prefs are the reader settings synced across devices.
type Prefs struct {
	UserID      int64
	Theme       string
	Font        string
	FontSize    int
	LineHeight  float64
	ColumnWidth string
	UpdatedAt   time.Time
}

// Allowed preference values, mirroring the CHECK constraints in 0001_init.sql.
var (
	Themes       = []string{"light", "sepia", "dark"}
	Fonts        = []string{"loop", "serif", "sans"}
	ColumnWidths = []string{"narrow", "normal", "wide"}
)

// Preference bounds, mirroring the CHECK constraints in 0001_init.sql.
const (
	MinFontSize   = 14
	MaxFontSize   = 28
	MinLineHeight = 1.4
	MaxLineHeight = 2.4
)

// DefaultPrefs matches the column defaults so a user with no row still gets a
// complete, valid settings object.
func DefaultPrefs(userID int64) Prefs {
	return Prefs{
		UserID:      userID,
		Theme:       "light",
		Font:        "loop",
		FontSize:    20,
		LineHeight:  2.0,
		ColumnWidth: "normal",
	}
}

// Validate rejects preferences the database would refuse, so the caller gets a
// 400 with a useful message instead of a constraint violation.
func (p Prefs) Validate() error {
	if !slices.Contains(Themes, p.Theme) {
		return ErrInvalidPrefs
	}
	if !slices.Contains(Fonts, p.Font) {
		return ErrInvalidPrefs
	}
	if !slices.Contains(ColumnWidths, p.ColumnWidth) {
		return ErrInvalidPrefs
	}
	if p.FontSize < MinFontSize || p.FontSize > MaxFontSize {
		return ErrInvalidPrefs
	}
	if p.LineHeight < MinLineHeight || p.LineHeight > MaxLineHeight {
		return ErrInvalidPrefs
	}
	return nil
}

// GenrePref is one onboarding taste weight.
type GenrePref struct {
	GenreID int64
	Weight  int
}

// Session is a refresh-token session. Tokens are stored hashed; FamilyID ties a
// rotation chain together.
type Session struct {
	ID        int64
	UserID    int64
	FamilyID  string
	UserAgent string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// TokenPair is what the auth endpoints hand back.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	User         User
}

// DefaultRoles are granted to a brand-new account.
func DefaultRoles() []string { return []string{roles.Reader} }
