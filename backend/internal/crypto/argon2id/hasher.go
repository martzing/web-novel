// Package argon2id hashes and verifies passwords with argon2id, encoded in the
// standard PHC string format.
package argon2id

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

// ErrInvalidHash is returned by ParseParams when the encoded string is not a
// well-formed argon2id PHC hash.
var ErrInvalidHash = errors.New("argon2id: invalid encoded hash")

// Params are the argon2id cost parameters.
type Params struct {
	Memory      uint32 // KiB
	Time        uint32 // iterations
	Parallelism uint8
	SaltLen     uint32
	KeyLen      uint32
}

// DefaultParams are the OWASP-recommended baseline: 64 MiB, 3 passes, 2 lanes.
func DefaultParams() Params {
	return Params{Memory: 64 * 1024, Time: 3, Parallelism: 2, SaltLen: 16, KeyLen: 32}
}

// Hasher produces and verifies argon2id password hashes.
type Hasher struct {
	params Params
	rand   io.Reader
}

// New builds a hasher. Zero-valued cost fields fall back to DefaultParams.
func New(p Params) *Hasher {
	def := DefaultParams()
	if p.Memory == 0 {
		p.Memory = def.Memory
	}
	if p.Time == 0 {
		p.Time = def.Time
	}
	if p.Parallelism == 0 {
		p.Parallelism = def.Parallelism
	}
	if p.SaltLen == 0 {
		p.SaltLen = def.SaltLen
	}
	if p.KeyLen == 0 {
		p.KeyLen = def.KeyLen
	}
	return &Hasher{params: p, rand: rand.Reader}
}

// WithRandom swaps the salt source. Tests use it to make output deterministic.
func (h *Hasher) WithRandom(r io.Reader) *Hasher {
	clone := *h
	clone.rand = r
	return &clone
}

// Params returns the configured cost parameters.
func (h *Hasher) Params() Params { return h.params }

// Hash derives a PHC-encoded argon2id hash of plain.
func (h *Hasher) Hash(plain string) (string, error) {
	salt := make([]byte, h.params.SaltLen)
	if _, err := io.ReadFull(h.rand, salt); err != nil {
		return "", fmt.Errorf("argon2id: read salt: %w", err)
	}
	key := argon2.IDKey([]byte(plain), salt, h.params.Time, h.params.Memory, h.params.Parallelism, h.params.KeyLen)
	return encode(h.params, salt, key), nil
}

// Verify reports whether plain matches the encoded hash.
//
// It returns false — never an error — for an unparseable hash. The seed data
// historically carried a bcrypt-shaped placeholder, and treating that as a
// failure rather than an error is what keeps a login attempt a 401 instead of
// a 500.
func (h *Hasher) Verify(plain, encoded string) bool {
	p, salt, want, err := decode(encoded)
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(plain), salt, p.Time, p.Memory, p.Parallelism, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// ParseParams extracts the cost parameters recorded in an encoded hash.
func ParseParams(encoded string) (Params, error) {
	p, _, _, err := decode(encoded)
	return p, err
}

func encode(p Params, salt, key []byte) string {
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Time, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}

func decode(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return Params{}, nil, nil, ErrInvalidHash
	}

	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &p.Parallelism); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	p.SaltLen = uint32(len(salt))
	p.KeyLen = uint32(len(key))
	return p, salt, key, nil
}
