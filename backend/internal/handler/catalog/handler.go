// Package catalog is the HTTP (Gin) adapter over the catalog service.
package catalog

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	domain "github.com/mokchan/webnovel-backend/internal/domain/catalog"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	catalogsvc "github.com/mokchan/webnovel-backend/internal/service/catalog"
)

// Handler exposes catalog use cases over HTTP.
type Handler struct {
	Service *catalogsvc.Service
}

// New wires a handler onto a service.
func New(svc *catalogsvc.Service) *Handler {
	return &Handler{Service: svc}
}

// Register mounts the catalog routes on the given Gin router group.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/genres", h.listGenres)
	r.GET("/novels", h.listNovels)
	r.GET("/novels/:slug", h.getNovel)
	r.GET("/novels/:id/chapters", h.listChapters)
	r.GET("/chapters/:id", h.getChapter)
}

func (h *Handler) listGenres(c *gin.Context) {
	items, err := h.Service.ListGenres(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": toGenreResponses(items)})
}

func (h *Handler) listNovels(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	filter := domain.NovelFilter{
		Query:     c.Query("q"),
		GenreSlug: c.Query("genre"),
		Sort:      c.Query("sort"),
		Limit:     limit,
	}
	items, err := h.Service.ListNovels(c.Request.Context(), filter)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": toNovelResponses(items)})
}

func (h *Handler) getNovel(c *gin.Context) {
	slug := c.Param("slug")
	novel, err := h.Service.GetNovelBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "novel not found")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	c.JSON(http.StatusOK, toNovelDetailResponse(*novel))
}

func (h *Handler) listChapters(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_ID", "invalid novel id")
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.Service.ListChapters(c.Request.Context(), id, limit)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": toChapterListResponses(items)})
}

func (h *Handler) getChapter(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_ID", "invalid chapter id")
		return
	}
	view, err := h.Service.GetChapter(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "NOT_FOUND", "chapter not found")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	c.JSON(http.StatusOK, toChapterViewResponse(*view))
}
