package reading

import (
	"testing"

	"github.com/mokchan/webnovel-backend/internal/domain/roles"
)

// Decide is the single entitlement rule in the system; every branch is
// exercised here without touching a database.
func TestDecide(t *testing.T) {
	reader := Viewer{UserID: 7, Roles: []string{roles.Reader}}
	admin := Viewer{UserID: 9, Roles: []string{roles.Reader, roles.Admin}}
	anonymous := Viewer{}

	tests := []struct {
		name         string
		priceCoins   int
		viewer       Viewer
		unlocked     bool
		isTranslator bool
		wantLocked   bool
	}{
		{"free chapter is readable anonymously", 0, anonymous, false, false, false},
		{"free chapter is readable by a reader", 0, reader, false, false, false},
		{"paid chapter is locked for anonymous", 5, anonymous, false, false, true},
		{"paid chapter is locked for a reader who has not unlocked it", 5, reader, false, false, true},
		{"paid chapter is readable once unlocked", 5, reader, true, false, false},
		{"paid chapter is readable by its own translator", 5, reader, false, true, false},
		{"paid chapter is readable by an admin", 5, admin, false, false, false},
		{"a negative price is treated as free", -1, anonymous, false, false, false},
		{"an unlock flag cannot rescue an anonymous viewer", 5, anonymous, true, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(tc.priceCoins, tc.viewer, tc.unlocked, tc.isTranslator)
			if got != tc.wantLocked {
				t.Fatalf("Decide(price=%d) = %v, want %v", tc.priceCoins, got, tc.wantLocked)
			}
		})
	}
}

func TestIsTranslatorOf(t *testing.T) {
	translatorID := int64(42)

	tests := []struct {
		name    string
		chapter Chapter
		viewer  Viewer
		want    bool
	}{
		{
			name:    "matching translator",
			chapter: Chapter{TranslatorID: &translatorID},
			viewer:  Viewer{UserID: 42},
			want:    true,
		},
		{
			name:    "different user",
			chapter: Chapter{TranslatorID: &translatorID},
			viewer:  Viewer{UserID: 43},
			want:    false,
		},
		{
			name:    "chapter has no translator",
			chapter: Chapter{},
			viewer:  Viewer{UserID: 42},
			want:    false,
		},
		{
			// Guards against an anonymous viewer (UserID 0) matching a chapter
			// whose translator id somehow read back as 0.
			name:    "anonymous viewer never matches",
			chapter: Chapter{TranslatorID: new(int64)},
			viewer:  Viewer{},
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTranslatorOf(tc.chapter, tc.viewer); got != tc.want {
				t.Fatalf("IsTranslatorOf = %v, want %v", got, tc.want)
			}
		})
	}
}
