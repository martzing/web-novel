package entities

import "time"

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

// User matches the `users` table (subset needed for Phase 1 fixture wiring).
// The `roles` text[] column is intentionally omitted so GORM lets Postgres apply its default.
type User struct {
	ID           int64     `gorm:"primaryKey;column:id"                   json:"id,string"`
	Username     string    `gorm:"column:username;uniqueIndex;not null"   json:"username"`
	Email        string    `gorm:"column:email;uniqueIndex;not null"      json:"email"`
	PasswordHash string    `gorm:"column:password_hash;not null"          json:"-"`
	DisplayName  string    `gorm:"column:display_name;not null"           json:"display_name"`
	AvatarURL    *string   `gorm:"column:avatar_url"                      json:"avatar_url,omitempty"`
	Status       string    `gorm:"column:status;not null;default:active"  json:"status"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"       json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"       json:"updated_at"`
}

func (User) TableName() string { return "users" }
