// Package library is the HTTP adapter for the reader's shelf.
package library

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	domain "github.com/mokchan/webnovel-backend/internal/domain/library"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/middleware"
	librarysvc "github.com/mokchan/webnovel-backend/internal/service/library"
)

// Handler exposes shelf, bookmark and follow use cases over HTTP.
type Handler struct {
	Service *librarysvc.Service
}

// New wires a handler onto a service.
func New(svc *librarysvc.Service) *Handler { return &Handler{Service: svc} }

// Register mounts the shelf routes. Every one of them is caller-scoped.
func (h *Handler) Register(r gin.IRouter, requireAuth gin.HandlerFunc) {
	me := r.Group("/me", requireAuth)

	me.GET("/library", h.listShelf)
	me.PUT("/library/:novel_id", h.putShelf)
	me.DELETE("/library/:novel_id", h.deleteShelf)

	me.GET("/bookmarks", h.listBookmarks)
	me.POST("/bookmarks", h.createBookmark)
	me.DELETE("/bookmarks/:id", h.deleteBookmark)

	me.POST("/follows/:novel_id", h.follow)
	me.DELETE("/follows/:novel_id", h.unfollow)
	me.GET("/follows/:novel_id", h.isFollowing)

	// ติดตามทั้งชุด lives at /series/:id/follow rather than under /me/follows/…
	// for a structural reason: /me/follows/:novel_id already owns that segment,
	// and a second wildcard name in it panics the router at startup. The `:id`
	// here matches the catalog's own /series/:id, so the two coexist.
	series := r.Group("/series/:id/follow", requireAuth)
	series.POST("", h.followSeries)
	series.DELETE("", h.unfollowSeries)
	series.GET("", h.seriesFollowState)
}

func (h *Handler) listShelf(c *gin.Context) {
	p := middleware.MustPrincipal(c)

	entries, next, err := h.Service.ListShelf(c.Request.Context(), p.UserID, c.Query("tab"), httpx.ParsePage(c, 20, 100))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	counts, err := h.Service.CountShelf(c.Request.Context(), p.UserID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        toShelfResponses(entries),
		"next_cursor": next,
		"counts": CountsResponse{
			Reading: counts.Reading,
			Saved:   counts.Saved,
			Done:    counts.Done,
			Total:   counts.Reading + counts.Saved + counts.Done,
		},
	})
}

func (h *Handler) putShelf(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "novel_id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	var body shelfRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	entry, err := h.Service.SetShelfStatus(c.Request.Context(), p.UserID, novelID, body.Status)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toEntryResponse(*entry))
}

func (h *Handler) deleteShelf(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "novel_id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	if err := h.Service.RemoveFromShelf(c.Request.Context(), p.UserID, novelID); err != nil {
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listBookmarks(c *gin.Context) {
	var novelID *int64
	if raw := c.Query("novel_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			httpx.BadRequest(c, "BAD_ID", "รหัสนิยายไม่ถูกต้อง")
			return
		}
		novelID = &id
	}

	p := middleware.MustPrincipal(c)
	bookmarks, next, err := h.Service.ListBookmarks(c.Request.Context(), p.UserID, novelID, httpx.ParsePage(c, 50, 200))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toBookmarkResponses(bookmarks), next)
}

func (h *Handler) createBookmark(c *gin.Context) {
	var body bookmarkRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลที่คั่นไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	bookmark, err := h.Service.CreateBookmark(c.Request.Context(), domain.Bookmark{
		UserID:     p.UserID,
		NovelID:    body.NovelID,
		ChapterID:  body.ChapterID,
		ParaAnchor: body.ParaAnchor,
		Excerpt:    body.Excerpt,
		Note:       body.Note,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, toBookmarkResponse(*bookmark))
}

func (h *Handler) deleteBookmark(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสที่คั่นไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	if err := h.Service.DeleteBookmark(c.Request.Context(), id, p.UserID); err != nil {
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) follow(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "novel_id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	if err := h.Service.Follow(c.Request.Context(), p.UserID, novelID); err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"following": true})
}

func (h *Handler) unfollow(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "novel_id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	if err := h.Service.Unfollow(c.Request.Context(), p.UserID, novelID); err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"following": false})
}

func (h *Handler) isFollowing(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "novel_id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	following, err := h.Service.IsFollowing(c.Request.Context(), p.UserID, novelID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.OK(c, gin.H{"following": following})
}

func (h *Handler) followSeries(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	state, err := h.Service.FollowSeries(c.Request.Context(), p.UserID, c.Param("id"))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toSeriesFollowResponse(state))
}

func (h *Handler) unfollowSeries(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	state, err := h.Service.UnfollowSeries(c.Request.Context(), p.UserID, c.Param("id"))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toSeriesFollowResponse(state))
}

func (h *Handler) seriesFollowState(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	state, err := h.Service.SeriesFollowState(c.Request.Context(), p.UserID, c.Param("id"))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toSeriesFollowResponse(state))
}

func (h *Handler) writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		httpx.Forbidden(c, "รายการนี้ไม่ใช่ของคุณ")
	case errors.Is(err, domain.ErrNotFound):
		httpx.NotFound(c, "ไม่พบรายการนี้")
	case errors.Is(err, domain.ErrInvalidStatus):
		httpx.BadRequest(c, "INVALID_STATUS", "สถานะชั้นหนังสือไม่ถูกต้อง")
	case errors.Is(err, domain.ErrInvalidInput):
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลไม่ถูกต้อง")
	case errors.Is(err, httpx.ErrBadCursor):
		httpx.BadRequest(c, "BAD_CURSOR", "ตัวชี้หน้าถัดไปไม่ถูกต้อง")
	default:
		httpx.Internal(c, err)
	}
}
