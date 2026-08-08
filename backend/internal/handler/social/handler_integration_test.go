package social_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/domain/roles"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

type authorResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type commentResponse struct {
	ID              string            `json:"id"`
	ChapterID       string            `json:"chapter_id"`
	ParentID        string            `json:"parent_id"`
	Body            string            `json:"body"`
	IsSpoilerHidden bool              `json:"is_spoiler_hidden"`
	LikesCount      int               `json:"likes_count"`
	Liked           bool              `json:"liked"`
	IsTranslator    bool              `json:"is_translator"`
	Author          authorResponse    `json:"author"`
	Replies         []commentResponse `json:"replies"`
}

type reviewResponse struct {
	ID      string         `json:"id"`
	NovelID string         `json:"novel_id"`
	Rating  int            `json:"rating"`
	Body    string         `json:"body"`
	Author  authorResponse `json:"author"`
}

type reviewListResponse struct {
	Data     []reviewResponse `json:"data"`
	MyReview *reviewResponse  `json:"my_review"`
}

type socialFixture struct {
	env          *apitest.Env
	reader       *entities.User
	readerToken  string
	other        *entities.User
	otherToken   string
	writer       *entities.User
	writerToken  string
	novel        *entities.Novel
	chapter      *entities.Chapter
	chapterIDStr string
}

func newSocialFixture(t *testing.T) *socialFixture {
	t.Helper()
	env := apitest.New(t)
	m := env.MakeMe

	writer := env.AUser(entities.RoleTranslator)
	reader := env.AUser()
	other := env.AUser()

	novel := m.ANewNovel().With(func(n *entities.Novel) {
		n.PrimaryTranslatorID = &writer.ID
	}).Please()
	chapter := m.ANewChapter().With(func(c *entities.Chapter) {
		c.NovelID = novel.ID
		c.ChapterNo = 87
		c.Status = entities.ChapterPublished
		c.TranslatorID = &writer.ID
	}).Please()

	return &socialFixture{
		env:          env,
		reader:       reader,
		readerToken:  env.TokenFor(reader),
		other:        other,
		otherToken:   env.TokenFor(other),
		writer:       writer,
		writerToken:  env.TokenFor(writer),
		novel:        novel,
		chapter:      chapter,
		chapterIDStr: fmt.Sprint(chapter.ID),
	}
}

func (f *socialFixture) comment(t *testing.T, token, body string, parentID *string) commentResponse {
	t.Helper()
	payload := map[string]any{"body": body}
	if parentID != nil {
		payload["parent_id"] = *parentID
	}
	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/chapters/" + f.chapterIDStr + "/comments",
		Token:  token,
		Body:   payload,
	})
	apitest.AssertStatus(t, rec, http.StatusCreated)
	return apitest.DecodeJSON[commentResponse](t, rec)
}

func (f *socialFixture) list(t *testing.T, token, sort string) []commentResponse {
	t.Helper()
	path := "/api/v1/chapters/" + f.chapterIDStr + "/comments"
	if sort != "" {
		path += "?sort=" + sort
	}
	rec := f.env.Do(apitest.Request{Method: http.MethodGet, Path: path, Token: token})
	apitest.AssertStatus(t, rec, http.StatusOK)
	return apitest.DecodeJSON[apitest.List[commentResponse]](t, rec).Data
}

func TestCreateComment_AppearsInTheThread(t *testing.T) {
	f := newSocialFixture(t)

	created := f.comment(t, f.readerToken, "ฉากที่ไป๋เฉินเทชาทิ้งลงหิมะ คือทั้งหมดที่ต้องรู้", nil)
	if created.Body == "" || created.Author.ID != fmt.Sprint(f.reader.ID) {
		t.Fatalf("comment = %+v, want it attributed to the poster", created)
	}

	comments := f.list(t, "", "")
	if len(comments) != 1 || comments[0].ID != created.ID {
		t.Fatalf("thread = %+v, want the new comment", comments)
	}
}

