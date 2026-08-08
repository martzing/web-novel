// Package middleware holds the Gin middleware shared by every handler package:
// authentication, role checks and rate limiting.
package middleware

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mokchan/webnovel-backend/internal/auth"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/httpx"
)

const principalKey = "mokchan.principal"

// Principal is the authenticated caller. The zero value is anonymous.
type Principal struct {
	UserID int64
	Roles  []string
}

// IsAnonymous reports whether no user is attached to the request.
func (p Principal) IsAnonymous() bool { return p.UserID == 0 }

// Has reports whether the principal carries the given role.
func (p Principal) Has(role string) bool { return slices.Contains(p.Roles, role) }

// IsAdmin reports whether the principal is a platform administrator.
func (p Principal) IsAdmin() bool { return p.Has(entities.RoleAdmin) }

// TokenParser is the narrow slice of the JWT issuer this package needs, so
// middleware tests can supply a fake without touching crypto.
type TokenParser interface {
	Parse(token string, now time.Time) (*auth.Claims, error)
}

// Clock supplies the current time; nil defaults to time.Now.
type Clock func() time.Time

func (c Clock) now() time.Time {
	if c == nil {
		return time.Now()
	}
	return c()
}

// RequireAuth rejects the request with 401 unless a valid bearer token is
// present.
func RequireAuth(parser TokenParser, clock Clock) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := parseBearer(c, parser, clock)
		if !ok {
			httpx.Unauthorized(c, "ต้องเข้าสู่ระบบก่อนใช้งาน")
			return
		}
		c.Set(principalKey, Principal{UserID: claims.UserID, Roles: claims.Roles})
		c.Next()
	}
}

// OptionalAuth attaches a principal when a valid token is present and otherwise
// continues anonymously.
//
// An absent, malformed, expired or badly signed token all mean "anonymous" and
// never 401 — otherwise a stale token left in a reader's browser would lock
// them out of free chapters they are entitled to read.
func OptionalAuth(parser TokenParser, clock Clock) gin.HandlerFunc {
	return func(c *gin.Context) {
		if claims, ok := parseBearer(c, parser, clock); ok {
			c.Set(principalKey, Principal{UserID: claims.UserID, Roles: claims.Roles})
		}
		c.Next()
	}
}

// RequireRole rejects the request with 403 unless the principal holds at least
// one of the given roles. Administrators always pass.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := PrincipalFrom(c)
		if !ok {
			httpx.Unauthorized(c, "ต้องเข้าสู่ระบบก่อนใช้งาน")
			return
		}
		if p.IsAdmin() {
			c.Next()
			return
		}
		if slices.ContainsFunc(roles, p.Has) {
			c.Next()
			return
		}
		httpx.Forbidden(c, "ไม่มีสิทธิ์เข้าถึงส่วนนี้")
	}
}

// PrincipalFrom returns the authenticated caller, if any.
func PrincipalFrom(c *gin.Context) (Principal, bool) {
	v, ok := c.Get(principalKey)
	if !ok {
		return Principal{}, false
	}
	p, ok := v.(Principal)
	return p, ok
}

// MustPrincipal returns the caller behind a RequireAuth-guarded route.
func MustPrincipal(c *gin.Context) Principal {
	p, ok := PrincipalFrom(c)
	if !ok {
		// Unreachable behind RequireAuth; abort defensively rather than panic.
		httpx.Unauthorized(c, "ต้องเข้าสู่ระบบก่อนใช้งาน")
		return Principal{}
	}
	return p
}

// ViewerID returns the caller's user id, or 0 when anonymous.
func ViewerID(c *gin.Context) int64 {
	p, ok := PrincipalFrom(c)
	if !ok {
		return 0
	}
	return p.UserID
}

func parseBearer(c *gin.Context, parser TokenParser, clock Clock) (*auth.Claims, bool) {
	header := c.GetHeader("Authorization")
	if header == "" {
		return nil, false
	}
	token, found := strings.CutPrefix(header, "Bearer ")
	if !found {
		return nil, false
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, false
	}
	claims, err := parser.Parse(token, clock.now())
	if err != nil {
		return nil, false
	}
	return claims, true
}

// abortRateLimited writes the shared 429 response.
func abortRateLimited(c *gin.Context, retryAfter time.Duration) {
	seconds := max(int(retryAfter.Seconds()), 1)
	c.Header("Retry-After", strconv.Itoa(seconds))
	httpx.Error(c, http.StatusTooManyRequests, "RATE_LIMITED", "คำขอถี่เกินไป กรุณาลองใหม่อีกครั้ง")
}
