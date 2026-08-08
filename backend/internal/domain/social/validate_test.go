package social

import (
	"errors"
	"strings"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/domain/roles"
)

// I-CM-01 — the database CHECK counts characters, so validation must too.
// Thai is three bytes per rune, and a byte-length check would reject a legal
// 1,700-character comment while accepting an illegal 5,000-rune one.
func TestValidateComment_CountsRunesNotBytes(t *testing.T) {
	t.Run("a long Thai comment within the rune limit is accepted", func(t *testing.T) {
		// 4,000 Thai runes = 12,000 bytes: well over any byte-based limit.
		body := strings.Repeat("ก", 4000)
		if len(body) <= MaxCommentRunes {
			t.Fatalf("test setup is wrong: %d bytes should exceed the limit", len(body))
		}
		if err := ValidateComment(body, 0); err != nil {
			t.Fatalf("a 4000-rune Thai comment must be accepted, got %v", err)
		}
	})

	t.Run("exactly at the limit is accepted", func(t *testing.T) {
		if err := ValidateComment(strings.Repeat("ก", MaxCommentRunes), 0); err != nil {
			t.Fatalf("a comment of exactly %d runes must be accepted, got %v", MaxCommentRunes, err)
		}
	})

	t.Run("one rune over the limit is rejected", func(t *testing.T) {
		err := ValidateComment(strings.Repeat("ก", MaxCommentRunes+1), 0)
		if !errors.Is(err, ErrCommentTooLong) {
			t.Fatalf("error = %v, want ErrCommentTooLong", err)
		}
	})

	t.Run("an ASCII comment over the limit is rejected too", func(t *testing.T) {
		err := ValidateComment(strings.Repeat("a", MaxCommentRunes+1), 0)
		if !errors.Is(err, ErrCommentTooLong) {
			t.Fatalf("error = %v, want ErrCommentTooLong", err)
		}
	})
}

func TestValidateComment_RejectsEmptyBodies(t *testing.T) {
	for _, body := range []string{"", "   ", "\t\n", "  "} {
		if err := ValidateComment(body, 0); !errors.Is(err, ErrCommentEmpty) {
			t.Fatalf("body %q: error = %v, want ErrCommentEmpty", body, err)
		}
	}
}

// Replies are one level deep; the schema allows arbitrary nesting, so the rule
// is enforced here.
func TestValidateComment_RejectsNestedReplies(t *testing.T) {
	if err := ValidateComment("ตอบกลับ", 0); err != nil {
		t.Fatalf("a top-level comment must be accepted, got %v", err)
	}
	if err := ValidateComment("ตอบกลับ", 1); !errors.Is(err, ErrReplyTooDeep) {
		t.Fatalf("error = %v, want ErrReplyTooDeep", err)
	}
}

func TestValidateRating(t *testing.T) {
	for rating := MinRating; rating <= MaxRating; rating++ {
		if err := ValidateRating(rating); err != nil {
			t.Fatalf("rating %d must be accepted, got %v", rating, err)
		}
	}
	for _, rating := range []int{0, -1, 6, 100} {
		if err := ValidateRating(rating); !errors.Is(err, ErrInvalidRating) {
			t.Fatalf("rating %d: error = %v, want ErrInvalidRating", rating, err)
		}
	}
}

func TestNormalizeSort(t *testing.T) {
	tests := map[string]string{
		"latest":       SortLatest,
		"with_replies": SortWithReplies,
		"popular":      SortPopular,
		"":             SortPopular,
		"trending":     SortPopular,
	}
	for in, want := range tests {
		if got := NormalizeSort(in); got != want {
			t.Fatalf("NormalizeSort(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCanDelete(t *testing.T) {
	const (
		authorID     = int64(10)
		translatorID = int64(20)
		strangerID   = int64(30)
	)
	comment := Comment{UserID: authorID}
	translator := translatorID

	tests := []struct {
		name         string
		viewer       Viewer
		chapterOwner *int64
		want         bool
	}{
		{"the author", Viewer{UserID: authorID}, &translator, true},
		{"the chapter's translator", Viewer{UserID: translatorID, Roles: []string{roles.Translator}}, &translator, true},
		{"an administrator", Viewer{UserID: strangerID, Roles: []string{roles.Admin}}, &translator, true},
		{"an unrelated reader", Viewer{UserID: strangerID, Roles: []string{roles.Reader}}, &translator, false},
		{"a translator of some other novel", Viewer{UserID: strangerID, Roles: []string{roles.Translator}}, &translator, false},
		{"anonymous", Viewer{}, &translator, false},
		{"a chapter with no translator", Viewer{UserID: strangerID}, nil, false},
		// Guards against an anonymous viewer (id 0) matching a zero owner id.
		{"anonymous never matches a zero translator id", Viewer{}, new(int64), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanDelete(comment, tc.viewer, tc.chapterOwner); got != tc.want {
				t.Fatalf("CanDelete = %v, want %v", got, tc.want)
			}
		})
	}
}
