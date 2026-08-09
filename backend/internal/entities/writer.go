package entities

import "time"

// Chapter statuses.
const (
	ChapterDraftStatus = "draft"
	ChapterScheduled   = "scheduled"
	ChapterPublished   = "published"
)

// ChapterDraft matches `chapter_drafts` — autosave history, last 20 kept per
// chapter by the application.
type ChapterDraft struct {
	ID         int64     `gorm:"primaryKey;column:id"`
	ChapterID  int64     `gorm:"column:chapter_id;not null;index"`
	AuthorID   int64     `gorm:"column:author_id;not null"`
	BodySource string    `gorm:"column:body_source;not null"`
	SavedAt    time.Time `gorm:"column:saved_at;autoCreateTime"`
}

func (ChapterDraft) TableName() string { return "chapter_drafts" }

// ChapterGlossaryRef matches `chapter_glossary_refs` — which glossary entries a
// rendered chapter actually binds.
type ChapterGlossaryRef struct {
	ChapterID int64 `gorm:"primaryKey;column:chapter_id"`
	EntryID   int64 `gorm:"primaryKey;column:entry_id"`
}

func (ChapterGlossaryRef) TableName() string { return "chapter_glossary_refs" }

// ChapterReadEvent matches the partitioned `chapter_read_events` table.
type ChapterReadEvent struct {
	ID         int64     `gorm:"primaryKey;column:id"`
	ChapterID  int64     `gorm:"column:chapter_id;not null"`
	UserID     *int64    `gorm:"column:user_id"`
	SessionID  *string   `gorm:"column:session_id;type:uuid"`
	OccurredAt time.Time `gorm:"primaryKey;column:occurred_at;autoCreateTime"`
	// Completed marks a read that reached the end of the chapter, which is what
	// the อ่านจบต่อบท KPI counts.
	Completed bool `gorm:"column:completed;not null;default:false"`
}

func (ChapterReadEvent) TableName() string { return "chapter_read_events" }

// ChapterDailyStat matches `chapter_daily_stats`.
type ChapterDailyStat struct {
	ChapterID     int64     `gorm:"primaryKey;column:chapter_id"`
	Day           time.Time `gorm:"primaryKey;column:day;type:date"`
	Reads         int       `gorm:"column:reads;not null;default:0"`
	UniqueReaders int       `gorm:"column:unique_readers;not null;default:0"`
	CoinsEarned   int       `gorm:"column:coins_earned;not null;default:0"`
	Completions   int       `gorm:"column:completions;not null;default:0"`
}

func (ChapterDailyStat) TableName() string { return "chapter_daily_stats" }

// NovelDailyStat matches `novel_daily_stats`.
type NovelDailyStat struct {
	NovelID         int64     `gorm:"primaryKey;column:novel_id"`
	Day             time.Time `gorm:"primaryKey;column:day;type:date"`
	Reads           int       `gorm:"column:reads;not null;default:0"`
	FollowersGained int       `gorm:"column:followers_gained;not null;default:0"`
	CoinsEarned     int       `gorm:"column:coins_earned;not null;default:0"`
	Completions     int       `gorm:"column:completions;not null;default:0"`
}

func (NovelDailyStat) TableName() string { return "novel_daily_stats" }

// RankingSnapshot matches `ranking_snapshots` — the weekly leaderboard history.
type RankingSnapshot struct {
	NovelID int64     `gorm:"primaryKey;column:novel_id"`
	Period  time.Time `gorm:"primaryKey;column:period;type:date"`
	Rank    int       `gorm:"column:rank;not null"`
	Score   float64   `gorm:"column:score;not null"`
}

func (RankingSnapshot) TableName() string { return "ranking_snapshots" }
