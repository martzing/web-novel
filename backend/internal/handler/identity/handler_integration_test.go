package identity_test

import (
	"net/http"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type authResponse struct {
	User struct {
		ID          string   `json:"id"`
		Username    string   `json:"username"`
		Email       string   `json:"email"`
		DisplayName string   `json:"display_name"`
		Roles       []string `json:"roles"`
		Status      string   `json:"status"`
	} `json:"user"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
}

type prefsResponse struct {
	Theme       string  `json:"theme"`
	Font        string  `json:"font"`
	FontSize    int     `json:"font_size"`
	LineHeight  float64 `json:"line_height"`
	ColumnWidth string  `json:"column_width"`
}

type genrePrefResponse struct {
	GenreID string `json:"genre_id"`
	Weight  int    `json:"weight"`
}

func register(t *testing.T, env *apitest.Env, username, email, password string) authResponse {
	t.Helper()
	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"username": username,
			"email":    email,
			"password": password,
		},
	})
	apitest.AssertStatus(t, rec, http.StatusCreated)
	return apitest.DecodeJSON[authResponse](t, rec)
}

func TestRegister_CreatesAccountWithReaderRoleAndDefaultPrefs(t *testing.T) {
	env := apitest.New(t)

	body := register(t, env, "newreader", "newreader@example.com", "hunter2hunter2")

	if body.Token == "" || body.RefreshToken == "" {
		t.Fatalf("expected both tokens, got %+v", body)
	}
	if len(body.User.Roles) != 1 || body.User.Roles[0] != entities.RoleReader {
		t.Fatalf("roles = %v, want [reader]", body.User.Roles)
	}
	if body.User.DisplayName != "newreader" {
		t.Fatalf("display name = %q, want the username", body.User.DisplayName)
	}

	// The access token must work immediately.
	rec := env.GETAuth("/api/v1/auth/me", body.Token)
	apitest.AssertStatus(t, rec, http.StatusOK)

	// A fresh account starts with a complete, valid settings object.
	rec = env.GETAuth("/api/v1/users/me/prefs", body.Token)
	apitest.AssertStatus(t, rec, http.StatusOK)
	prefs := apitest.DecodeJSON[prefsResponse](t, rec)
	if prefs.Theme != "light" || prefs.Font != "loop" || prefs.FontSize != 20 || prefs.ColumnWidth != "normal" {
		t.Fatalf("prefs = %+v, want the documented defaults", prefs)
	}
}

// I-AUTH-01 — a duplicate email is a 409 and creates no second account.
func TestRegister_DuplicateEmailReturns409AndCreatesNoUser(t *testing.T) {
	env := apitest.New(t)

	register(t, env, "first", "taken@example.com", "hunter2hunter2")

	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"username": "second",
			"email":    "taken@example.com",
			"password": "hunter2hunter2",
		},
	})
	apitest.AssertErrorCode(t, rec, http.StatusConflict, "EMAIL_TAKEN")

	var count int64
	if err := env.MakeMe.DB.Model(&entities.User{}).
		Where("email = ?", "taken@example.com").Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("users with that email = %d, want exactly 1", count)
	}
}

func TestRegister_DuplicateUsernameReturns409(t *testing.T) {
	env := apitest.New(t)

	register(t, env, "duplicate", "one@example.com", "hunter2hunter2")

	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/register",
		Body: map[string]string{
			"username": "duplicate",
			"email":    "two@example.com",
			"password": "hunter2hunter2",
		},
	})
	apitest.AssertErrorCode(t, rec, http.StatusConflict, "USERNAME_TAKEN")
}

func TestRegister_RejectsWeakPasswordAndBadEmail(t *testing.T) {
	env := apitest.New(t)

	tests := []struct {
		name string
		body map[string]string
		code string
	}{
		{"short password", map[string]string{"username": "a1", "email": "a@b.co", "password": "short"}, "INVALID_USERNAME"},
		{"weak password", map[string]string{"username": "reader1", "email": "a@b.co", "password": "short"}, "WEAK_PASSWORD"},
		{"bad email", map[string]string{"username": "reader1", "email": "nope", "password": "hunter2hunter2"}, "INVALID_EMAIL"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := env.Do(apitest.Request{Method: http.MethodPost, Path: "/api/v1/auth/register", Body: tc.body})
			apitest.AssertErrorCode(t, rec, http.StatusBadRequest, tc.code)
		})
	}
}

// I-AUTH-02 — a wrong password is a 401 whose body is indistinguishable from
// the response for an email that was never registered.
func TestLogin_WrongPasswordReturns401WithoutLeakingEmailExistence(t *testing.T) {
	env := apitest.New(t)

	register(t, env, "known", "known@example.com", "hunter2hunter2")

	wrongPassword := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body:   map[string]string{"email": "known@example.com", "password": "not-the-password"},
	})
	unknownEmail := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body:   map[string]string{"email": "never-registered@example.com", "password": "not-the-password"},
	})

	apitest.AssertErrorCode(t, wrongPassword, http.StatusUnauthorized, "INVALID_CREDENTIALS")
	apitest.AssertErrorCode(t, unknownEmail, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	if wrongPassword.Body.String() != unknownEmail.Body.String() {
		t.Fatalf("responses differ and leak whether the account exists:\n wrong password: %s\n unknown email:  %s",
			wrongPassword.Body.String(), unknownEmail.Body.String())
	}
}

func TestLogin_SucceedsWithCorrectPassword(t *testing.T) {
	env := apitest.New(t)

	registered := register(t, env, "loginuser", "loginuser@example.com", "hunter2hunter2")

	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body:   map[string]string{"email": "loginuser@example.com", "password": "hunter2hunter2"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[authResponse](t, rec)
	if body.User.ID != registered.User.ID {
		t.Fatalf("user id = %q, want %q", body.User.ID, registered.User.ID)
	}
	if body.Token == "" || body.RefreshToken == "" {
		t.Fatal("expected both tokens")
	}
}

// The email column is citext, so the login lookup is case-insensitive.
func TestLogin_EmailIsCaseInsensitive(t *testing.T) {
	env := apitest.New(t)
	register(t, env, "casetest", "casetest@example.com", "hunter2hunter2")

	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body:   map[string]string{"email": "CaseTest@Example.COM", "password": "hunter2hunter2"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)
}

// Fixture accounts carry a real argon2id hash, so they can actually sign in.
// A bcrypt-shaped placeholder would make this a 500 instead of a 401.
func TestLogin_FixtureUserCanAuthenticate(t *testing.T) {
	env := apitest.New(t)
	user := env.AUser()

	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body:   map[string]string{"email": user.Email, "password": apitest.DefaultPassword},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)
}

func TestLogin_SuspendedAccountIsForbidden(t *testing.T) {
	env := apitest.New(t)

	user := env.AUser()
	if err := env.MakeMe.DB.Model(&entities.User{}).
		Where("id = ?", user.ID).Update("status", "suspended").Error; err != nil {
		t.Fatalf("suspend user: %v", err)
	}

	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body:   map[string]string{"email": user.Email, "password": apitest.DefaultPassword},
	})
	apitest.AssertErrorCode(t, rec, http.StatusForbidden, "FORBIDDEN")
}

func TestRefresh_RotatesTheToken(t *testing.T) {
	env := apitest.New(t)
	first := register(t, env, "rotator", "rotator@example.com", "hunter2hunter2")

	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/refresh",
		Body:   map[string]string{"refresh_token": first.RefreshToken},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	second := apitest.DecodeJSON[authResponse](t, rec)
	if second.RefreshToken == first.RefreshToken {
		t.Fatal("refresh must issue a new token, not return the old one")
	}
	if second.Token == "" {
		t.Fatal("expected a fresh access token")
	}

	// The successor works.
	rec = env.GETAuth("/api/v1/auth/me", second.Token)
	apitest.AssertStatus(t, rec, http.StatusOK)
}

// I-AUTH-03 — replaying a consumed refresh token is treated as theft: it is
// rejected, and the whole rotation family is revoked so the thief's successor
// token dies too.
func TestRefresh_RevokedTokenReturns401AndRevokesFamily(t *testing.T) {
	env := apitest.New(t)
	first := register(t, env, "reuser", "reuser@example.com", "hunter2hunter2")

	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/refresh",
		Body:   map[string]string{"refresh_token": first.RefreshToken},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)
	second := apitest.DecodeJSON[authResponse](t, rec)

	t.Run("replaying the consumed token is rejected", func(t *testing.T) {
		rec := env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   "/api/v1/auth/refresh",
			Body:   map[string]string{"refresh_token": first.RefreshToken},
		})
		apitest.AssertErrorCode(t, rec, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN")
	})

	t.Run("the successor is revoked along with the family", func(t *testing.T) {
		rec := env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   "/api/v1/auth/refresh",
			Body:   map[string]string{"refresh_token": second.RefreshToken},
		})
		apitest.AssertErrorCode(t, rec, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN")
	})

	t.Run("no live sessions remain", func(t *testing.T) {
		var live int64
		if err := env.MakeMe.DB.Table("refresh_tokens").
			Where("revoked_at IS NULL").Count(&live).Error; err != nil {
			t.Fatalf("count sessions: %v", err)
		}
		if live != 0 {
			t.Fatalf("live refresh tokens = %d, want 0", live)
		}
	})
}

func TestRefresh_UnknownTokenReturns401(t *testing.T) {
	env := apitest.New(t)

	for _, token := range []string{"", "not-a-token", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY"} {
		rec := env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   "/api/v1/auth/refresh",
			Body:   map[string]string{"refresh_token": token},
		})
		apitest.AssertStatus(t, rec, http.StatusUnauthorized)
	}
}

func TestLogout_RevokesTheSession(t *testing.T) {
	env := apitest.New(t)
	session := register(t, env, "quitter", "quitter@example.com", "hunter2hunter2")

	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/logout",
		Token:  session.Token,
		Body:   map[string]string{"refresh_token": session.RefreshToken},
	})
	apitest.AssertStatus(t, rec, http.StatusNoContent)

	rec = env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/refresh",
		Body:   map[string]string{"refresh_token": session.RefreshToken},
	})
	apitest.AssertStatus(t, rec, http.StatusUnauthorized)
}

func TestProtectedRoutes_RequireAValidBearerToken(t *testing.T) {
	env := apitest.New(t)

	tests := []struct {
		name  string
		token string
	}{
		{"no token", ""},
		{"garbage token", "not-a-jwt"},
		{"token signed with another secret", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.wrong"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := env.GETAuth("/api/v1/users/me", tc.token)
			apitest.AssertStatus(t, rec, http.StatusUnauthorized)
		})
	}
}

func TestUpdateMe_PatchesProfileFields(t *testing.T) {
	env := apitest.New(t)
	session := register(t, env, "editor", "editor@example.com", "hunter2hunter2")

	rec := env.Do(apitest.Request{
		Method: http.MethodPatch,
		Path:   "/api/v1/users/me",
		Token:  session.Token,
		Body:   map[string]string{"display_name": "หมอกจันทร์"},
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[struct {
		DisplayName string `json:"display_name"`
		Username    string `json:"username"`
	}](t, rec)
	if body.DisplayName != "หมอกจันทร์" {
		t.Fatalf("display name = %q, want the patched value", body.DisplayName)
	}
	if body.Username != "editor" {
		t.Fatalf("username = %q, want it unchanged", body.Username)
	}
}

// E-RD-02's backend half: preferences round-trip through the API so a second
// device sees the same reader settings.
func TestPrefs_RoundTripAndValidation(t *testing.T) {
	env := apitest.New(t)
	session := register(t, env, "prefsuser", "prefsuser@example.com", "hunter2hunter2")

	t.Run("valid preferences are stored", func(t *testing.T) {
		rec := env.Do(apitest.Request{
			Method: http.MethodPut,
			Path:   "/api/v1/users/me/prefs",
			Token:  session.Token,
			Body: map[string]any{
				"theme": "sepia", "font": "serif", "font_size": 22,
				"line_height": 2.2, "column_width": "wide",
			},
		})
		apitest.AssertStatus(t, rec, http.StatusOK)

		rec = env.GETAuth("/api/v1/users/me/prefs", session.Token)
		prefs := apitest.DecodeJSON[prefsResponse](t, rec)
		want := prefsResponse{Theme: "sepia", Font: "serif", FontSize: 22, LineHeight: 2.2, ColumnWidth: "wide"}
		if prefs != want {
			t.Fatalf("prefs = %+v, want %+v", prefs, want)
		}
	})

	t.Run("out-of-range values are rejected before hitting the constraint", func(t *testing.T) {
		bodies := []map[string]any{
			{"theme": "midnight", "font": "serif", "font_size": 22, "line_height": 2.0, "column_width": "wide"},
			{"theme": "sepia", "font": "comic", "font_size": 22, "line_height": 2.0, "column_width": "wide"},
			{"theme": "sepia", "font": "serif", "font_size": 99, "line_height": 2.0, "column_width": "wide"},
			{"theme": "sepia", "font": "serif", "font_size": 22, "line_height": 9.9, "column_width": "wide"},
			{"theme": "sepia", "font": "serif", "font_size": 22, "line_height": 2.0, "column_width": "ultra"},
		}
		for _, body := range bodies {
			rec := env.Do(apitest.Request{
				Method: http.MethodPut,
				Path:   "/api/v1/users/me/prefs",
				Token:  session.Token,
				Body:   body,
			})
			apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "INVALID_PREFS")
		}
	})
}

// R-18 — onboarding genre chips are saved to user_genre_prefs.
func TestGenrePrefs_ReplacesTheWholeSet(t *testing.T) {
	env := apitest.New(t)
	session := register(t, env, "taste", "taste@example.com", "hunter2hunter2")
	m := env.MakeMe

	xianxia := m.ANewGenre().Please()
	wuxia := m.ANewGenre().Please()
	rebirth := m.ANewGenre().Please()

	put := func(t *testing.T, genres []map[string]any) []genrePrefResponse {
		t.Helper()
		rec := env.Do(apitest.Request{
			Method: http.MethodPut,
			Path:   "/api/v1/users/me/genre-prefs",
			Token:  session.Token,
			Body:   map[string]any{"genres": genres},
		})
		apitest.AssertStatus(t, rec, http.StatusOK)
		return apitest.DecodeJSON[apitest.List[genrePrefResponse]](t, rec).Data
	}

	first := put(t, []map[string]any{
		{"genre_id": itoa(xianxia.ID), "weight": 3},
		{"genre_id": itoa(wuxia.ID), "weight": 1},
	})
	if len(first) != 2 {
		t.Fatalf("saved %d prefs, want 2", len(first))
	}

	// A second PUT replaces rather than merges.
	second := put(t, []map[string]any{{"genre_id": itoa(rebirth.ID), "weight": 2}})
	if len(second) != 1 || second[0].GenreID != itoa(rebirth.ID) {
		t.Fatalf("prefs = %+v, want only the rebirth genre", second)
	}
}

// I-SEC-03 — the auth endpoints are rate-limited per IP.
func TestAuthRoutes_RateLimitedPerIP(t *testing.T) {
	cfg := apitest.Config()
	cfg.RateLimitEnabled = true
	cfg.AuthRatePerMin = 5
	env := apitest.NewWith(t, cfg)

	attempt := func() int {
		return env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   "/api/v1/auth/login",
			Body:   map[string]string{"email": "nobody@example.com", "password": "wrong"},
		}).Code
	}

	for i := range cfg.AuthRatePerMin {
		if code := attempt(); code == http.StatusTooManyRequests {
			t.Fatalf("request %d was throttled before the limit was reached", i+1)
		}
	}

	rec := env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/auth/login",
		Body:   map[string]string{"email": "nobody@example.com", "password": "wrong"},
	})
	apitest.AssertErrorCode(t, rec, http.StatusTooManyRequests, "RATE_LIMITED")
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("a throttled response must carry Retry-After")
	}

	// The catalog is unaffected: the limiter is scoped to /auth.
	apitest.AssertStatus(t, env.GET("/api/v1/genres"), http.StatusOK)
}

func itoa(id int64) string {
	const digits = "0123456789"
	if id == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for id > 0 {
		i--
		buf[i] = digits[id%10]
		id /= 10
	}
	return string(buf[i:])
}
