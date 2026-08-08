package makeme

import (
	"github.com/mokchan/webnovel-backend/internal/entities"
)

// ANewLibraryEntry creates a shelf-entry builder. Set UserID and NovelID via With.
func (m *MakeMe) ANewLibraryEntry() *Builder[entities.LibraryEntry] {
	m.t.Helper()
	seq := m.next()
	model := &entities.LibraryEntry{
		Status:  entities.LibraryReading,
		AddedAt: timeForSequence(seq),
	}
	return newBuilder(m, model, func(row *entities.LibraryEntry) map[string]any {
		return map[string]any{"user_id": row.UserID, "novel_id": row.NovelID}
	})
}

// ANewReadingProgress creates a progress builder. Set UserID and NovelID via With.
func (m *MakeMe) ANewReadingProgress() *Builder[entities.ReadingProgress] {
	m.t.Helper()
	seq := m.next()
	model := &entities.ReadingProgress{
		ParaAnchor: 0,
		Pct:        0,
		UpdatedAt:  timeForSequence(seq),
	}
	return newBuilder(m, model, func(row *entities.ReadingProgress) map[string]any {
		return map[string]any{"user_id": row.UserID, "novel_id": row.NovelID}
	})
}

// ANewBookmark creates a bookmark builder. Set UserID, NovelID and ChapterID via With.
func (m *MakeMe) ANewBookmark() *Builder[entities.Bookmark] {
	m.t.Helper()
	seq := m.next()
	model := &entities.Bookmark{
		ParaAnchor: int(seq % 50),
		Excerpt:    fixtureThaiText("ข้อความที่คั่น", seq),
		CreatedAt:  timeForSequence(seq),
	}
	return newBuilder(m, model, func(row *entities.Bookmark) map[string]any {
		return map[string]any{"id": row.ID}
	})
}

// ANewFollow creates a follow builder. Set UserID and NovelID via With.
func (m *MakeMe) ANewFollow() *Builder[entities.Follow] {
	m.t.Helper()
	seq := m.next()
	model := &entities.Follow{Since: timeForSequence(seq)}
	return newBuilder(m, model, func(row *entities.Follow) map[string]any {
		return map[string]any{"user_id": row.UserID, "novel_id": row.NovelID}
	})
}
