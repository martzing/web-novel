// Package writer is the HTTP adapter for the translator workspace.
package writer

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mokchan/webnovel-backend/internal/domain/roles"
	domain "github.com/mokchan/webnovel-backend/internal/domain/writer"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/middleware"
	writersvc "github.com/mokchan/webnovel-backend/internal/service/writer"
)

// Handler exposes the writer workspace over HTTP.
type Handler struct {
	Service *writersvc.Service
}

// New wires a handler onto a service.
func New(svc *writersvc.Service) *Handler { return &Handler{Service: svc} }

// Register mounts the writer routes. Every one of them requires the translator
// role, and each mutation additionally checks ownership of the specific novel.
func (h *Handler) Register(r gin.IRouter, requireAuth gin.HandlerFunc) {
	w := r.Group("/writer", requireAuth, middleware.RequireRole(roles.Translator))

	w.GET("/novels", h.listNovels)
	w.POST("/novels", h.createNovel)
	w.PATCH("/novels/:id", h.updateNovel)
	w.POST("/novels/:id/cover", h.uploadCover)

	w.GET("/series", h.listSeries)
	w.POST("/series", h.createSeries)
	w.PATCH("/series/:id", h.updateSeries)
	w.DELETE("/series/:id", h.deleteSeries)
	w.GET("/series/:id/books", h.listSeriesBooks)
	w.PUT("/series/:id/order", h.reorderSeriesBooks)

	w.PUT("/novels/:id/series-note", h.setSeriesNote)
	w.GET("/novels/:id/relations", h.listRelations)
	w.POST("/novels/:id/relations", h.createRelation)
	w.DELETE("/novels/:id/relations/:related_id", h.deleteRelation)

	w.GET("/novels/:id/arcs", h.listArcs)
	w.POST("/novels/:id/arcs", h.createArc)
	w.PATCH("/arcs/:id", h.updateArc)

	w.GET("/novels/:id/chapters", h.listChapters)
	w.POST("/novels/:id/chapters", h.createChapter)
	w.GET("/chapters/:id", h.getChapter)
	w.PUT("/chapters/:id", h.saveChapter)
	w.POST("/chapters/:id/publish", h.publishChapter)
	w.POST("/chapters/:id/unpublish", h.unpublishChapter)

	w.GET("/novels/:id/glossary", h.listGlossary)
	w.POST("/novels/:id/glossary", h.createGlossary)
	w.PATCH("/glossary-entries/:id", h.updateGlossaryEntry)
	w.DELETE("/glossary-entries/:id", h.deleteGlossaryEntry)
	w.DELETE("/glossary-groups/:id", h.deleteGlossaryGroup)

	w.GET("/stats/novels/:id", h.stats)
}

func (h *Handler) listNovels(c *gin.Context) {
	p := middleware.MustPrincipal(c)
	novels, next, err := h.Service.ListNovels(c.Request.Context(), p.UserID, httpx.ParsePage(c, 20, 100))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toNovelResponses(novels), next)
}

func (h *Handler) createNovel(c *gin.Context) {
	draft, ok := bindNovelPatch(c)
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	novel, err := h.Service.CreateNovel(c.Request.Context(), draft, p.UserID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, toNovelResponse(*novel))
}

func (h *Handler) updateNovel(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	draft, ok := bindNovelPatch(c)
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	novel, err := h.Service.UpdateNovel(c.Request.Context(), p.UserID, novelID, draft)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toNovelResponse(*novel))
}

func (h *Handler) uploadCover(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	file, err := c.FormFile("cover")
	if err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "กรุณาแนบไฟล์ภาพปก")
		return
	}
	if file.Size > writersvc.MaxCoverBytes {
		httpx.BadRequest(c, "FILE_TOO_LARGE", "ไฟล์ภาพต้องไม่เกิน 2 MB")
		return
	}

	opened, err := file.Open()
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	defer opened.Close()

	// Bound the read as well as the declared size: Size is client-reported.
	data, err := io.ReadAll(io.LimitReader(opened, writersvc.MaxCoverBytes+1))
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	p := middleware.MustPrincipal(c)
	url, err := h.Service.UploadCover(c.Request.Context(), p.UserID, novelID, data)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"cover_url": url})
}

func (h *Handler) listArcs(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	arcs, err := h.Service.ListArcs(c.Request.Context(), p.UserID, novelID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toArcResponses(arcs), "")
}

