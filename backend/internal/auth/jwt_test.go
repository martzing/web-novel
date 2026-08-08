package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

const testSecret = "test-secret-at-least-16-chars"

var testNow = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

func TestNewIssuer_RejectsWeakSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"empty secret", "", true},
		{"short secret", "too-short", true},
		{"exactly the minimum", strings.Repeat("x", MinSecretLen), false},
		{"long secret", testSecret, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewIssuer(tc.secret, time.Minute)
			if tc.wantErr && !errors.Is(err, ErrWeakSecret) {
				t.Fatalf("expected ErrWeakSecret, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestIssuer_RoundTrip(t *testing.T) {
	issuer, err := NewIssuer(testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	roles := []string{"reader", "translator"}
	token, expiresAt, err := issuer.Issue(1832, roles, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := testNow.Add(15 * time.Minute); !expiresAt.Equal(want) {
		t.Fatalf("expiresAt = %v, want %v", expiresAt, want)
	}

	claims, err := issuer.Parse(token, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != 1832 {
		t.Fatalf("UserID = %d, want 1832", claims.UserID)
	}
	if !reflect.DeepEqual(claims.Roles, roles) {
		t.Fatalf("Roles = %v, want %v", claims.Roles, roles)
	}
	if claims.TokenID == "" {
		t.Fatal("expected a jti to be set")
	}
}

func TestIssue_EmitsDistinctTokenIDs(t *testing.T) {
	issuer, _ := NewIssuer(testSecret, time.Minute)

	first, _, _ := issuer.Issue(1, nil, testNow)
	second, _, _ := issuer.Issue(1, nil, testNow)
	if first == second {
		t.Fatal("two tokens issued at the same instant must differ by jti")
	}
}

func TestParse_RejectsExpiredToken(t *testing.T) {
	issuer, _ := NewIssuer(testSecret, time.Minute)
	token, _, _ := issuer.Issue(5, nil, testNow)

	if _, err := issuer.Parse(token, testNow.Add(90*time.Second)); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for an expired token, got %v", err)
	}
	if _, err := issuer.Parse(token, testNow.Add(30*time.Second)); err != nil {
		t.Fatalf("token inside its lifetime must parse, got %v", err)
	}
}

func TestParse_RejectsTamperedSignature(t *testing.T) {
	issuer, _ := NewIssuer(testSecret, time.Minute)
	token, _, _ := issuer.Issue(5, nil, testNow)

	tampered := token[:len(token)-2] + func() string {
		if strings.HasSuffix(token, "A") {
			return "BB"
		}
		return "AA"
	}()

	if _, err := issuer.Parse(tampered, testNow); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for a tampered signature, got %v", err)
	}
}

func TestParse_RejectsTokenSignedWithAnotherSecret(t *testing.T) {
	mine, _ := NewIssuer(testSecret, time.Minute)
	theirs, _ := NewIssuer("a-completely-different-secret", time.Minute)

	token, _, _ := theirs.Issue(5, nil, testNow)
	if _, err := mine.Parse(token, testNow); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for a foreign signature, got %v", err)
	}
}

// The accepted algorithm is pinned to HS256, so "alg":"none" is rejected before
// any signature verification happens.
func TestParse_RejectsAlgNone(t *testing.T) {
	issuer, _ := NewIssuer(testSecret, time.Minute)

	enc := func(v any) string {
		raw, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	header := enc(map[string]any{"alg": "none", "typ": "JWT"})
	payload := enc(map[string]any{
		"sub": "1",
		"exp": testNow.Add(time.Hour).Unix(),
		"iat": testNow.Unix(),
	})

	for _, token := range []string{
		header + "." + payload + ".",
		header + "." + payload,
	} {
		if _, err := issuer.Parse(token, testNow); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("expected ErrInvalidToken for an alg=none token, got %v", err)
		}
	}
}

// A token whose header claims HS512 must not be accepted even when the caller
// signs it correctly with our secret.
func TestParse_RejectsUnexpectedAlgorithm(t *testing.T) {
	issuer, _ := NewIssuer(testSecret, time.Minute)

	enc := func(v any) string {
		raw, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	header := enc(map[string]any{"alg": "HS512", "typ": "JWT"})
	payload := enc(map[string]any{"sub": "1", "exp": testNow.Add(time.Hour).Unix()})

	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write([]byte(header + "." + payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if _, err := issuer.Parse(header+"."+payload+"."+signature, testNow); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for an HS512 header, got %v", err)
	}
}

func TestParse_RejectsMalformedInput(t *testing.T) {
	issuer, _ := NewIssuer(testSecret, time.Minute)

	for _, token := range []string{"", "not-a-token", "a.b.c", "....", "Bearer x.y.z"} {
		if _, err := issuer.Parse(token, testNow); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("expected ErrInvalidToken for %q, got %v", token, err)
		}
	}
}
