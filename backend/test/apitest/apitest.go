// Package apitest is the shared harness for handler integration tests: it
// builds a real Gin engine over a testcontainers Postgres, mints access tokens,
// and performs requests through httptest.
package apitest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"github.com/mokchan/webnovel-backend/internal/auth"
	"github.com/mokchan/webnovel-backend/internal/config"
	"github.com/mokchan/webnovel-backend/internal/crypto/argon2id"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/server"
	"github.com/mokchan/webnovel-backend/test/makeme"
)

// TestJWTSecret is long enough to satisfy auth.NewIssuer's minimum.
const TestJWTSecret = "integration-test-secret-key"

// DefaultPassword is the plaintext behind every fixture user's password hash.
const DefaultPassword = makeme.DefaultPassword

// Config returns a configuration suitable for handler tests. Rate limiting is
// off by default so unrelated tests are not throttled; tests that assert on
// throttling turn it on explicitly.
func Config() *config.Config {
	return &config.Config{
		Env:                 "test",
		CORSOrigins:         []string{"*"},
		JWTSecret:           TestJWTSecret,
		AccessTokenTTL:      15 * time.Minute,
		RefreshTokenTTL:     720 * time.Hour,
		Argon2Memory:        8 * 1024, // keep hashing fast in tests
		Argon2Time:          1,
		Argon2Parallelism:   1,
		RateLimitEnabled:    false,
		AuthRatePerMin:      60,
		BodyFetchPerMin:     20,
		PaymentsMockEnabled: true,
		BonusTTL:            720 * time.Hour,
		PlatformFeePercent:  30,
		UploadDir:           "",
		PublicUploadBase:    "",
		RunJobsInAPI:        false,
	}
}

// Env is one configured engine plus the database it reads.
type Env struct {
	T      testing.TB
	MakeMe *makeme.MakeMe
	Engine *gin.Engine
	Config *config.Config
}

// New boots a clean database and engine with the default test configuration.
func New(t testing.TB) *Env { return NewWith(t, Config()) }

// NewWith boots a clean database and engine with a caller-supplied config, for
// tests that need a feature flag flipped.
func NewWith(t testing.TB, cfg *config.Config) *Env {
	t.Helper()
	gin.SetMode(gin.TestMode)

	m := makeme.New(t)
	m.Reset()

	engine, err := server.Build(cfg, m.DB)
	if err != nil {
		t.Fatalf("build server: %v", err)
	}
	return &Env{T: t, MakeMe: m, Engine: engine, Config: cfg}
}

// Hasher matches the argon2id settings the engine was built with.
func (e *Env) Hasher() *argon2id.Hasher {
	return argon2id.New(argon2id.Params{
		Memory:      e.Config.Argon2Memory,
		Time:        e.Config.Argon2Time,
		Parallelism: e.Config.Argon2Parallelism,
	})
}

// Token mints a valid access token for the given user.
func (e *Env) Token(userID int64, roles ...string) string {
	e.T.Helper()
	if len(roles) == 0 {
		roles = []string{entities.RoleReader}
	}
	issuer, err := auth.NewIssuer(e.Config.JWTSecret, e.Config.AccessTokenTTL)
	if err != nil {
		e.T.Fatalf("new issuer: %v", err)
	}
	token, _, err := issuer.Issue(userID, roles, time.Now())
	if err != nil {
		e.T.Fatalf("issue token: %v", err)
	}
	return token
}

// TokenFor mints an access token carrying the user's persisted roles.
func (e *Env) TokenFor(u *entities.User) string {
	e.T.Helper()
	roles := []string(u.Roles)
	if len(roles) == 0 {
		roles = []string{entities.RoleReader}
	}
	return e.Token(u.ID, roles...)
}

// Request is one HTTP call through the engine.
type Request struct {
	Method  string
	Path    string
	Body    any
	Token   string
	Headers map[string]string
}

// Do performs the request and returns the recorder.
func (e *Env) Do(r Request) *httptest.ResponseRecorder {
	e.T.Helper()

	var reader *bytes.Reader
	hasBody := r.Body != nil
	if hasBody {
		raw, err := json.Marshal(r.Body)
		if err != nil {
			e.T.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(r.Method, r.Path, reader)
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.Token != "" {
		req.Header.Set("Authorization", "Bearer "+r.Token)
	}
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	e.Engine.ServeHTTP(rec, req)
	return rec
}

// GET performs an unauthenticated GET.
func (e *Env) GET(path string) *httptest.ResponseRecorder {
	return e.Do(Request{Method: http.MethodGet, Path: path})
}

// GETAuth performs an authenticated GET.
func (e *Env) GETAuth(path, token string) *httptest.ResponseRecorder {
	return e.Do(Request{Method: http.MethodGet, Path: path, Token: token})
}

// AUser creates a fixture user with a real argon2id password hash so it can
// actually log in, and the roles the caller asked for.
func (e *Env) AUser(roles ...string) *entities.User {
	e.T.Helper()
	if len(roles) == 0 {
		roles = []string{entities.RoleReader}
	}
	return e.MakeMe.ANewUser().With(func(u *entities.User) {
		u.Roles = pq.StringArray(roles)
	}).Create()
}

// DecodeJSON unmarshals a recorder body into T.
func DecodeJSON[T any](t testing.TB, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var body T
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode JSON body %q: %v", rec.Body.String(), err)
	}
	return body
}

// AssertStatus fails the test when the response status differs.
func AssertStatus(t testing.TB, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, want, rec.Body.String())
	}
}

// AssertErrorCode asserts both the status and the error envelope's code.
func AssertErrorCode(t testing.TB, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	AssertStatus(t, rec, status)
	body := DecodeJSON[ErrorEnvelope](t, rec)
	if body.Error.Code != code {
		t.Fatalf("error code = %q, want %q; body = %s", body.Error.Code, code, rec.Body.String())
	}
}

// Ptr returns a pointer to value, for building fixture fields.
func Ptr[T any](value T) *T { return &value }

// ErrorEnvelope mirrors the documented error shape.
type ErrorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// List is the documented paginated envelope.
type List[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor"`
}
