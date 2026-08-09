package writer

import (
	"net/http"

	"github.com/gin-gonic/gin"

	domain "github.com/mokchan/webnovel-backend/internal/domain/writer"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/middleware"
)

func (h *Handler) listSeries(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	list, err := h.Service.ListSeries(c.Request.Context(), p.UserID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toSeriesResponses(list), "")
}

func (h *Handler) createSeries(c *gin.Context) {
	var body seriesRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลชุดหนังสือไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	series, err := h.Service.CreateSeries(c.Request.Context(), p.UserID, domain.Series{
		Title:       body.Title,
		Description: body.Description,
		CoverURL:    body.CoverURL,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, toSeriesResponse(*series))
}

func (h *Handler) updateSeries(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสชุดหนังสือไม่ถูกต้อง")
	if !ok {
		return
	}
	var body seriesRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลชุดหนังสือไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	series, err := h.Service.UpdateSeries(c.Request.Context(), p.UserID, id, domain.Series{
		Title:       body.Title,
		Description: body.Description,
		CoverURL:    body.CoverURL,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toSeriesResponse(*series))
}

func (h *Handler) deleteSeries(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสชุดหนังสือไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	if err := h.Service.DeleteSeries(c.Request.Context(), p.UserID, id); err != nil {
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listSeriesBooks(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสชุดหนังสือไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	books, err := h.Service.SeriesBooks(c.Request.Context(), p.UserID, id)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toSeriesBookResponses(books), "")
}

func (h *Handler) reorderSeriesBooks(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสชุดหนังสือไม่ถูกต้อง")
	if !ok {
		return
	}
	var body reorderRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ลำดับการอ่านไม่ถูกต้อง")
		return
	}

	novelIDs, err := body.ids()
	if err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ลำดับการอ่านไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	books, err := h.Service.ReorderSeries(c.Request.Context(), p.UserID, id, novelIDs)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toSeriesBookResponses(books), "")
}

func (h *Handler) setSeriesNote(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	var body seriesNoteRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อความไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	if err := h.Service.SetSeriesNote(c.Request.Context(), p.UserID, novelID, body.Note); err != nil {
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listRelations(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	relations, err := h.Service.ListRelations(c.Request.Context(), p.UserID, novelID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toRelationResponses(relations), "")
}

func (h *Handler) createRelation(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	var body relationRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลเรื่องเกี่ยวเนื่องไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	rel, err := h.Service.LinkNovels(c.Request.Context(), p.UserID, domain.Relation{
		NovelID:        novelID,
		RelatedNovelID: body.RelatedNovelID,
		Kind:           body.Kind,
		Note:           body.Note,
		SortNo:         body.SortNo,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, toRelationResponses([]domain.Relation{*rel})[0])
}

func (h *Handler) deleteRelation(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	relatedID, ok := httpx.IDParam(c, "related_id", "รหัสนิยายที่เกี่ยวเนื่องไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	if err := h.Service.UnlinkNovels(c.Request.Context(), p.UserID, novelID, relatedID); err != nil {
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
