// Package wallet is the HTTP adapter for the coin economy.
package wallet

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mokchan/webnovel-backend/internal/domain/roles"
	domain "github.com/mokchan/webnovel-backend/internal/domain/wallet"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/middleware"
	walletsvc "github.com/mokchan/webnovel-backend/internal/service/wallet"
)

// Handler exposes coin use cases over HTTP.
type Handler struct {
	Service *walletsvc.Service
	// MockPaymentsEnabled gates the mock-completion routes.
	MockPaymentsEnabled bool
}

// New wires a handler onto a service.
func New(svc *walletsvc.Service, mockPaymentsEnabled bool) *Handler {
	return &Handler{Service: svc, MockPaymentsEnabled: mockPaymentsEnabled}
}

// Register mounts the coin routes.
//
// The mock-payment routes are registered only when the feature flag is on, so
// disabling it yields a structural 404 from gin's NoRoute rather than relying
// on a check inside a handler that someone could forget (I-COIN-08).
func (h *Handler) Register(r gin.IRouter, requireAuth gin.HandlerFunc) {
	r.GET("/coin-packs", h.listPacks)

	authed := r.Group("", requireAuth)
	authed.GET("/me/wallet", h.getBalance)
	authed.GET("/me/wallet/ledger", h.listLedger)
	authed.POST("/purchases", h.createPurchase)
	authed.POST("/chapters/:id/unlock", h.unlockChapter)
	authed.POST("/chapters/:id/tip", h.tipChapter)

	// /arcs/:id is a fresh path segment, so it cannot collide with the
	// /novels/:id wildcard-name rule that gin enforces per segment.
	authed.GET("/arcs/:id/bundle", h.quoteArcBundle)
	authed.POST("/arcs/:id/unlock", h.unlockArc)

	authed.GET("/me/auto-unlock", h.listSubscriptions)
	authed.PUT("/me/auto-unlock/:novel_id", h.putSubscription)
	authed.DELETE("/me/auto-unlock/:novel_id", h.deleteSubscription)

	if h.MockPaymentsEnabled {
		authed.POST("/purchases/:id/mock-complete", h.mockComplete)
		authed.POST("/purchases/:id/mock-fail", h.mockFail)
	}

	writer := r.Group("/writer", requireAuth, middleware.RequireRole(roles.Translator))
	writer.GET("/earnings", h.listEarnings)
	writer.POST("/payouts", h.requestPayout)

	admin := r.Group("/admin", requireAuth, middleware.RequireRole(roles.Admin))
	admin.POST("/wallet-adjust", h.adjust)
}

func (h *Handler) getBalance(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	balance, err := h.Service.GetBalance(c.Request.Context(), p.UserID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.OK(c, toBalanceResponse(*balance))
}

func (h *Handler) listLedger(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	entries, next, err := h.Service.ListLedger(c.Request.Context(), p.UserID, httpx.ParsePage(c, 20, 100))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toLedgerResponses(entries), next)
}

func (h *Handler) listPacks(c *gin.Context) {
	packs, err := h.Service.ListPacks(c.Request.Context())
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toPackResponses(packs), "")
}

func (h *Handler) createPurchase(c *gin.Context) {
	key, ok := httpx.IdempotencyKey(c)
	if !ok {
		return
	}
	var body createPurchaseRequest
	if err := c.ShouldBindJSON(&body); err != nil || body.PackID <= 0 {
		httpx.BadRequest(c, "INVALID_BODY", "กรุณาเลือกแพ็กเหรียญ")
		return
	}

	p := middleware.MustPrincipal(c)
	purchase, err := h.Service.CreatePurchase(c.Request.Context(), p.UserID, body.PackID, key)
	if err != nil {
		h.writeErr(c, err)
		return
	}

	c.JSON(http.StatusCreated, PurchaseResponse{
		PurchaseID:      purchase.ID,
		Status:          purchase.Status,
		AmountSatang:    purchase.AmountSatang,
		Currency:        purchase.Currency,
		Provider:        purchase.Provider,
		MockCheckoutURL: fmt.Sprintf("/checkout/%d", purchase.ID),
	})
}

func (h *Handler) mockComplete(c *gin.Context) {
	key, ok := httpx.IdempotencyKey(c)
	if !ok {
		return
	}
	id, ok := httpx.IDParam(c, "id", "รหัสคำสั่งซื้อไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	receipt, err := h.Service.CompletePurchase(c.Request.Context(), p.UserID, id, key)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toReceiptResponse(*receipt))
}

func (h *Handler) mockFail(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสคำสั่งซื้อไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	purchase, err := h.Service.FailPurchase(c.Request.Context(), p.UserID, id)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"purchase_id": fmt.Sprint(purchase.ID), "status": purchase.Status})
}

