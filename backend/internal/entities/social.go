package entities

import "time"

// Comment matches `comments`. parent_id supports one level of replies;
// is_translator is stamped server-side at insert time.
type Comment struct {
	ID              int64      `gorm:"primaryKey;column:id"`
	ChapterID       int64      `gorm:"column:chapter_id;not null;index"`
	UserID          int64      `gorm:"column:user_id;not null"`
	ParentID        *int64     `gorm:"column:parent_id"`
	Body            string     `gorm:"column:body;not null"`
	IsSpoilerHidden bool       `gorm:"column:is_spoiler_hidden;not null;default:false"`
	LikesCount      int        `gorm:"column:likes_count;not null;default:0"`
	IsTranslator    bool       `gorm:"column:is_translator;not null;default:false"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime"`
	DeletedAt       *time.Time `gorm:"column:deleted_at"`
}

func (Comment) TableName() string { return "comments" }

// CommentLike matches `comment_likes`.
type CommentLike struct {
	UserID    int64     `gorm:"primaryKey;column:user_id"`
	CommentID int64     `gorm:"primaryKey;column:comment_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (CommentLike) TableName() string { return "comment_likes" }

// Review matches `reviews`. UNIQUE (novel_id, user_id) enforces one per user.
type Review struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	NovelID   int64     `gorm:"column:novel_id;not null;index"`
	UserID    int64     `gorm:"column:user_id;not null"`
	Rating    int16     `gorm:"column:rating;not null"`
	Body      *string   `gorm:"column:body"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Review) TableName() string { return "reviews" }

// Notification matches `notifications`. Payload is kind-specific jsonb.
type Notification struct {
	ID        int64      `gorm:"primaryKey;column:id"`
	UserID    int64      `gorm:"column:user_id;not null;index"`
	Kind      string     `gorm:"column:kind;not null"`
	Payload   string     `gorm:"column:payload;type:jsonb;not null"`
	ReadAt    *time.Time `gorm:"column:read_at"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (Notification) TableName() string { return "notifications" }

// Notification kinds.
const (
	NotifyNewChapter    = "new_chapter"
	NotifyReply         = "reply"
	NotifyBonusExpiring = "bonus_expiring"
)