// I-CM-01 — the limit is 5,000 characters, not bytes. A 4,000-rune Thai
// comment is legal even though it is 12,000 bytes.
func TestCreateComment_Over5000CharsReturns400(t *testing.T) {
	f := newSocialFixture(t)

	t.Run("a 4000-rune Thai comment is accepted", func(t *testing.T) {
		body := strings.Repeat("ก", 4000)
		if len(body) <= 5000 {
			t.Fatalf("test setup wrong: %d bytes should exceed 5000", len(body))
		}
		f.comment(t, f.readerToken, body, nil)
	})

	t.Run("a 5001-rune comment is rejected", func(t *testing.T) {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   "/api/v1/chapters/" + f.chapterIDStr + "/comments",
			Token:  f.readerToken,
			Body:   map[string]any{"body": strings.Repeat("ก", 5001)},
		})
		apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "COMMENT_TOO_LONG")
	})

	t.Run("an empty comment is rejected", func(t *testing.T) {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   "/api/v1/chapters/" + f.chapterIDStr + "/comments",
			Token:  f.readerToken,
			Body:   map[string]any{"body": "   "},
		})
		apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "COMMENT_EMPTY")
	})
}

// I-CM-02 — liking twice leaves the count at one.
func TestLikeComment_TwiceKeepsCountAtOne(t *testing.T) {
	f := newSocialFixture(t)
	created := f.comment(t, f.readerToken, "ความเห็นที่จะถูกกดถูกใจ", nil)

	like := func(token string) int {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   "/api/v1/comments/" + created.ID + "/like",
			Token:  token,
		})
		apitest.AssertStatus(t, rec, http.StatusOK)
		return apitest.DecodeJSON[struct {
			LikesCount int `json:"likes_count"`
		}](t, rec).LikesCount
	}

	if got := like(f.otherToken); got != 1 {
		t.Fatalf("likes = %d, want 1", got)
	}
	if got := like(f.otherToken); got != 1 {
		t.Fatalf("likes = %d after a repeat like, want it to stay at 1", got)
	}

	// A different reader does add one.
	if got := like(f.readerToken); got != 2 {
		t.Fatalf("likes = %d, want 2 after a second distinct reader", got)
	}

	// Unliking is symmetric and equally idempotent.
	unlike := func(token string) int {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodDelete,
			Path:   "/api/v1/comments/" + created.ID + "/like",
			Token:  token,
		})
		apitest.AssertStatus(t, rec, http.StatusOK)
		return apitest.DecodeJSON[struct {
			LikesCount int `json:"likes_count"`
		}](t, rec).LikesCount
	}
	if got := unlike(f.otherToken); got != 1 {
		t.Fatalf("likes = %d after unlike, want 1", got)
	}
	if got := unlike(f.otherToken); got != 1 {
		t.Fatalf("likes = %d after a repeat unlike, want it to stay at 1", got)
	}
}

func TestListComments_ReportsWhetherTheViewerLiked(t *testing.T) {
	f := newSocialFixture(t)
	created := f.comment(t, f.readerToken, "ความเห็น", nil)

	f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/comments/" + created.ID + "/like",
		Token:  f.otherToken,
	})

	if liker := f.list(t, f.otherToken, ""); !liker[0].Liked {
		t.Fatal("the reader who liked must see liked=true")
	}
	if stranger := f.list(t, f.readerToken, ""); stranger[0].Liked {
		t.Fatal("a reader who did not like must see liked=false")
	}
	if anon := f.list(t, "", ""); anon[0].Liked {
		t.Fatal("an anonymous viewer must see liked=false")
	}
}

// I-CM-03 — a translator's reply is serialized with role=translator, and the
// flag is stamped server-side rather than taken from the client.
func TestTranslatorReply_SerializedWithRoleTranslator(t *testing.T) {
	f := newSocialFixture(t)

	parent := f.comment(t, f.readerToken, "สังเกตว่าคนแต่งใช้คำว่าแบกฟืนซ้ำสามครั้ง", nil)
	reply := f.comment(t, f.writerToken, "เดาได้ดีครับ ต้นฉบับใช้คำเดียวกันทั้งสามจุด", &parent.ID)

	if !reply.IsTranslator {
		t.Fatal("the chapter's translator must be flagged is_translator")
	}
	if reply.Author.Role != roles.Translator {
		t.Fatalf("author role = %q, want %q", reply.Author.Role, roles.Translator)
	}

	// A plain reader is never flagged, even though they posted first.
	if parent.IsTranslator {
		t.Fatal("a reader must not be flagged as the translator")
	}
	if parent.Author.Role != roles.Reader {
		t.Fatalf("author role = %q, want %q", parent.Author.Role, roles.Reader)
	}

	// The reply hangs off its parent in the thread.
	thread := f.list(t, "", "")
	if len(thread) != 1 {
		t.Fatalf("thread has %d top-level comments, want 1", len(thread))
	}
	if len(thread[0].Replies) != 1 || thread[0].Replies[0].ID != reply.ID {
		t.Fatalf("replies = %+v, want the translator's reply attached", thread[0].Replies)
	}
}

