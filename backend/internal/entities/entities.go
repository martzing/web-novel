// Package entities holds the GORM persistence models. Only repository adapters
// import it; domain packages never see these types.
package entities

import (
	"slices"
	"time"

	"github.com/lib/pq"
)

// Genre matches the `genres` table.
type Genre struct {
	ID     int64  `gorm:"primaryKey;column:id"                 json:"id,string"`
	Slug   string `gorm:"column:slug;uniqueIndex;not null"     json:"slug"`
	NameTH string `gorm:"column:name_th;not null"              json:"name_th"`
}

func (Genre) TableName() string { return "genres" }

// Series matches the `series` table.
type Series struct {
	ID          int64     `gorm:"primaryKey;column:id"          json:"id,string"`
	Title       string    `gorm:"column:title;not null"         json:"title"`
	Description *string   `gorm:"column:description"            json:"description,omitempty"`
	CoverURL    *string   `gorm:"column:cover_url"              json:"cover_url,omitempty"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Series) TableName() string { return "series" }

// Novel matches the `novels` table.
type Novel struct {
	ID                  int64     `gorm:"primaryKey;column:id"                       json:"id,string"`
	SeriesID            *int64    `gorm:"column:series_id"                           json:"series_id,string,omitempty"`
	Slug                string    `gorm:"column:slug;uniqueIndex;not null"           json:"slug"`
	TitleTH             string    `gorm:"column:title_th;not null"                   json:"title_th"`
	TitleCN             *string   `gorm:"column:title_cn"                            json:"title_cn,omitempty"`
	AuthorName          *string   `gorm:"column:author_name"                         json:"author_name,omitempty"`
	Description         *string   `gorm:"column:description"                         json:"description,omitempty"`
	CoverURL            *string   `gorm:"column:cover_url"                           json:"cover_url,omitempty"`
	Status              string    `gorm:"column:status;not null;default:ongoing"     json:"status"`
	PrimaryTranslatorID *int64    `gorm:"column:primary_translator_id"               json:"primary_translator_id,string,omitempty"`
	RatingAvg           float64   `gorm:"column:rating_avg;not null;default:0"       json:"rating_avg"`
	RatingCount         int       `gorm:"column:rating_count;not null;default:0"     json:"rating_count"`
	FollowersCount      int       `gorm:"column:followers_count;not null;default:0"  json:"followers_count"`
	ChaptersCount       int       `gorm:"column:chapters_count;not null;default:0"   json:"chapters_count"`
	GlossaryRev         int       `gorm:"column:glossary_rev;not null;default:0"     json:"glossary_rev"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime"           json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime"           json:"updated_at"`

	Genres []Genre `gorm:"many2many:novel_genres;joinForeignKey:novel_id;joinReferences:genre_id" json:"genres,omitempty"`
	Arcs   []Arc   `gorm:"foreignKey:NovelID"                                                     json:"arcs,omitempty"`
}

func (Novel) TableName() string { return "novels" }

// NovelGenre matches the `novel_genres` join table.
type NovelGenre struct {
	NovelID int64 `gorm:"primaryKey;column:novel_id"`
	GenreID int64 `gorm:"primaryKey;column:genre_id"`
}

func (NovelGenre) TableName() string { return "novel_genres" }

// Arc matches the `arcs` table.
type Arc struct {
	ID            int64  `gorm:"primaryKey;column:id"                           json:"id,string"`
	NovelID       int64  `gorm:"column:novel_id;not null;index"                 json:"novel_id,string"`
	ArcNo         int16  `gorm:"column:arc_no;not null"                         json:"arc_no"`
	Name          string `gorm:"column:name;not null"                           json:"name"`
	FromChapterNo int    `gorm:"column:from_chapter_no;not null"                json:"from_chapter_no"`
	ToChapterNo   int    `gorm:"column:to_chapter_no;not null"                  json:"to_chapter_no"`
}

func (Arc) TableName() string { return "arcs" }

// Chapter matches the `chapters` table.
type Chapter struct {
	ID           int64      `gorm:"primaryKey;column:id"                    json:"id,string"`
	NovelID      int64      `gorm:"column:novel_id;not null;index"          json:"novel_id,string"`
	ArcID        *int64     `gorm:"column:arc_id"                           json:"arc_id,string,omitempty"`
	ChapterNo    int        `gorm:"column:chapter_no;not null"              json:"chapter_no"`
	Title        string     `gorm:"column:title;not null"                   json:"title"`
	Status       string     `gorm:"column:status;not null;default:draft"    json:"status"`
	PriceCoins   int16      `gorm:"column:price_coins;not null;default:0"   json:"price_coins"`
	WordCount    int        `gorm:"column:word_count;not null;default:0"    json:"word_count"`
	TranslatorID *int64     `gorm:"column:translator_id"                    json:"translator_id,string,omitempty"`
	PublishedAt  *time.Time `gorm:"column:published_at"                     json:"published_at,omitempty"`
	ScheduledAt  *time.Time `gorm:"column:scheduled_at"                     json:"scheduled_at,omitempty"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"        json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"        json:"updated_at"`
}

func (Chapter) TableName() string { return "chapters" }

// ChapterBody matches `chapter_bodies`. The DDL stores rendered HTML separately from chapter metadata.
type ChapterBody struct {
	ChapterID   int64     `gorm:"primaryKey;column:chapter_id"           json:"chapter_id,string"`
	BodyHTML    string    `gorm:"column:body_html;not null"              json:"body_html"`
	BodySource  string    `gorm:"column:body_source;not null"            json:"body_source"`
	Revision    int       `gorm:"column:revision;not null;default:1"     json:"revision"`
	GlossaryRev int       `gorm:"column:glossary_rev;not null;default:0" json:"glossary_rev"`
	RenderedAt  time.Time `gorm:"column:rendered_at;autoCreateTime"      json:"rendered_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"       json:"updated_at"`
}

func (ChapterBody) TableName() string { return "chapter_bodies" }

// GlossaryGroup matches `glossary_groups`.
type GlossaryGroup struct {
	ID      int64  `gorm:"primaryKey;column:id"                json:"id,string"`
	NovelID int64  `gorm:"column:novel_id;not null;index"      json:"novel_id,string"`
	Name    string `gorm:"column:name;not null"                json:"name"`
	SortNo  int16  `gorm:"column:sort_no;not null;default:0"   json:"sort_no"`
}

func (GlossaryGroup) TableName() string { return "glossary_groups" }

// GlossaryEntry matches `glossary_entries`.
type GlossaryEntry struct {
	ID      int64   `gorm:"primaryKey;column:id"        json:"id,string"`
	GroupID int64   `gorm:"column:group_id;not null;index" json:"group_id,string"`
	TermKey string  `gorm:"column:term_key;not null"    json:"term_key"`
	TitleTH string  `gorm:"column:title_th;not null"    json:"title_th"`
	TitleCN *string `gorm:"column:title_cn"             json:"title_cn,omitempty"`
	Body    string  `gorm:"column:body;not null"        json:"body"`
	Kind    *string `gorm:"column:kind"                 json:"kind,omitempty"`
}

func (GlossaryEntry) TableName() string { return "glossary_entries" }

// Role names stored in users.roles.
const (
	RoleReader     = "reader"
	RoleTranslator = "translator"
	RoleAdmin      = "admin"
)

// User matches the `users` table.
//
// Roles maps the `text[]` column through pq.StringArray. It must always be set
// on insert: once the field exists GORM sends it on every INSERT, so leaving it
// nil writes NULL and violates the NOT NULL constraint rather than falling back
// to the column's ARRAY['reader'] default.
type User struct {
	ID           int64          `gorm:"primaryKey;column:id"                     json:"id,string"`
	Username     string         `gorm:"column:username;uniqueIndex;not null"     json:"username"`
	Email        string         `gorm:"column:email;uniqueIndex;not null"        json:"email"`
	PasswordHash string         `gorm:"column:password_hash;not null"            json:"-"`
	DisplayName  string         `gorm:"column:display_name;not null"             json:"display_name"`
	AvatarURL    *string        `gorm:"column:avatar_url"                        json:"avatar_url,omitempty"`
	Roles        pq.StringArray `gorm:"column:roles;type:text[];not null"        json:"roles"`
	Status       string         `gorm:"column:status;not null;default:active"    json:"status"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime"         json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime"         json:"updated_at"`
}

func (User) TableName() string { return "users" }

// HasRole reports whether the user carries the given role.
func (u User) HasRole(role string) bool {
	return slices.Contains(u.Roles, role)
}

// WriterProfile matches `writer_profiles`.
type WriterProfile struct {
	UserID     int64     `gorm:"primaryKey;column:user_id"          json:"user_id,string"`
	PenName    string    `gorm:"column:pen_name;not null"           json:"pen_name"`
	Bio        *string   `gorm:"column:bio"                         json:"bio,omitempty"`
	SectName   *string   `gorm:"column:sect_name"                   json:"sect_name,omitempty"`
	PayoutInfo string    `gorm:"column:payout_info;type:jsonb;default:'{}'" json:"payout_info"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"   json:"updated_at"`
}

