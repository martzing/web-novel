// Package auth issues and parses the HS256 access tokens used on protected
// routes.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Errors returned by Parse.
var (
	ErrInvalidToken = errors.New("auth: invalid token")
	ErrWeakSecret   = errors.New("auth: jwt secret must be at least 16 characters")
)

// MinSecretLen keeps a throwaway development secret from silently reaching
// production.
const MinSecretLen = 16

// Claims is the access-token payload.
type Claims struct {
	UserID    int64
	Roles     []string
	IssuedAt  time.Time
	ExpiresAt time.Time
	TokenID   string
}

// Issuer signs and verifies access tokens.
type Issuer struct {
	secret []byte
	ttl    time.Duration
}

// NewIssuer builds an issuer, rejecting an empty or short secret.
func NewIssuer(secret string, ttl time.Duration) (*Issuer, error) {
	if len(secret) < MinSecretLen {
		return nil, ErrWeakSecret
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &Issuer{secret: []byte(secret), ttl: ttl}, nil
}

// TTL is the access-token lifetime.
func (i *Issuer) TTL() time.Duration { return i.ttl }

// Issue mints a signed access token for the user.
func (i *Issuer) Issue(userID int64, roles []string, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(i.ttl)
	tokenID, err := randomID()
	if err != nil {
		return "", time.Time{}, err
	}
	if roles == nil {
		roles = []string{}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   strconv.FormatInt(userID, 10),
		"roles": roles,
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
		"jti":   tokenID,
	})
	signed, err := token.SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign token: %w", err)
	}
	return signed, expiresAt, nil
}

// Parse verifies a token's signature and expiry and returns its claims.
//
// The accepted algorithm is pinned to HS256, so "alg": "none" and asymmetric
// algorithm-confusion attacks are rejected before any signature check runs.
func (i *Issuer) Parse(token string, now time.Time) (*Claims, error) {
	parsed, err := jwt.Parse(token,
		func(*jwt.Token) (any, error) { return i.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	raw, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	sub, err := raw.GetSubject()
	if err != nil {
		return nil, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil || userID <= 0 {
		return nil, ErrInvalidToken
	}

	out := &Claims{UserID: userID, Roles: []string{}}
	if issued, err := raw.GetIssuedAt(); err == nil && issued != nil {
		out.IssuedAt = issued.Time
	}
	if exp, err := raw.GetExpirationTime(); err == nil && exp != nil {
		out.ExpiresAt = exp.Time
	}
	if jti, ok := raw["jti"].(string); ok {
		out.TokenID = jti
	}
	if roles, ok := raw["roles"].([]any); ok {
		for _, r := range roles {
			if s, ok := r.(string); ok {
				out.Roles = append(out.Roles, s)
			}
		}
	}
	return out, nil
}

func randomID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
