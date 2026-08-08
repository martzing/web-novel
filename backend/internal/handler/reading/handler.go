// Package reading is the HTTP adapter for the chapter reader.
package reading

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	domain "github.com/mokchan/webnovel-backend/internal/domain/reading"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/middleware"
	readingsvc "github.com/mokchan/webnovel-backend/internal/service/reading"
)

// Handler exposes reader use cases over HTTP.
type Handler struct {
	Service *readingsvc.Service
}

// New wires a handler onto a service.
func New(svc *readingsvc.Service) *Handler { return &Handler{Service: svc} }

// Register mounts the reader routes.
//
// GET /chapters/:id lives here, not in the catalog handler: registering the
// same path in two packages panics at startup.
func (h *Handler) Register(r gin.IRouter, optionalAuth, requireAuth, bodyLimit gin.HandlerFunc) {
	r.GET("/chapters/:id", optionalAuth, bodyLimit, h.getChapter)
	r.GET("/chapters/:id/next", optionalAuth, h.next)
	r.GET("/chapters/:id/prev", optionalAuth, h.prev)
	r.POST("/chapters/:id/read-event", optionalAuth, h.readEvent)

	r.GET("/me/progress/:novel_id", requireAuth, h.getProgress)
	r.PUT("/me/progress/:novel_id", requireAuth, h.putProgress)
}

func (h *Handler) getChapter(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสบทไม่ถูกต้อง")
	if !ok {
		return
	}
	view, err := h.Service.GetChapter(c.Request.Context(), id, viewer(c))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toChapterViewResponse(*view))
}

func (h *Handler) next(c *gin.Context) { h.neighbour(c, "next") }
func (h *Handler) prev(c *gin.Context) { h.neighbour(c, "prev") }

func (h *Handler) neighbour(c *gin.Context, direction string) {
	id, ok := httpx.IDParam(c, "id", "รหัสบทไม่ถูกต้อง")
	if !ok {
		return
	}
	neighbourID, err := h.Service.Neighbour(c.Request.Context(), id, direction, viewer(c))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	if neighbourID == nil {
		httpx.NotFound(c, "ไม่มีบทถัดไป")
		return
	}
	httpx.OK(c, neighbourResponse{ID: neighbourID})
}

// readEvent is fire-and-forget: it always answers 202 so a failed analytics
// write never breaks the reading experience.
func (h *Handler) readEvent(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสบทไม่ถูกต้อง")
	if !ok {
		return
	}
	var body readEventRequest
	_ = c.ShouldBindJSON(&body)

	if err := h.Service.RecordRead(c.Request.Context(), id, middleware.ViewerID(c), body.SessionID); err != nil {
		httpx.Internal(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

func (h *Handler) getProgress(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "novel_id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	p := middleware.MustPrincipal(c)
	progress, err := h.Service.GetProgress(c.Request.Context(), p.UserID, novelID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	if progress == nil {
		httpx.NotFound(c, "ยังไม่มีความคืบหน้าในการอ่าน")
		return
	}
	httpx.OK(c, toProgressResponse(*progress))
}

func (h *Handler) putProgress(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "novel_id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	var body progressRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	saved, err := h.Service.SaveProgress(c.Request.Context(), domain.Progress{
		UserID:        p.UserID,
		NovelID:       novelID,
		LastChapterID: body.LastChapterID,
		LastChapterNo: body.LastChapterNo,
		ParaAnchor:    body.ParaAnchor,
		Pct:           body.Pct,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toProgressResponse(*saved))
}

func viewer(c *gin.Context) domain.Viewer {
	p, ok := middleware.PrincipalFrom(c)
	if !ok {
		return domain.Viewer{}
	}
	return domain.Viewer{UserID: p.UserID, Roles: p.Roles}
}

func (h *Handler) writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		httpx.NotFound(c, "ไม่พบบทนี้")
	case errors.Is(err, domain.ErrInvalidProgress):
		httpx.BadRequest(c, "INVALID_PROGRESS", "ตำแหน่งการอ่านไม่ถูกต้อง")
	default:
		httpx.Internal(c, err)
	}
}