func (WriterProfile) TableName() string { return "writer_profiles" }

// UserPrefs matches `user_prefs` — reader settings synced across devices.
type UserPrefs struct {
	UserID      int64     `gorm:"primaryKey;column:user_id"                json:"user_id,string"`
	Theme       string    `gorm:"column:theme;not null;default:light"      json:"theme"`
	Font        string    `gorm:"column:font;not null;default:loop"        json:"font"`
	FontSize    int16     `gorm:"column:font_size;not null;default:20"     json:"font_size"`
	LineHeight  float64   `gorm:"column:line_height;not null;default:2.0"  json:"line_height"`
	ColumnWidth string    `gorm:"column:column_width;not null;default:normal" json:"column_width"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"         json:"updated_at"`
}

func (UserPrefs) TableName() string { return "user_prefs" }

// UserGenrePref matches `user_genre_prefs` — onboarding taste weights.
type UserGenrePref struct {
	UserID  int64 `gorm:"primaryKey;column:user_id"          json:"user_id,string"`
	GenreID int64 `gorm:"primaryKey;column:genre_id"         json:"genre_id,string"`
	Weight  int16 `gorm:"column:weight;not null;default:1"   json:"weight"`
}

func (UserGenrePref) TableName() string { return "user_genre_prefs" }

// RefreshToken matches `refresh_tokens` (migration 0003). Only the SHA-256 of
// the token is stored, so a database dump grants no sessions.
type RefreshToken struct {
	ID         int64      `gorm:"primaryKey;column:id"`
	UserID     int64      `gorm:"column:user_id;not null;index"`
	FamilyID   string     `gorm:"column:family_id;not null;index"`
	TokenHash  []byte     `gorm:"column:token_hash;not null;uniqueIndex"`
	UserAgent  *string    `gorm:"column:user_agent"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
	ReplacedBy *int64     `gorm:"column:replaced_by"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
