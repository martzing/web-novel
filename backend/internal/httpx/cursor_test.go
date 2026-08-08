package httpx

import (
	"errors"
	"reflect"
	"testing"
)

func TestEncodeDecodeCursor(t *testing.T) {
	tests := []struct {
		name   string
		cursor Cursor
	}{
		{"single key", Cursor{Sort: "popular", Keys: []string{"1832"}}},
		{"composite key", Cursor{Sort: "latest", Keys: []string{"2026-01-02T03:04:05Z", "1832"}}},
		{"thai text in a key", Cursor{Sort: "latest", Keys: []string{"เซียนดาบ", "7"}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded := EncodeCursor(tc.cursor)
			if encoded == "" {
				t.Fatal("expected a non-empty encoding")
			}
			// base64url without padding keeps the cursor URL-safe.
			for _, r := range encoded {
				if r == '+' || r == '/' || r == '=' {
					t.Fatalf("cursor %q is not URL-safe", encoded)
				}
			}

			got, err := DecodeCursor(encoded)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.cursor) {
				t.Fatalf("round trip = %+v, want %+v", got, tc.cursor)
			}
		})
	}
}

func TestDecodeCursor_EmptyMeansFirstPage(t *testing.T) {
	got, err := DecodeCursor("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, Cursor{}) {
		t.Fatalf("expected the zero cursor, got %+v", got)
	}
}

func TestDecodeCursor_Garbage(t *testing.T) {
	for _, in := range []string{"!!!not-base64!!!", "YWJj", "e30", "bnVsbA"} {
		if _, err := DecodeCursor(in); err == nil {
			t.Fatalf("expected an error for %q", in)
		} else if !errors.Is(err, ErrBadCursor) {
			t.Fatalf("expected ErrBadCursor for %q, got %v", in, err)
		}
	}
}

// A cursor minted under one ordering must not be silently replayed against
// another, which would interleave two different sorts.
func TestDecodeCursorFor_RejectsSortMismatch(t *testing.T) {
	encoded := EncodeCursor(Cursor{Sort: "popular", Keys: []string{"10", "5"}})

	if _, err := DecodeCursorFor(encoded, "latest"); !errors.Is(err, ErrBadCursor) {
		t.Fatalf("expected ErrBadCursor on sort mismatch, got %v", err)
	}
	if _, err := DecodeCursorFor(encoded, "popular"); err != nil {
		t.Fatalf("matching sort must decode, got %v", err)
	}
	if _, err := DecodeCursorFor("", "latest"); err != nil {
		t.Fatalf("an empty cursor is valid for any sort, got %v", err)
	}
}
