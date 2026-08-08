package identity

import (
	"errors"
	"testing"
)

// Prefs.Validate mirrors the CHECK constraints in 0001_init.sql so an invalid
// value becomes a 400 with a useful message rather than a constraint violation.
func TestPrefs_Validate(t *testing.T) {
	valid := DefaultPrefs(1)

	tests := []struct {
		name    string
		mutate  func(*Prefs)
		wantErr bool
	}{
		{"defaults are valid", func(*Prefs) {}, false},
		{"sepia theme", func(p *Prefs) { p.Theme = "sepia" }, false},
		{"dark theme", func(p *Prefs) { p.Theme = "dark" }, false},
		{"unknown theme", func(p *Prefs) { p.Theme = "midnight" }, true},
		{"empty theme", func(p *Prefs) { p.Theme = "" }, true},

		{"serif font", func(p *Prefs) { p.Font = "serif" }, false},
		{"sans font", func(p *Prefs) { p.Font = "sans" }, false},
		{"unknown font", func(p *Prefs) { p.Font = "comic" }, true},

		{"narrow column", func(p *Prefs) { p.ColumnWidth = "narrow" }, false},
		{"wide column", func(p *Prefs) { p.ColumnWidth = "wide" }, false},
		{"unknown column width", func(p *Prefs) { p.ColumnWidth = "ultra" }, true},

		{"minimum font size", func(p *Prefs) { p.FontSize = MinFontSize }, false},
		{"maximum font size", func(p *Prefs) { p.FontSize = MaxFontSize }, false},
		{"font size below range", func(p *Prefs) { p.FontSize = MinFontSize - 1 }, true},
		{"font size above range", func(p *Prefs) { p.FontSize = MaxFontSize + 1 }, true},
		{"zero font size", func(p *Prefs) { p.FontSize = 0 }, true},

		{"minimum line height", func(p *Prefs) { p.LineHeight = MinLineHeight }, false},
		{"maximum line height", func(p *Prefs) { p.LineHeight = MaxLineHeight }, false},
		{"line height below range", func(p *Prefs) { p.LineHeight = 1.3 }, true},
		{"line height above range", func(p *Prefs) { p.LineHeight = 2.5 }, true},
		{"zero line height", func(p *Prefs) { p.LineHeight = 0 }, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := valid
			tc.mutate(&p)

			err := p.Validate()
			if tc.wantErr && !errors.Is(err, ErrInvalidPrefs) {
				t.Fatalf("expected ErrInvalidPrefs, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDefaultPrefs_MatchesColumnDefaults(t *testing.T) {
	got := DefaultPrefs(42)
	want := Prefs{
		UserID:      42,
		Theme:       "light",
		Font:        "loop",
		FontSize:    20,
		LineHeight:  2.0,
		ColumnWidth: "normal",
	}
	if got != want {
		t.Fatalf("DefaultPrefs = %+v, want %+v", got, want)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("the defaults must themselves be valid: %v", err)
	}
}

func TestRegistration_Normalize(t *testing.T) {
	tests := []struct {
		name            string
		in              Registration
		wantEmail       string
		wantDisplayName string
	}{
		{
			name:            "trims and lower-cases the email",
			in:              Registration{Username: "reader", Email: "  Reader@Example.COM "},
			wantEmail:       "reader@example.com",
			wantDisplayName: "reader",
		},
		{
			name:            "keeps an explicit display name",
			in:              Registration{Username: "reader", Email: "a@b.co", DisplayName: "  หมอกจันทร์  "},
			wantEmail:       "a@b.co",
			wantDisplayName: "หมอกจันทร์",
		},
		{
			name:            "falls back to the username",
			in:              Registration{Username: "reader", Email: "a@b.co", DisplayName: "   "},
			wantEmail:       "a@b.co",
			wantDisplayName: "reader",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.Normalize()
			if got.Email != tc.wantEmail {
				t.Fatalf("email = %q, want %q", got.Email, tc.wantEmail)
			}
			if got.DisplayName != tc.wantDisplayName {
				t.Fatalf("display name = %q, want %q", got.DisplayName, tc.wantDisplayName)
			}
		})
	}
}

func TestUser_HasRoleAndIsActive(t *testing.T) {
	u := User{Roles: []string{"reader", "translator"}, Status: StatusActive}

	if !u.HasRole("translator") {
		t.Fatal("expected the translator role")
	}
	if u.HasRole("admin") {
		t.Fatal("did not expect the admin role")
	}
	if !u.IsActive() {
		t.Fatal("expected an active account")
	}

	suspended := User{Status: StatusSuspended}
	if suspended.IsActive() {
		t.Fatal("a suspended account must not be active")
	}
}
