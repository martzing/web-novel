package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/mokchan/webnovel-backend/internal/ratelimit"
)

// RateLimitByIP throttles per client IP.
//
// Gin's ClientIP honours X-Forwarded-For, so the engine must call
// SetTrustedProxies with an explicit allowlist (or nil). Without that a caller
// defeats this limiter by rotating the header.
func RateLimitByIP(limiter ratelimit.Limiter) gin.HandlerFunc {
	if limiter == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		if ok, retryAfter := limiter.Allow(c.ClientIP()); !ok {
			abortRateLimited(c, retryAfter)
			return
		}
		c.Next()
	}
}

// LimitDistinctChapterBodies caps how many distinct chapters one caller may
// fetch the body of inside the counter's window. Repeat reads of the same
// chapter are always allowed; only breadth is limited.
func LimitDistinctChapterBodies(counter *ratelimit.DistinctCounter, param string) gin.HandlerFunc {
	if counter == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		key := c.ClientIP()
		if id := ViewerID(c); id != 0 {
			key = "user:" + strconv.FormatInt(id, 10)
		}
		if ok, retryAfter := counter.Observe(key, c.Param(param)); !ok {
			abortRateLimited(c, retryAfter)
			return
		}
		c.Next()
	}
}
