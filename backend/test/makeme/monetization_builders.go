package makeme

import (
	"github.com/mokchan/webnovel-backend/internal/entities"
)

// ANewSeries creates a series builder. Set OwnerUserID via With.
func (m *MakeMe) ANewSeries() *Builder[entities.Series] {
	m.t.Helper()
	seq := m.next()
	model := &entities.Series{
		Slug:  fixtureCode("series-", seq, 6),
		Title: fixtureThaiText("ชุดหนังสือ", seq),
	}
	return newBuilder(m, model, func(row *entities.Series) map[string]any {
		return map[string]any{"slug": row.Slug}
	})
}

// ANewNovelRelation creates a related-works link builder. Set NovelID and
// RelatedNovelID via With.
func (m *MakeMe) ANewNovelRelation() *Builder[entities.NovelRelation] {
	m.t.Helper()
	model := &entities.NovelRelation{Kind: entities.RelationSideStory}
	return newBuilder(m, model, func(row *entities.NovelRelation) map[string]any {
		return map[string]any{"novel_id": row.NovelID, "related_novel_id": row.RelatedNovelID}
	})
}

// ANewAutoUnlockSubscription creates an auto-unlock opt-in builder. Set UserID
// and NovelID via With.
func (m *MakeMe) ANewAutoUnlockSubscription() *Builder[entities.AutoUnlockSubscription] {
	m.t.Helper()
	model := &entities.AutoUnlockSubscription{Active: true}
	return newBuilder(m, model, func(row *entities.AutoUnlockSubscription) map[string]any {
		return map[string]any{"user_id": row.UserID, "novel_id": row.NovelID}
	})
}

// ANewAutoUnlockAttempt creates a fan-out outcome builder. Set UserID and
// ChapterID via With.
func (m *MakeMe) ANewAutoUnlockAttempt() *Builder[entities.AutoUnlockAttempt] {
	m.t.Helper()
	seq := m.next()
	model := &entities.AutoUnlockAttempt{
		Outcome:     entities.AutoUnlockInsufficient,
		Attempts:    1,
		AttemptedAt: timeForSequence(seq),
	}
	return newBuilder(m, model, func(row *entities.AutoUnlockAttempt) map[string]any {
		return map[string]any{"user_id": row.UserID, "chapter_id": row.ChapterID}
	})
}