// An unrelated translator gets no badge: the flag is per chapter, not a role.
func TestComment_UnrelatedTranslatorIsNotBadged(t *testing.T) {
	f := newSocialFixture(t)
	stranger := f.env.AUser(entities.RoleTranslator)

	created := f.comment(t, f.env.TokenFor(stranger), "ผมแปลเรื่องอื่น", nil)
	if created.IsTranslator {
		t.Fatal("only the chapter's own translator may be flagged")
	}
	if created.Author.Role != roles.Reader {
		t.Fatalf("author role = %q, want reader", created.Author.Role)
	}
}

func TestCreateComment_RejectsNestedReplies(t *testing.T) {
	f := newSocialFixture(t)

	parent := f.comment(t, f.readerToken, "ความเห็นหลัก", nil)
	reply := f.comment(t, f.otherToken, "ตอบกลับ", &parent.ID)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/chapters/" + f.chapterIDStr + "/comments",
		Token:  f.readerToken,
		Body:   map[string]any{"body": "ตอบกลับของตอบกลับ", "parent_id": reply.ID},
	})
	apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "REPLY_TOO_DEEP")
}

func TestListComments_SortOrders(t *testing.T) {
	f := newSocialFixture(t)

	first := f.comment(t, f.readerToken, "ความเห็นแรก", nil)
	second := f.comment(t, f.otherToken, "ความเห็นที่สอง", nil)

	// Give the first comment a like so it outranks the second by popularity.
	f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/comments/" + first.ID + "/like",
		Token:  f.otherToken,
	})

	t.Run("popular puts the most-liked first", func(t *testing.T) {
		got := f.list(t, "", "popular")
		if len(got) != 2 || got[0].ID != first.ID {
			t.Fatalf("order = %+v, want the liked comment first", got)
		}
	})

	t.Run("latest puts the newest first", func(t *testing.T) {
		got := f.list(t, "", "latest")
		if len(got) != 2 || got[0].ID != second.ID {
			t.Fatalf("order = %+v, want the newest comment first", got)
		}
	})

	t.Run("with_replies keeps only threads that have replies", func(t *testing.T) {
		f.comment(t, f.writerToken, "ตอบกลับความเห็นที่สอง", &second.ID)

		got := f.list(t, "", "with_replies")
		if len(got) != 1 || got[0].ID != second.ID {
			t.Fatalf("filtered = %+v, want only the replied-to comment", got)
		}
	})
}

func TestDeleteComment_Permissions(t *testing.T) {
	f := newSocialFixture(t)

	del := func(token, id string) int {
		return f.env.Do(apitest.Request{
			Method: http.MethodDelete,
			Path:   "/api/v1/comments/" + id,
			Token:  token,
		}).Code
	}

	t.Run("a stranger cannot delete", func(t *testing.T) {
		created := f.comment(t, f.readerToken, "ของฉัน", nil)
		if code := del(f.otherToken, created.ID); code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", code)
		}
	})

	t.Run("the author can delete", func(t *testing.T) {
		created := f.comment(t, f.readerToken, "ของฉัน", nil)
		if code := del(f.readerToken, created.ID); code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", code)
		}
	})

	t.Run("the chapter's translator can moderate", func(t *testing.T) {
		created := f.comment(t, f.readerToken, "สแปม", nil)
		if code := del(f.writerToken, created.ID); code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", code)
		}
	})

	t.Run("an admin can moderate", func(t *testing.T) {
		created := f.comment(t, f.readerToken, "สแปม", nil)
		admin := f.env.AUser(entities.RoleAdmin)
		if code := del(f.env.TokenFor(admin), created.ID); code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", code)
		}
	})
}

// Deletion is a soft delete: the row survives for audit but leaves the thread.
func TestDeleteComment_SoftDeletesAndHidesFromTheThread(t *testing.T) {
	f := newSocialFixture(t)
	created := f.comment(t, f.readerToken, "จะถูกลบ", nil)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   "/api/v1/comments/" + created.ID,
		Token:  f.readerToken,
	})
	apitest.AssertStatus(t, rec, http.StatusNoContent)

	if got := f.list(t, "", ""); len(got) != 0 {
		t.Fatalf("thread = %+v, want the deleted comment hidden", got)
	}

	var row entities.Comment
	if err := f.env.MakeMe.DB.Where("id = ?", created.ID).Take(&row).Error; err != nil {
		t.Fatalf("the row must survive for audit: %v", err)
	}
	if row.DeletedAt == nil {
		t.Fatal("expected deleted_at to be stamped")
	}
}