func (h *Handler) createArc(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	var body arcRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลภาคไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	arc, err := h.Service.CreateArc(c.Request.Context(), p.UserID, domain.Arc{
		NovelID:       novelID,
		ArcNo:         body.ArcNo,
		Name:          body.Name,
		FromChapterNo: body.FromChapterNo,
		ToChapterNo:   body.ToChapterNo,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, toArcResponse(*arc))
}

func (h *Handler) updateArc(c *gin.Context) {
	arcID, ok := httpx.IDParam(c, "id", "รหัสภาคไม่ถูกต้อง")
	if !ok {
		return
	}
	var body arcRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลภาคไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	arc, err := h.Service.UpdateArc(c.Request.Context(), p.UserID, arcID, domain.Arc{
		ArcNo:         body.ArcNo,
		Name:          body.Name,
		FromChapterNo: body.FromChapterNo,
		ToChapterNo:   body.ToChapterNo,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toArcResponse(*arc))
}

func (h *Handler) listChapters(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	chapters, next, err := h.Service.ListChapters(c.Request.Context(), p.UserID, novelID, httpx.ParsePage(c, 100, 500))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toChapterResponses(chapters), next)
}

func (h *Handler) createChapter(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	var body chapterRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลบทไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	chapter, err := h.Service.CreateChapter(c.Request.Context(), p.UserID, domain.Chapter{
		NovelID:    novelID,
		ChapterNo:  body.ChapterNo,
		Title:      body.Title,
		BodySource: body.BodySource,
		PriceCoins: body.PriceCoins,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, toChapterResponse(*chapter))
}

func (h *Handler) getChapter(c *gin.Context) {
	chapterID, ok := httpx.IDParam(c, "id", "รหัสบทไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	chapter, err := h.Service.GetChapter(c.Request.Context(), p.UserID, chapterID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toChapterResponse(*chapter))
}

func (h *Handler) saveChapter(c *gin.Context) {
	chapterID, ok := httpx.IDParam(c, "id", "รหัสบทไม่ถูกต้อง")
	if !ok {
		return
	}
	var body chapterRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลบทไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	chapter, err := h.Service.SaveChapter(c.Request.Context(), p.UserID, chapterID, domain.Chapter{
		ChapterNo:  body.ChapterNo,
		Title:      body.Title,
		BodySource: body.BodySource,
		PriceCoins: body.PriceCoins,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toChapterResponse(*chapter))
}

func (h *Handler) publishChapter(c *gin.Context) {
	chapterID, ok := httpx.IDParam(c, "id", "รหัสบทไม่ถูกต้อง")
	if !ok {
		return
	}
	var body publishRequest
	_ = c.ShouldBindJSON(&body)

	p := middleware.MustPrincipal(c)
	chapter, err := h.Service.PublishChapter(c.Request.Context(), p.UserID, chapterID, body.ScheduledAt)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toChapterResponse(*chapter))
}

func (h *Handler) unpublishChapter(c *gin.Context) {
	chapterID, ok := httpx.IDParam(c, "id", "รหัสบทไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	chapter, err := h.Service.UnpublishChapter(c.Request.Context(), p.UserID, chapterID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toChapterResponse(*chapter))
}

func (h *Handler) listGlossary(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	groups, err := h.Service.ListGlossary(c.Request.Context(), p.UserID, novelID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toGlossaryResponses(groups), "")
}

// createGlossary adds a group when the body names one, otherwise a term.
func (h *Handler) createGlossary(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	var group glossaryGroupRequest
	var entry glossaryEntryRequest
	if err := c.ShouldBindBodyWithJSON(&group); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลอภิธานไม่ถูกต้อง")
		return
	}
	_ = c.ShouldBindBodyWithJSON(&entry)

	p := middleware.MustPrincipal(c)

	if entry.TermKey != "" {
		created, err := h.Service.CreateGlossaryEntry(c.Request.Context(), p.UserID, novelID, domain.GlossaryEntry{
			GroupID: entry.GroupID,
			TermKey: entry.TermKey,
			TitleTH: entry.TitleTH,
			TitleCN: entry.TitleCN,
			Body:    entry.Body,
			Kind:    entry.Kind,
		})
		if err != nil {
			h.writeErr(c, err)
			return
		}
		c.JSON(http.StatusCreated, toGlossaryEntryResponse(*created))
		return
	}

	created, err := h.Service.CreateGlossaryGroup(c.Request.Context(), p.UserID, domain.GlossaryGroup{
		NovelID: novelID,
		Name:    group.Name,
		SortNo:  group.SortNo,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, GlossaryGroupResponse{
		ID:      created.ID,
		NovelID: created.NovelID,
		Name:    created.Name,
		SortNo:  created.SortNo,
		Entries: []GlossaryEntryResponse{},
	})
}

func (h *Handler) updateGlossaryEntry(c *gin.Context) {
	entryID, ok := httpx.IDParam(c, "id", "รหัสศัพท์ไม่ถูกต้อง")
	if !ok {
		return
	}
	var body glossaryEntryRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลศัพท์ไม่ถูกต้อง")
		return
	}

	p := middleware.MustPrincipal(c)
	entry, err := h.Service.UpdateGlossaryEntry(c.Request.Context(), p.UserID, entryID, domain.GlossaryEntry{
		TermKey: body.TermKey,
		TitleTH: body.TitleTH,
		TitleCN: body.TitleCN,
		Body:    body.Body,
		Kind:    body.Kind,
	})
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toGlossaryEntryResponse(*entry))
}

func (h *Handler) deleteGlossaryEntry(c *gin.Context) {
	entryID, ok := httpx.IDParam(c, "id", "รหัสศัพท์ไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	if err := h.Service.DeleteGlossaryEntry(c.Request.Context(), p.UserID, entryID); err != nil {
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) deleteGlossaryGroup(c *gin.Context) {
	groupID, ok := httpx.IDParam(c, "id", "รหัสหมวดศัพท์ไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	if err := h.Service.DeleteGlossaryGroup(c.Request.Context(), p.UserID, groupID); err != nil {
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) stats(c *gin.Context) {
	novelID, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}

	p := middleware.MustPrincipal(c)
	stats, err := h.Service.Stats(c.Request.Context(), p.UserID, novelID, c.Query("period"))
	if err != nil {
		h.writeErr(c, err)
		return
	}
	httpx.OK(c, toStatsResponse(*stats))
}

// toDomainNovel maps a patch. seriesIDSet says whether the request mentioned
// series_id at all, which is the only way to tell "leave the series alone" from
// "take this novel out of its series" — both arrive as a nil SeriesID.
func toDomainNovel(body novelRequest, seriesIDSet bool) domain.NovelDraft {
	return domain.NovelDraft{
		Slug:        body.Slug,
		TitleTH:     body.TitleTH,
		TitleCN:     body.TitleCN,
		AuthorName:  body.AuthorName,
		Description: body.Description,
		Status:      body.Status,
		SeriesID:    body.SeriesID,
		SeriesIDSet: seriesIDSet,
		GenreIDs:    body.GenreIDs,

		SourceChaptersCount: body.SourceChaptersCount,
		PricePerChapter:     body.PricePerChapter,
		FreeUntilChapter:    body.FreeUntilChapter,
		SellByArc:           body.SellByArc,
		TipsEnabled:         body.TipsEnabled,
		EarlyAccessHours:    body.EarlyAccessHours,

		ReleaseSchedule: body.ReleaseSchedule,
		CoverStyle:      body.CoverStyle,
		CoverColor:      body.CoverColor,
		CoverText:       body.CoverText,
		SeriesNote:      body.SeriesNote,
		SeriesPosition:  body.SeriesPosition,
	}
}

// bindNovelPatch decodes the body twice: once into the typed request and once
// into a map, because encoding/json cannot report which keys were present and
// the series_id patch semantics depend on exactly that.
func bindNovelPatch(c *gin.Context) (domain.NovelDraft, bool) {
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxNovelPatchBytes))
	if err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลนิยายไม่ถูกต้อง")
		return domain.NovelDraft{}, false
	}

	var body novelRequest
	if err := json.Unmarshal(raw, &body); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลนิยายไม่ถูกต้อง")
		return domain.NovelDraft{}, false
	}

	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลนิยายไม่ถูกต้อง")
		return domain.NovelDraft{}, false
	}
	_, seriesIDSet := keys["series_id"]

	return toDomainNovel(body, seriesIDSet), true
}

