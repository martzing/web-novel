// Package httpx holds the shared HTTP wire helpers: the error envelope, the
// paginated list envelope, cursors, and Idempotency-Key parsing.
package httpx

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorBody struct {
	Error ErrorPayload `json:"error"`
}

// Error writes the documented error envelope and aborts the request.
func Error(c *gin.Context, status int, code, msg string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: ErrorPayload{Code: code, Message: msg}})
}

// Internal logs the underlying cause server-side and returns a generic 500.
// The cause is never echoed to the client.
func Internal(c *gin.Context, err error) {
	slog.Error("request failed",
		"method", c.Request.Method,
		"path", c.FullPath(),
		"err", err,
	)
	Error(c, http.StatusInternalServerError, "INTERNAL", "เกิดข้อผิดพลาดภายในระบบ")
}

// NotFound is the standard 404 envelope.
func NotFound(c *gin.Context, msg string) {
	Error(c, http.StatusNotFound, "NOT_FOUND", msg)
}

// BadRequest is the standard 400 envelope.
func BadRequest(c *gin.Context, code, msg string) {
	Error(c, http.StatusBadRequest, code, msg)
}

// Forbidden is the standard 403 envelope.
func Forbidden(c *gin.Context, msg string) {
	Error(c, http.StatusForbidden, "FORBIDDEN", msg)
}

// Unauthorized is the standard 401 envelope.
func Unauthorized(c *gin.Context, msg string) {
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", msg)
}

// IDParam parses a path parameter as an int64, writing a 400 and aborting when
// it is not numeric.
func IDParam(c *gin.Context, name, msg string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		BadRequest(c, "BAD_ID", msg)
		return 0, false
	}
	return id, true
}
