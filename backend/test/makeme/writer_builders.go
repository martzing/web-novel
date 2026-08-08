package makeme

import (
	"time"

	"github.com/mokchan/webnovel-backend/internal/entities"
)

// ANewChapterDraft creates an autosave-revision builder. Set ChapterID and
// AuthorID via With.
func (m *MakeMe) ANewChapterDraft() *Builder[entities.ChapterDraft] {
	m.t.Helper()
	seq := m.next()
	model := &entities.ChapterDraft{
		BodySource: fixtureThaiText("ร่างบท", seq),
		SavedAt:    timeForSequence(seq),
	}
	return newBuilder(m, model, func(row *entities.ChapterDraft) map[string]any {
		return map[string]any{"id": row.ID}
	})
}

// ANewChapterGlossaryRef creates a chapter↔term link builder.
func (m *MakeMe) ANewChapterGlossaryRef() *Builder[entities.ChapterGlossaryRef] {
	m.t.Helper()
	model := &entities.ChapterGlossaryRef{}
	return newBuilder(m, model, func(row *entities.ChapterGlossaryRef) map[string]any {
		return map[string]any{"chapter_id": row.ChapterID, "entry_id": row.EntryID}
	})
}

// ANewChapterReadEvent creates a read-event builder. Set ChapterID via With.
func (m *MakeMe) ANewChapterReadEvent() *Builder[entities.ChapterReadEvent] {
	m.t.Helper()
	seq := m.next()
	model := &entities.ChapterReadEvent{OccurredAt: timeForSequence(seq)}
	return newBuilder(m, model, func(row *entities.ChapterReadEvent) map[string]any {
		return map[string]any{"id": row.ID, "occurred_at": row.OccurredAt}
	})
}

// ANewChapterDailyStat creates a per-chapter daily rollup builder.
func (m *MakeMe) ANewChapterDailyStat() *Builder[entities.ChapterDailyStat] {
	m.t.Helper()
	model := &entities.ChapterDailyStat{
		Day:           truncateToDay(time.Now()),
		Reads:         100,
		UniqueReaders: 80,
		CoinsEarned:   20,
	}
	return newBuilder(m, model, func(row *entities.ChapterDailyStat) map[string]any {
		return map[string]any{"chapter_id": row.ChapterID, "day": row.Day}
	})
}

// ANewNovelDailyStat creates a per-novel daily rollup builder.
func (m *MakeMe) ANewNovelDailyStat() *Builder[entities.NovelDailyStat] {
	m.t.Helper()
	model := &entities.NovelDailyStat{
		Day:             truncateToDay(time.Now()),
		Reads:           100,
		FollowersGained: 5,
		CoinsEarned:     20,
	}
	return newBuilder(m, model, func(row *entities.NovelDailyStat) map[string]any {
		return map[string]any{"novel_id": row.NovelID, "day": row.Day}
	})
}

// ANewRankingSnapshot creates a weekly-leaderboard row builder.
func (m *MakeMe) ANewRankingSnapshot() *Builder[entities.RankingSnapshot] {
	m.t.Helper()
	model := &entities.RankingSnapshot{
		Period: truncateToDay(time.Now()),
		Rank:   1,
		Score:  100,
	}
	return newBuilder(m, model, func(row *entities.RankingSnapshot) map[string]any {
		return map[string]any{"novel_id": row.NovelID, "period": row.Period}
	})
}

// ANewWriterProfile creates a translator-profile builder. Set UserID via With.
func (m *MakeMe) ANewWriterProfile() *Builder[entities.WriterProfile] {
	m.t.Helper()
	seq := m.next()
	model := &entities.WriterProfile{
		PenName:    fixtureThaiText("สำนักแปล", seq),
		PayoutInfo: `{}`,
	}
	return newBuilder(m, model, func(row *entities.WriterProfile) map[string]any {
		return map[string]any{"user_id": row.UserID}
	})
}

// ANewUserPrefs creates a reader-settings builder. Set UserID via With.
func (m *MakeMe) ANewUserPrefs() *Builder[entities.UserPrefs] {
	m.t.Helper()
	model := &entities.UserPrefs{
		Theme:       "light",
		Font:        "loop",
		FontSize:    20,
		LineHeight:  2.0,
		ColumnWidth: "normal",
	}
	return newBuilder(m, model, func(row *entities.UserPrefs) map[string]any {
		return map[string]any{"user_id": row.UserID}
	})
}

// ANewUserGenrePref creates an onboarding taste-weight builder.
func (m *MakeMe) ANewUserGenrePref() *Builder[entities.UserGenrePref] {
	m.t.Helper()
	model := &entities.UserGenrePref{Weight: 1}
	return newBuilder(m, model, func(row *entities.UserGenrePref) map[string]any {
		return map[string]any{"user_id": row.UserID, "genre_id": row.GenreID}
	})
}

// truncateToDay keeps DATE-typed columns from carrying a time component.
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
