package httpx

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MaxIdempotencyKeyLen bounds the client-supplied key so it always fits the
// coin_ledger column and cannot be used to bloat the index.
const MaxIdempotencyKeyLen = 200

// IdempotencyKey reads and validates the Idempotency-Key header required on
// every coin-mutating route. On failure it writes the error envelope, aborts
// the request, and reports false.
func IdempotencyKey(c *gin.Context) (string, bool) {
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" {
		Error(c, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "ต้องระบุ Idempotency-Key สำหรับคำขอนี้")
		return "", false
	}
	if len(key) > MaxIdempotencyKeyLen || !isPrintableASCII(key) {
		Error(c, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key ไม่ถูกต้อง")
		return "", false
	}
	return key, true
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x21 || s[i] > 0x7e {
			return false
		}
	}
	return true
}
