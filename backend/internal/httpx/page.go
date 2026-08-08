package httpx

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
)

// ParsePage reads ?limit= and ?cursor= and clamps the limit.
func ParsePage(c *gin.Context, def, max int) page.Page {
	limit, _ := strconv.Atoi(c.Query("limit"))
	return page.Page{Limit: limit, Cursor: c.Query("cursor")}.Normalize(def, max)
}

// listBody is the paginated envelope from docs/api-spec.md.
type listBody struct {
	Data       any    `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// List writes {"data": [...], "next_cursor": "..."}. next_cursor is omitted on
// the last page.
func List(c *gin.Context, status int, data any, nextCursor string) {
	c.JSON(status, listBody{Data: data, NextCursor: nextCursor})
}

// OK writes a bare object response.
func OK(c *gin.Context, body any) {
	c.JSON(http.StatusOK, body)
}
