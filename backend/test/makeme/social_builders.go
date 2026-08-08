package makeme

import (
	"github.com/mokchan/webnovel-backend/internal/entities"
)

// ANewComment creates a comment builder. Set ChapterID and UserID via With.
func (m *MakeMe) ANewComment() *Builder[entities.Comment] {
	m.t.Helper()
	seq := m.next()
	model := &entities.Comment{
		Body:      fixtureThaiText("ความเห็น", seq),
		CreatedAt: timeForSequence(seq),
	}
	return newBuilder(m, model, func(row *entities.Comment) map[string]any {
		return map[string]any{"id": row.ID}
	})
}

// ANewCommentLike creates a like builder. Set UserID and CommentID via With.
func (m *MakeMe) ANewCommentLike() *Builder[entities.CommentLike] {
	m.t.Helper()
	seq := m.next()
	model := &entities.CommentLike{CreatedAt: timeForSequence(seq)}
	return newBuilder(m, model, func(row *entities.CommentLike) map[string]any {
		return map[string]any{"user_id": row.UserID, "comment_id": row.CommentID}
	})
}

// ANewReview creates a review builder. Set NovelID and UserID via With.
func (m *MakeMe) ANewReview() *Builder[entities.Review] {
	m.t.Helper()
	seq := m.next()
	body := fixtureThaiText("รีวิว", seq)
	model := &entities.Review{
		Rating:    5,
		Body:      &body,
		CreatedAt: timeForSequence(seq),
	}
	return newBuilder(m, model, func(row *entities.Review) map[string]any {
		return map[string]any{"novel_id": row.NovelID, "user_id": row.UserID}
	})
}

// ANewNotification creates a notification builder. Set UserID via With.
func (m *MakeMe) ANewNotification() *Builder[entities.Notification] {
	m.t.Helper()
	seq := m.next()
	model := &entities.Notification{
		Kind:      entities.NotifyNewChapter,
		Payload:   `{}`,
		CreatedAt: timeForSequence(seq),
	}
	return newBuilder(m, model, func(row *entities.Notification) map[string]any {
		return map[string]any{"id": row.ID}
	})
}