func (h *Handler) unlockChapter(c *gin.Context) {
	key, ok := httpx.IdempotencyKey(c)
	if !ok {
		return
	}
	id, ok := httpx.IDParam(c, "id", "รหัสบทไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	receipt, err := h.Service.UnlockChapter(c.Request.Context(), p.UserID, id, key)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toReceiptResponse(*receipt))
}

func (h *Handler) adjust(c *gin.Context) {
	key, ok := httpx.IdempotencyKey(c)
	if !ok {
		return
	}
	var body walletAdjustRequest
	if err := c.ShouldBindJSON(&body); err != nil || body.UserID <= 0 {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลการปรับยอดไม่ถูกต้อง")
		return
	}
	if body.Reason == "" {
		httpx.BadRequest(c, "REASON_REQUIRED", "ต้องระบุเหตุผลในการปรับยอด")
		return
	}

	actor := middleware.MustPrincipal(c)
	receipt, err := h.Service.Adjust(
		c.Request.Context(), body.UserID, body.Delta, body.BonusDelta, body.Reason, actor.UserID, key,
	)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toReceiptResponse(*receipt))
}

func (h *Handler) listEarnings(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	earnings, next, err := h.Service.ListEarnings(c.Request.Context(), p.UserID, httpx.ParsePage(c, 20, 100))
	if err != nil {
		h.writeErr(c, err)
		return
	}

	available, err := h.Service.AvailableEarnings(c.Request.Context(), p.UserID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":             toEarningResponses(earnings),
		"next_cursor":      next,
		"available_satang": available,
	})
}

func (h *Handler) requestPayout(c *gin.Context) {
	var body payoutRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลคำขอถอนเงินไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	payout, err := h.Service.RequestPayout(c.Request.Context(), p.UserID, body.AmountSatang)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, toPayoutResponse(*payout))
}

func (h *Handler) writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInsufficientCoins):
		httpx.Error(c, http.StatusPaymentRequired, "INSUFFICIENT_COINS", "เหรียญไม่พอ กรุณาเติมเหรียญก่อน")
	case errors.Is(err, domain.ErrAlreadyUnlocked):
		httpx.Error(c, http.StatusConflict, "CHAPTER_ALREADY_UNLOCKED", "คุณปลดล็อกบทนี้ไว้แล้ว")
	case errors.Is(err, domain.ErrChapterNotForSale):
		httpx.BadRequest(c, "CHAPTER_NOT_FOR_SALE", "บทนี้อ่านฟรี ไม่ต้องปลดล็อก")

	// A tip paid from bonus coins would turn promotional currency into a real
	// payout obligation, so it is refused with its own code — "เหรียญไม่พอ"
	// beside a healthy bonus balance would be baffling.
	case errors.Is(err, domain.ErrInsufficientPaidCoins):
		httpx.Error(c, http.StatusPaymentRequired, "INSUFFICIENT_PAID_COINS",
			"ทิปใช้ได้เฉพาะเหรียญที่ซื้อ ไม่รวมเหรียญโบนัส")
	case errors.Is(err, domain.ErrTipsDisabled):
		httpx.BadRequest(c, "TIPS_DISABLED", "ผู้แปลยังไม่เปิดรับทิปสำหรับเรื่องนี้")
	case errors.Is(err, domain.ErrCannotTipSelf):
		httpx.BadRequest(c, "CANNOT_TIP_SELF", "ไม่สามารถให้ทิปกับผลงานของตัวเองได้")

	case errors.Is(err, domain.ErrArcNotForSale):
		httpx.BadRequest(c, "ARC_NOT_FOR_SALE", "ภาคนี้ยังไม่เปิดขายแบบเหมาภาค")
	case errors.Is(err, domain.ErrArcAlreadyOwned):
		httpx.Error(c, http.StatusConflict, "ARC_ALREADY_OWNED", "คุณเป็นเจ้าของทุกบทในภาคนี้แล้ว")
	case errors.Is(err, domain.ErrBundleStale):
		httpx.Error(c, http.StatusConflict, "ARC_BUNDLE_STALE", "รายการในภาคเปลี่ยนไป กรุณาลองใหม่")

	case errors.Is(err, domain.ErrEarlyAccessOnly):
		// A distinct code, not a bare FORBIDDEN: the client's response is to
		// offer auto-unlock, which it cannot know from a generic refusal.
		httpx.Error(c, http.StatusForbidden, "EARLY_ACCESS_ONLY",
			"บทนี้เปิดให้ผู้ที่เปิดปลดล็อกอัตโนมัติอ่านก่อน")
	case errors.Is(err, domain.ErrIdempotencyConflict):
		httpx.Error(c, http.StatusConflict, "IDEMPOTENCY_KEY_CONFLICT", "Idempotency-Key นี้ถูกใช้กับคำสั่งอื่นแล้ว")
	case errors.Is(err, domain.ErrPurchaseNotPending):
		httpx.Error(c, http.StatusConflict, "PURCHASE_NOT_PENDING", "คำสั่งซื้อนี้ถูกดำเนินการไปแล้ว")
	case errors.Is(err, domain.ErrPackNotFound):
		httpx.NotFound(c, "ไม่พบแพ็กเหรียญนี้")
	case errors.Is(err, domain.ErrInvalidAmount):
		httpx.BadRequest(c, "INVALID_AMOUNT", "จำนวนไม่ถูกต้อง")
	case errors.Is(err, domain.ErrNotFound):
		httpx.NotFound(c, "ไม่พบรายการนี้")
	case errors.Is(err, httpx.ErrBadCursor):
		httpx.BadRequest(c, "BAD_CURSOR", "ตัวชี้หน้าถัดไปไม่ถูกต้อง")
	default:
		httpx.Internal(c, err)
	}
}