func TestCommentRoutes_RequireAuthToWrite(t *testing.T) {
	f := newSocialFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/chapters/" + f.chapterIDStr + "/comments",
		Body:   map[string]any{"body": "anonymous"},
	})
	apitest.AssertStatus(t, rec, http.StatusUnauthorized)

	// Reading the thread stays public.
	apitest.AssertStatus(t, f.env.GET("/api/v1/chapters/"+f.chapterIDStr+"/comments"), http.StatusOK)
}

// R-19 — one review per reader per novel, and it drives the rating aggregate.
func TestReviews_OnePerUserAndUpdateTheNovelAggregate(t *testing.T) {
	f := newSocialFixture(t)
	novelID := fmt.Sprint(f.novel.ID)

	review := func(token string, rating int, body string) reviewResponse {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   "/api/v1/novels/" + novelID + "/reviews",
			Token:  token,
			Body:   map[string]any{"rating": rating, "body": body},
		})
		apitest.AssertStatus(t, rec, http.StatusOK)
		return apitest.DecodeJSON[reviewResponse](t, rec)
	}

	aggregate := func() (float64, int) {
		var row entities.Novel
		if err := f.env.MakeMe.DB.Where("id = ?", f.novel.ID).Take(&row).Error; err != nil {
			t.Fatalf("load novel: %v", err)
		}
		return row.RatingAvg, row.RatingCount
	}

	review(f.readerToken, 5, "สนุกมาก")
	if avg, count := aggregate(); avg != 5 || count != 1 {
		t.Fatalf("aggregate = %v/%d, want 5/1", avg, count)
	}

	review(f.otherToken, 3, "พอใช้")
	if avg, count := aggregate(); avg != 4 || count != 2 {
		t.Fatalf("aggregate = %v/%d, want 4/2", avg, count)
	}

	// Re-reviewing replaces rather than adds.
	review(f.readerToken, 1, "เปลี่ยนใจ")
	if avg, count := aggregate(); avg != 2 || count != 2 {
		t.Fatalf("aggregate = %v/%d, want 2/2 after an update", avg, count)
	}

	var rows int64
	if err := f.env.MakeMe.DB.Model(&entities.Review{}).
		Where("novel_id = ?", f.novel.ID).Count(&rows).Error; err != nil {
		t.Fatalf("count reviews: %v", err)
	}
	if rows != 2 {
		t.Fatalf("review rows = %d, want 2", rows)
	}
}

func TestReviews_RejectOutOfRangeRatings(t *testing.T) {
	f := newSocialFixture(t)
	novelID := fmt.Sprint(f.novel.ID)

	for _, rating := range []int{0, -1, 6, 99} {
		rec := f.env.Do(apitest.Request{
			Method: http.MethodPost,
			Path:   "/api/v1/novels/" + novelID + "/reviews",
			Token:  f.readerToken,
			Body:   map[string]any{"rating": rating},
		})
		apitest.AssertErrorCode(t, rec, http.StatusBadRequest, "INVALID_RATING")
	}
}

func TestReviews_ListIncludesTheCallersOwnReview(t *testing.T) {
	f := newSocialFixture(t)
	novelID := fmt.Sprint(f.novel.ID)

	f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/novels/" + novelID + "/reviews",
		Token:  f.readerToken,
		Body:   map[string]any{"rating": 4, "body": "ดี"},
	})

	rec := f.env.GETAuth("/api/v1/novels/"+novelID+"/reviews", f.readerToken)
	apitest.AssertStatus(t, rec, http.StatusOK)

	body := apitest.DecodeJSON[reviewListResponse](t, rec)
	if len(body.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(body.Data))
	}
	if body.MyReview == nil || body.MyReview.Rating != 4 {
		t.Fatalf("my_review = %+v, want the caller's 4-star review", body.MyReview)
	}

	// Another reader has no review of their own to pre-fill.
	rec = f.env.GETAuth("/api/v1/novels/"+novelID+"/reviews", f.otherToken)
	if apitest.DecodeJSON[reviewListResponse](t, rec).MyReview != nil {
		t.Fatal("my_review must be absent for a reader who has not reviewed")
	}
}
