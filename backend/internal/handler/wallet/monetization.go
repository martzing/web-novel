package wallet

import (
	"net/http"

	"github.com/gin-gonic/gin"

	domain "github.com/mokchan/webnovel-backend/internal/domain/wallet"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/middleware"
	walletsvc "github.com/mokchan/webnovel-backend/internal/service/wallet"
)

// ArcBundleResponse is the quote a reader sees before buying a whole arc.
type ArcBundleResponse struct {
	ArcID           int64                   `json:"arc_id,string"`
	NovelID         int64                   `json:"novel_id,string"`
	ArcNo           int                     `json:"arc_no"`
	Name            string                  `json:"name"`
	ChapterCount    int                     `json:"chapter_count"`
	Gross           int                     `json:"gross"`
	DiscountPercent int                     `json:"discount_percent"`
	Discount        int                     `json:"discount"`
	Total           int                     `json:"total"`
	Chapters        []ArcBundleItemResponse `json:"chapters"`
}

// ArcBundleItemResponse is one chapter inside the quote.
type ArcBundleItemResponse struct {
	ChapterID int64 `json:"chapter_id,string"`
	ChapterNo int   `json:"chapter_no"`
	ListPrice int   `json:"list_price"`
	Coins     int   `json:"coins"`
}

// SubscriptionResponse is one auto-unlock opt-in.
type SubscriptionResponse struct {
	NovelID            int64  `json:"novel_id,string"`
	NovelTitleTH       string `json:"novel_title_th,omitempty"`
	NovelSlug          string `json:"novel_slug,omitempty"`
	Active             bool   `json:"active"`
	MaxCoinsPerChapter int    `json:"max_coins_per_chapter"`
}

type tipRequest struct {
	Coins int `json:"coins"`
}

type subscriptionRequest struct {
	Active             *bool `json:"active"`
	MaxCoinsPerChapter int   `json:"max_coins_per_chapter"`
}

func (h *Handler) tipChapter(c *gin.Context) {
	key, ok := httpx.IdempotencyKey(c)
	if !ok {
		return
	}
	id, ok := httpx.IDParam(c, "id", "รหัสบทไม่ถูกต้อง")
	if !ok {
		return
	}
	var body tipRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "จำนวนทิปไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	receipt, err := h.Service.TipChapter(c.Request.Context(), p.UserID, id, body.Coins, key)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toReceiptResponse(*receipt))
}

func (h *Handler) quoteArcBundle(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสภาคไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	quote, err := h.Service.QuoteArcBundle(c.Request.Context(), p.UserID, id)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toArcBundleResponse(*quote))
}

func (h *Handler) unlockArc(c *gin.Context) {
	key, ok := httpx.IdempotencyKey(c)
	if !ok {
		return
	}
	id, ok := httpx.IDParam(c, "id", "รหัสภาคไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	receipt, err := h.Service.UnlockArc(c.Request.Context(), p.UserID, id, key)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toReceiptResponse(*receipt))
}

func (h *Handler) listSubscriptions(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	subs, err := h.Service.ListSubscriptions(c.Request.Context(), p.UserID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	out := make([]SubscriptionResponse, 0, len(subs))
	for _, s := range subs {
		out = append(out, toSubscriptionResponse(s))
	}
	httpx.List(c, http.StatusOK, out, "")
}

func (h *Handler) putSubscription(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "novel_id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	var body subscriptionRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลไม่ถูกต้อง")
		return
	}

	// Omitting `active` means "turn it on"; the endpoint exists to subscribe.
	active := true
	if body.Active != nil {
		active = *body.Active
	}

	p := middleware.MustPrincipal(c)
	sub, err := h.Service.SetSubscription(c.Request.Context(), p.UserID, novelID, active, body.MaxCoinsPerChapter)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toSubscriptionResponse(*sub))
}

func (h *Handler) deleteSubscription(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "novel_id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	if err := h.Service.RemoveSubscription(c.Request.Context(), p.UserID, novelID); err != nil {
		httpx.Internal(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func toArcBundleResponse(q walletsvc.ArcQuote) ArcBundleResponse {
	chapters := make([]ArcBundleItemResponse, 0, len(q.Items))
	for i, item := range q.Items {
		row := ArcBundleItemResponse{
			ChapterID: item.ChapterID,
			ListPrice: item.ListPrice,
			Coins:     item.Coins,
		}
		if i < len(q.Chapters) {
			row.ChapterNo = q.Chapters[i].ChapterNo
		}
		chapters = append(chapters, row)
	}

	return ArcBundleResponse{
		ArcID:           q.ArcID,
		NovelID:         q.NovelID,
		ArcNo:           q.ArcNo,
		Name:            q.Name,
		ChapterCount:    len(chapters),
		Gross:           q.Quote.Gross,
		DiscountPercent: q.Quote.DiscountPercent,
		Discount:        q.Quote.Discount,
		Total:           q.Quote.Total,
		Chapters:        chapters,
	}
}

func toSubscriptionResponse(s domain.Subscription) SubscriptionResponse {
	return SubscriptionResponse{
		NovelID:            s.NovelID,
		NovelTitleTH:       s.NovelTitleTH,
		NovelSlug:          s.NovelSlug,
		Active:             s.Active,
		MaxCoinsPerChapter: s.MaxCoinsPerChapter,
	}
}
