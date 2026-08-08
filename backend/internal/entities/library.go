package entities

import "time"

// LibraryEntry matches `library_entries`. Composite PK — every key column must
// be tagged primaryKey or GORM looks for an auto-increment id that is not there.
type LibraryEntry struct {
	UserID  int64     `gorm:"primaryKey;column:user_id"`
	NovelID int64     `gorm:"primaryKey;column:novel_id"`
	Status  string    `gorm:"column:status;not null"`
	AddedAt time.Time `gorm:"column:added_at;autoCreateTime"`
}

func (LibraryEntry) TableName() string { return "library_entries" }

// Library statuses.
const (
	LibraryReading = "reading"
	LibrarySaved   = "saved"
	LibraryDone    = "done"
)

// ReadingProgress matches `reading_progress`.
type ReadingProgress struct {
	UserID        int64     `gorm:"primaryKey;column:user_id"`
	NovelID       int64     `gorm:"primaryKey;column:novel_id"`
	LastChapterID *int64    `gorm:"column:last_chapter_id"`
	LastChapterNo *int      `gorm:"column:last_chapter_no"`
	ParaAnchor    int       `gorm:"column:para_anchor;not null;default:0"`
	Pct           float64   `gorm:"column:pct;not null;default:0"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (ReadingProgress) TableName() string { return "reading_progress" }

// Bookmark matches `bookmarks`.
type Bookmark struct {
	ID         int64     `gorm:"primaryKey;column:id"`
	UserID     int64     `gorm:"column:user_id;not null;index"`
	NovelID    int64     `gorm:"column:novel_id;not null"`
	ChapterID  int64     `gorm:"column:chapter_id;not null"`
	ParaAnchor int       `gorm:"column:para_anchor;not null"`
	Excerpt    string    `gorm:"column:excerpt;not null"`
	Note       *string   `gorm:"column:note"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Bookmark) TableName() string { return "bookmarks" }

// Follow matches `follows`.
type Follow struct {
	UserID  int64     `gorm:"primaryKey;column:user_id"`
	NovelID int64     `gorm:"primaryKey;column:novel_id"`
	Since   time.Time `gorm:"column:since;autoCreateTime"`
}

func (Follow) TableName() string { return "follows" }