// maxNovelPatchBytes bounds the patch body. Descriptions are the largest field
// and are capped well below this.
const maxNovelPatchBytes = 256 << 10

func (h *Handler) writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		httpx.Forbidden(c, "ผลงานนี้ไม่ใช่ของคุณ")
	case errors.Is(err, domain.ErrNotFound):
		httpx.NotFound(c, "ไม่พบรายการนี้")
	case errors.Is(err, domain.ErrSlugTaken):
		httpx.Error(c, http.StatusConflict, "SLUG_TAKEN", "slug นี้ถูกใช้แล้ว")
	case errors.Is(err, domain.ErrChapterNoTaken):
		httpx.Error(c, http.StatusConflict, "CHAPTER_NO_TAKEN", "เลขบทนี้ถูกใช้แล้ว")
	case errors.Is(err, domain.ErrInvalidPrice):
		httpx.BadRequest(c, "INVALID_PRICE", "ราคาต้องเป็น 0 หรือมากกว่า")
	case errors.Is(err, domain.ErrUnsupportedFile):
		httpx.BadRequest(c, "UNSUPPORTED_FILE", "รองรับเฉพาะไฟล์ภาพ JPEG, PNG, WebP หรือ GIF")
	case errors.Is(err, domain.ErrFileTooLarge):
		httpx.BadRequest(c, "FILE_TOO_LARGE", "ไฟล์ภาพต้องไม่เกิน 2 MB")
	case errors.Is(err, domain.ErrGroupNotEmpty):
		httpx.Error(c, http.StatusConflict, "GROUP_NOT_EMPTY",
			"หมวดนี้ยังมีศัพท์อยู่ ย้ายหรือลบศัพท์ออกก่อน")
	case errors.Is(err, domain.ErrInvalidInput):
		httpx.BadRequest(c, "INVALID_BODY", "ข้อมูลไม่ถูกต้อง")
	case errors.Is(err, httpx.ErrBadCursor):
		httpx.BadRequest(c, "BAD_CURSOR", "ตัวชี้หน้าถัดไปไม่ถูกต้อง")
	default:
		httpx.Internal(c, err)
	}
}
