// Package notification is the HTTP adapter for the reader inbox.
package notification

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/middleware"
	notificationsvc "github.com/mokchan/webnovel-backend/internal/service/notification"
)

// Handler exposes the inbox over HTTP.
type Handler struct {
	Service *notificationsvc.Service
}

// New wires a handler onto a service.
func New(svc *notificationsvc.Service) *Handler { return &Handler{Service: svc} }

// NotificationResponse is one inbox row.
type NotificationResponse struct {
	ID        int64          `json:"id,string"`
	Kind      string         `json:"kind"`
	Payload   map[string]any `json:"payload"`
	Read      bool           `json:"read"`
	CreatedAt string         `json:"created_at"`
}

// markReadRequest carries the ids to mark read; an empty list means "all".
// The ids are plain numbers here rather than the usual string-encoded int64,
// because the `,string` option does not apply to slice elements.
type markReadRequest struct {
	IDs []int64 `json:"ids"`
}

// Register mounts the inbox routes.
//
// These are a documented extension: docs/api-spec.md predates the Phase 4
// notification requirement (R-17).
func (h *Handler) Register(r gin.IRouter, requireAuth gin.HandlerFunc) {
	me := r.Group("/me/notifications", requireAuth)
	me.GET("", h.list)
	me.GET("/unread-count", h.unreadCount)
	me.POST("/read", h.markRead)
}

func (h *Handler) list(c *gin.Context) {
	p := middleware.MustPrincipal(c)

	items, next, err := h.Service.List(
		c.Request.Context(), p.UserID, c.Query("unread") == "true", httpx.ParsePage(c, 20, 100),
	)
	if err != nil {
		if errors.Is(err, httpx.ErrBadCursor) {
			httpx.BadRequest(c, "BAD_CURSOR", "ตัวชี้หน้าถัดไปไม่ถูกต้อง")
			return
		}
		httpx.Internal(c, err)
		return
	}

	out := make([]NotificationResponse, 0, len(items))
	for _, n := range items {
		out = append(out, NotificationResponse{
			ID:        n.ID,
			Kind:      n.Kind,
			Payload:   n.Payload,
			Read:      n.ReadAt != nil,
			CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	httpx.List(c, http.StatusOK, out, next)
}

func (h *Handler) unreadCount(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	count, err := h.Service.CountUnread(c.Request.Context(), p.UserID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.OK(c, gin.H{"unread": count})
}

func (h *Handler) markRead(c *gin.Context) {
	var body markReadRequest
	_ = c.ShouldBindJSON(&body)

	p := middleware.MustPrincipal(c)
	if err := h.Service.MarkRead(c.Request.Context(), p.UserID, body.IDs); err != nil {
		httpx.Internal(c, err)
		return
	}

	count, err := h.Service.CountUnread(c.Request.Context(), p.UserID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.OK(c, gin.H{"unread": count})
}
