package reading

import (
	"testing"
	"time"

	"github.com/mokchan/webnovel-backend/internal/domain/roles"
)

var visNow = time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

func visAt(offset time.Duration) *time.Time {
	t := visNow.Add(offset)
	return &t
}

func TestSee(t *testing.T) {
	anon := Viewer{}
	reader := Viewer{UserID: 7, Roles: []string{roles.Reader}}
	admin := Viewer{UserID: 9, Roles: []string{roles.Reader, roles.Admin}}

	published := Availability{Status: StatusPublished, PublishedAt: visAt(-time.Hour)}
	early := Availability{
		Status:      StatusPublished,
		PublishedAt: visAt(-time.Hour),
		PublicAt:    visAt(23 * time.Hour), // still inside a 24h window
	}
	draft := Availability{Status: StatusDraft}
	scheduled := Availability{Status: StatusScheduled}

	tests := []struct {
		name  string
		avail Availability
		acc   Access
		want  Visibility
	}{
		{"a draft is hidden from readers", draft, Access{Viewer: reader}, VisibleHidden},
		{"a scheduled chapter is hidden", scheduled, Access{Viewer: reader}, VisibleHidden},
		{"a draft is hidden from anonymous", draft, Access{}, VisibleHidden},
		{"its own translator previews the draft", draft, Access{Viewer: reader, IsTranslator: true}, VisibleFull},
		{"an admin previews the draft", draft, Access{Viewer: admin}, VisibleFull},

		{"a public chapter is readable anonymously", published, Access{Viewer: anon}, VisibleFull},
		{"a public chapter is readable by a reader", published, Access{Viewer: reader}, VisibleFull},

		{"inside the early window, anonymous sees a teaser", early, Access{Viewer: anon}, VisibleTeaser},
		{"inside the early window, a plain reader sees a teaser", early, Access{Viewer: reader}, VisibleTeaser},
		{"a subscriber reads it early", early, Access{Viewer: reader, Subscribed: true}, VisibleFull},
		// Ownership must win over subscription, or cancelling would revoke
		// chapters the reader has already paid for.
		{"an owner who unsubscribed keeps access", early, Access{Viewer: reader, Owns: true}, VisibleFull},
		{"its translator reads it early", early, Access{Viewer: reader, IsTranslator: true}, VisibleFull},
		{"an admin reads it early", early, Access{Viewer: admin}, VisibleFull},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := See(tc.avail, tc.acc, visNow); got != tc.want {
				t.Fatalf("See = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSee_PublicAtBoundary(t *testing.T) {
	// Exactly at the deadline the chapter is public: the window has elapsed.
	exactly := Availability{Status: StatusPublished, PublicAt: &visNow}
	if got := See(exactly, Access{}, visNow); got != VisibleFull {
		t.Fatalf("at the deadline See = %v, want full", got)
	}

	oneSecondEarly := Availability{Status: StatusPublished, PublicAt: visAt(time.Second)}
	if got := See(oneSecondEarly, Access{}, visNow); got != VisibleTeaser {
		t.Fatalf("one second before the deadline See = %v, want teaser", got)
	}
}

func TestSee_NilPublicAtMeansImmediately(t *testing.T) {
	avail := Availability{Status: StatusPublished, PublishedAt: visAt(-time.Minute)}
	if got := See(avail, Access{}, visNow); got != VisibleFull {
		t.Fatalf("See = %v, want full when there is no early-access window", got)
	}
}

func TestVisibility_String(t *testing.T) {
	tests := map[Visibility]string{
		VisibleHidden: "hidden",
		VisibleTeaser: "early_access",
		VisibleFull:   "public",
	}
	for v, want := range tests {
		if got := v.String(); got != want {
			t.Fatalf("%d.String() = %q, want %q", v, got, want)
		}
	}
}
