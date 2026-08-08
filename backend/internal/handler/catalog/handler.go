// Package catalog is the HTTP (Gin) adapter over the catalog service.
package catalog

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	domain "github.com/mokchan/webnovel-backend/internal/domain/catalog"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/middleware"
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

// Register mounts the catalog routes.
//
// Every `/novels/...` route uses the same `:id` wildcard name. Gin panics at
// startup when two different names occupy the same path segment, so id-or-slug
// discrimination happens inside the service instead.
func (h *Handler) Register(r gin.IRouter, optionalAuth gin.HandlerFunc) {
	r.GET("/genres", h.listGenres)
	r.GET("/novels", h.listNovels)
	r.GET("/novels/:id", h.getNovel)
	r.GET("/novels/:id/chapters", optionalAuth, h.listChapters)
	r.GET("/novels/:id/arcs", h.listArcs)
	r.GET("/novels/:id/glossary", h.getGlossary)
	r.GET("/search", h.search)
	r.GET("/ranking/weekly", h.weeklyRanking)
}

func (h *Handler) listGenres(c *gin.Context) {
	items, err := h.Service.ListGenres(c.Request.Context())
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toGenreResponses(items), "")
}

func (h *Handler) listNovels(c *gin.Context) {
	h.respondNovelList(c, c.Query("q"))
}

// search is the discovery-utility alias of listNovels; the blended
// trigram/FTS ranking lives in the repository.
func (h *Handler) search(c *gin.Context) {
	h.respondNovelList(c, c.Query("q"))
}

func (h *Handler) respondNovelList(c *gin.Context, query string) {
	p := httpx.ParsePage(c, 20, 100)

	sort := c.Query("sort")
	if sort != domain.SortLatest {
		sort = domain.SortPopular
	}

	cursor, err := httpx.DecodeCursorFor(p.Cursor, sort)
	if err != nil {
		httpx.BadRequest(c, "BAD_CURSOR", "ตัวชี้หน้าถัดไปไม่ถูกต้อง")
		return
	}

	filter := domain.NovelFilter{
		Query:     query,
		GenreSlug: c.Query("genre"),
		Sort:      sort,
		Limit:     p.Limit,
	}
	if len(cursor.Keys) == 2 {
		filter.AfterValue = cursor.Keys[0]
		filter.AfterID, _ = strconv.ParseInt(cursor.Keys[1], 10, 64)
	}

	items, hasMore, err := h.Service.ListNovels(c.Request.Context(), filter)
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	next := ""
	// Search results are ranked by relevance, which has no stable keyset, so
	// they are returned as a single page.
	if hasMore && len(items) > 0 && query == "" {
		last := items[len(items)-1]
		key := strconv.Itoa(last.FollowersCount)
		if sort == domain.SortLatest {
			key = last.UpdatedAt
		}
		next = httpx.EncodeCursor(httpx.Cursor{
			Sort: sort,
			Keys: []string{key, strconv.FormatInt(last.ID, 10)},
		})
	}
	httpx.List(c, http.StatusOK, toNovelResponses(items), next)
}

func (h *Handler) getNovel(c *gin.Context) {
	novel, err := h.Service.GetNovel(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeErr(c, err, "ไม่พบนิยายเรื่องนี้")
		return
	}
	httpx.OK(c, toNovelDetailResponse(*novel))
}

func (h *Handler) listArcs(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	items, err := h.Service.ListArcs(c.Request.Context(), id)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toArcResponses(items), "")
}

func (h *Handler) getGlossary(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	groups, err := h.Service.GetGlossary(c.Request.Context(), id)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toGlossaryResponses(groups), "")
}

func (h *Handler) listChapters(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id", "รหัสนิยายไม่ถูกต้อง")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.Service.ListChapters(c.Request.Context(), id, limit, middleware.ViewerID(c))
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toChapterListResponses(items), "")
}

func (h *Handler) weeklyRanking(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.Service.WeeklyRanking(c.Request.Context(), limit)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.List(c, http.StatusOK, toRankedNovelResponses(items), "")
}

func (h *Handler) writeErr(c *gin.Context, err error, notFoundMsg string) {
	if errors.Is(err, domain.ErrNotFound) {
		httpx.NotFound(c, notFoundMsg)
		return
	}
	httpx.Internal(c, err)
}
