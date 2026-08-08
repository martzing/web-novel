// Package social is the HTTP adapter for comments, likes and reviews.
package social

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	domain "github.com/mokchan/webnovel-backend/internal/domain/social"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/middleware"
	socialsvc "github.com/mokchan/webnovel-backend/internal/service/social"
)

// Handler exposes the social use cases over HTTP.
type Handler struct {
	Service *socialsvc.Service
}

// New wires a handler onto a service.
func New(svc *socialsvc.Service) *Handler { return &Handler{Service: svc} }

// Register mounts the comment and review routes.
func (h *Handler) Register(r gin.IRouter, optionalAuth, requireAuth gin.HandlerFunc) {
	r.GET("/chapters/:id/comments", optionalAuth, h.listComments)
	r.POST("/chapters/:id/comments", requireAuth, h.createComment)

	r.POST("/comments/:id/like", requireAuth, h.like)
	r.DELETE("/comments/:id/like", requireAuth, h.unlike)
	r.DELETE("/comments/:id", requireAuth, h.deleteComment)

	// Reviews hang off the same /novels/:id wildcard the catalog uses; gin
	// requires the parameter name to match across the whole segment.
	r.GET("/novels/:id/reviews", optionalAuth, h.listReviews)
	r.POST("/novels/:id/reviews", requireAuth, h.upsertReview)
}

func (h *Handler) listComments(c *gin.Context) {
	chapterID, ok := httpx.IDParam(c, "id", "รหัสบทไม่ถูกต้อง")
	if !ok {
		return
	}

	comments, next, err := h.Service.ListComments(
		c.Request.Context(), chapterID, c.Query("sort"),
		middleware.ViewerID(c), httpx.ParsePage(c, 20, 100),
	)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toCommentResponses(comments), next)
}

func (h *Handler) createComment(c *gin.Context) {
	chapterID, ok := httpx.IDParam(c, "id", "รหัสบทไม่ถูกต้อง")
	if !ok {
		return
	}
	var body commentRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลความเห็นไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	created, err := h.Service.CreateComment(c.Request.Context(), domain.Comment{
		ChapterID:       chapterID,
		UserID:          p.UserID,
		ParentID:        body.ParentID,
		Body:            body.Body,
		IsSpoilerHidden: body.IsSpoilerHidden,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, toCommentResponse(*created))
}

func (h *Handler) like(c *gin.Context)   { h.toggleLike(c, true) }
func (h *Handler) unlike(c *gin.Context) { h.toggleLike(c, false) }

func (h *Handler) toggleLike(c *gin.Context, liked bool) {
	commentID, ok := httpx.IDParam(c, "id", "รหัสความเห็นไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)

	var (
		likes int
		err   error
	)
	if liked {
		likes, err = h.Service.Like(c.Request.Context(), p.UserID, commentID)
	} else {
		likes, err = h.Service.Unlike(c.Request.Context(), p.UserID, commentID)
	}
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"likes_count": likes, "liked": liked})
}

func (h *Handler) deleteComment(c *gin.Context) {
	commentID, ok := httpx.IDParam(c, "id", "รหัสความเห็นไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	err := h.Service.DeleteComment(c.Request.Context(), commentID, domain.Viewer{
		UserID: p.UserID,
		Roles:  p.Roles,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listReviews(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	reviews, next, err := h.Service.ListReviews(c.Request.Context(), novelID, httpx.ParsePage(c, 20, 100))
	if err != nil {
		h.writeErr(c, err)
		return
	}

	body := gin.H{"data": toReviewResponses(reviews), "next_cursor": next}

	// Surface the caller's own review so the form opens pre-filled.
	if viewerID := middleware.ViewerID(c); viewerID != 0 {
		mine, err := h.Service.GetUserReview(c.Request.Context(), novelID, viewerID)
		if err != nil {
			httpx.Internal(c, err)
			return
		}
		if mine != nil {
			body["my_review"] = toReviewResponse(*mine)
		}
	}
	c.JSON(http.StatusOK, body)
}

func (h *Handler) upsertReview(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	var body reviewRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลรีวิวไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	review, err := h.Service.UpsertReview(c.Request.Context(), domain.Review{
		NovelID: novelID,
		UserID:  p.UserID,
		Rating:  body.Rating,
		Body:    body.Body,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toReviewResponse(*review))
}

func (h *Handler) writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrCommentEmpty):
		httpx.BadRequest(c, "COMMENT_EMPTY", "กรุณาพิมพ์ข้อความก่อนส่ง")
	case errors.Is(err, domain.ErrCommentTooLong):
		httpx.BadRequest(c, "COMMENT_TOO_LONG", "ความเห็นยาวเกิน 5,000 ตัวอักษร")
	case errors.Is(err, domain.ErrReplyTooDeep):
		httpx.BadRequest(c, "REPLY_TOO_DEEP", "ตอบกลับได้เพียงหนึ่งระดับ")
	case errors.Is(err, domain.ErrInvalidRating):
		httpx.BadRequest(c, "INVALID_RATING", "ให้คะแนนได้ 1 ถึง 5 ดาว")
	case errors.Is(err, domain.ErrForbidden):
		httpx.Forbidden(c, "ไม่มีสิทธิ์ลบความเห็นนี้")
	case errors.Is(err, domain.ErrNotFound):
		httpx.NotFound(c, "ไม่พบรายการนี้")
	case errors.Is(err, httpx.ErrBadCursor):
		httpx.BadRequest(c, "BAD_CURSOR", "ตัวชี้หน้าถัดไปไม่ถูกต้อง")
	default:
		httpx.Internal(c, err)
	}
}
