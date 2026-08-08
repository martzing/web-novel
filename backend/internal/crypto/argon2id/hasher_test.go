package argon2id

import (
	"bytes"
	"strings"
	"testing"
)

// U-AUTH-01 — password hashing uses argon2id with the configured parameters.
func TestHash_UsesConfiguredArgon2idParams(t *testing.T) {
	// Small cost keeps the test fast; the assertion is that whatever is
	// configured is what ends up encoded in the hash.
	params := Params{Memory: 8 * 1024, Time: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32}
	hasher := New(params)

	encoded, err := hasher.Hash("ปลาดาบเก้าสายธาร")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=8192,t=1,p=1$") {
		t.Fatalf("hash does not carry the configured argon2id params: %q", encoded)
	}

	got, err := ParseParams(encoded)
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	if got.Memory != params.Memory || got.Time != params.Time || got.Parallelism != params.Parallelism {
		t.Fatalf("parsed params = %+v, want memory/time/parallelism from %+v", got, params)
	}
	if got.SaltLen != params.SaltLen || got.KeyLen != params.KeyLen {
		t.Fatalf("parsed lengths = salt %d key %d, want salt %d key %d",
			got.SaltLen, got.KeyLen, params.SaltLen, params.KeyLen)
	}
}

func TestHash_SaltsEachHashIndependently(t *testing.T) {
	hasher := New(Params{Memory: 8 * 1024, Time: 1, Parallelism: 1})

	first, err := hasher.Hash("same-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := hasher.Hash("same-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first == second {
		t.Fatal("two hashes of the same password must differ; the salt is not random")
	}
	if !hasher.Verify("same-password", first) || !hasher.Verify("same-password", second) {
		t.Fatal("both independently salted hashes must verify")
	}
}

func TestVerify(t *testing.T) {
	hasher := New(Params{Memory: 8 * 1024, Time: 1, Parallelism: 1})
	encoded, err := hasher.Hash("correct horse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{"correct password", "correct horse", encoded, true},
		{"wrong password", "battery staple", encoded, false},
		{"empty password", "", encoded, false},
		// The seed file historically carried a bcrypt-shaped placeholder.
		// Verify must report a mismatch, not an error, so a login against such
		// a row is a 401 rather than a 500.
		{"bcrypt-shaped placeholder", "anything", "$2a$10$placeholderplaceholderplaceholder", false},
		{"empty hash", "anything", "", false},
		{"garbage hash", "anything", "not-a-hash", false},
		{"truncated hash", "anything", "$argon2id$v=19$m=8192,t=1,p=1$", false},
		{"wrong algorithm", "anything", "$argon2i$v=19$m=8192,t=1,p=1$c2FsdA$aGFzaA", false},
		{"unsupported version", "anything", "$argon2id$v=16$m=8192,t=1,p=1$c2FsdA$aGFzaA", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasher.Verify(tc.password, tc.hash); got != tc.want {
				t.Fatalf("Verify = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNew_FillsZeroValuedParamsWithDefaults(t *testing.T) {
	got := New(Params{}).Params()
	want := DefaultParams()
	if got != want {
		t.Fatalf("params = %+v, want %+v", got, want)
	}
}

func TestWithRandom_MakesHashDeterministic(t *testing.T) {
	hasher := New(Params{Memory: 8 * 1024, Time: 1, Parallelism: 1})

	fixed := func() *Hasher {
		return hasher.WithRandom(bytes.NewReader(bytes.Repeat([]byte{0xAB}, 64)))
	}

	first, err := fixed().Hash("deterministic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := fixed().Hash("deterministic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first != second {
		t.Fatalf("a fixed salt source must produce a stable hash:\n%s\n%s", first, second)
	}
}
