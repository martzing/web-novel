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

// Series matches the `series` table — a collection of related novels
// (ชุดหนังสือ) with a translator-curated reading order.
type Series struct {
	ID          int64     `gorm:"primaryKey;column:id"             json:"id,string"`
	Slug        string    `gorm:"column:slug;uniqueIndex;not null" json:"slug"`
	Title       string    `gorm:"column:title;not null"            json:"title"`
	Description *string   `gorm:"column:description"               json:"description,omitempty"`
	CoverURL    *string   `gorm:"column:cover_url"                 json:"cover_url,omitempty"`
	OwnerUserID *int64    `gorm:"column:owner_user_id"             json:"owner_user_id,string,omitempty"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Series) TableName() string { return "series" }

// Relation kinds between two novels, matching the CHECK on
// novel_relations.kind.
const (
	RelationSequel    = "sequel"     // ภาคต่อโดยตรง
	RelationPrequel   = "prequel"    // ปฐมบท
	RelationSpinoff   = "spinoff"    // ภาคแยก
	RelationSideStory = "side_story" // ภาคพิเศษ
	RelationSameWorld = "same_world" // เกิดในโลกเดียวกัน
)

// NovelRelation matches `novel_relations` — เรื่องเกี่ยวเนื่อง.
//
// Stored directional: the kind is stated from NovelID's point of view.
// RelationSameWorld is the one symmetric kind and is mirrored when read.
type NovelRelation struct {
	NovelID        int64     `gorm:"primaryKey;column:novel_id"`
	RelatedNovelID int64     `gorm:"primaryKey;column:related_novel_id"`
	Kind           string    `gorm:"column:kind;not null"`
	Note           *string   `gorm:"column:note"`
	SortNo         int16     `gorm:"column:sort_no;not null;default:0"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (NovelRelation) TableName() string { return "novel_relations" }

// Novel publication statuses, matching the CHECK on novels.status.
const (
	NovelOngoing  = "ongoing"
	NovelComplete = "complete"
	NovelHiatus   = "hiatus"
	// NovelHidden is ซ่อนจากหน้าร้าน: still editable by its translator, but
	// absent from every reader-facing list, search result and ranking.
	NovelHidden = "hidden"
)

// Cover styles. CoverImage means "use cover_url"; the rest are generated from
// cover_color and cover_text.
const (
	CoverImage = "image"
	CoverInk   = "ink"
	CoverSeal  = "seal"
	CoverBrush = "brush"
	CoverPlain = "plain"
)

// Novel matches the `novels` table.
type Novel struct {
	ID                  int64   `gorm:"primaryKey;column:id"                       json:"id,string"`
	SeriesID            *int64  `gorm:"column:series_id"                           json:"series_id,string,omitempty"`
	Slug                string  `gorm:"column:slug;uniqueIndex;not null"           json:"slug"`
	TitleTH             string  `gorm:"column:title_th;not null"                   json:"title_th"`
	TitleCN             *string `gorm:"column:title_cn"                            json:"title_cn,omitempty"`
	AuthorName          *string `gorm:"column:author_name"                         json:"author_name,omitempty"`
	Description         *string `gorm:"column:description"                         json:"description,omitempty"`
	CoverURL            *string `gorm:"column:cover_url"                           json:"cover_url,omitempty"`
	Status              string  `gorm:"column:status;not null;default:ongoing"     json:"status"`
	PrimaryTranslatorID *int64  `gorm:"column:primary_translator_id"               json:"primary_translator_id,string,omitempty"`
	RatingAvg           float64 `gorm:"column:rating_avg;not null;default:0"       json:"rating_avg"`
	RatingCount         int     `gorm:"column:rating_count;not null;default:0"     json:"rating_count"`
	FollowersCount      int     `gorm:"column:followers_count;not null;default:0"  json:"followers_count"`
	ChaptersCount       int     `gorm:"column:chapters_count;not null;default:0"   json:"chapters_count"`
	GlossaryRev         int     `gorm:"column:glossary_rev;not null;default:0"     json:"glossary_rev"`

	// SourceChaptersCount is how many chapters the original work has, as
	// opposed to ChaptersCount which counts what has been translated and
	// published. Every "บทในต้นฉบับ" figure reads this.
	SourceChaptersCount int `gorm:"column:source_chapters_count;not null;default:0" json:"source_chapters_count"`

	// Monetisation settings, all writer-controlled.
	PricePerChapter  int16  `gorm:"column:price_per_chapter;not null;default:0"  json:"price_per_chapter"`
	FreeUntilChapter int    `gorm:"column:free_until_chapter;not null;default:0" json:"free_until_chapter"`
	SellByArc        bool   `gorm:"column:sell_by_arc;not null;default:false"    json:"sell_by_arc"`
	TipsEnabled      bool   `gorm:"column:tips_enabled;not null;default:false"   json:"tips_enabled"`
	EarlyAccessHours int16  `gorm:"column:early_access_hours;not null;default:0" json:"early_access_hours"`
	ReleaseSchedule  string `gorm:"column:release_schedule"                      json:"release_schedule,omitempty"`

	// Cover: either an uploaded image (CoverURL) or a generated template.
	CoverStyle string  `gorm:"column:cover_style;not null;default:image" json:"cover_style"`
	CoverColor *string `gorm:"column:cover_color"                        json:"cover_color,omitempty"`
	CoverText  *string `gorm:"column:cover_text"                         json:"cover_text,omitempty"`

	// Placement within SeriesID. A novel belongs to at most one series, so the
	// reading-order slot and its note live here rather than in a join table.
	SeriesPosition *int16  `gorm:"column:series_position" json:"series_position,omitempty"`
	SeriesNote     *string `gorm:"column:series_note"     json:"series_note,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

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

	// PublicAt is when a published chapter becomes visible to readers who are
	// not auto-unlock subscribers. It is snapshotted at publish time as
	// published_at + novels.early_access_hours, never derived at read time.
	// nil means "immediately".
	PublicAt *time.Time `gorm:"column:public_at" json:"public_at,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
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
