package makeme

import (
	"sync"

	"github.com/lib/pq"

	"github.com/mokchan/webnovel-backend/internal/crypto/argon2id"
	"github.com/mokchan/webnovel-backend/internal/entities"
)

// DefaultPassword is the plaintext behind every fixture user's password hash,
// so auth integration tests can log a fixture account in.
const DefaultPassword = "fixture-password"

// defaultPasswordHash is derived once per test binary. The cost is deliberately
// low: fixtures need a *parseable argon2id* hash, not a slow one.
var defaultPasswordHash = sync.OnceValue(func() string {
	hasher := argon2id.New(argon2id.Params{Memory: 8 * 1024, Time: 1, Parallelism: 1})
	hash, err := hasher.Hash(DefaultPassword)
	if err != nil {
		panic("makeme: hash default password: " + err.Error())
	}
	return hash
})

// ANewGenre creates a Genre builder with a unique slug and Thai name.
func (m *MakeMe) ANewGenre() *Builder[entities.Genre] {
	m.t.Helper()
	seq := m.next()
	model := &entities.Genre{
		Slug:   fixtureCode("genre-", seq, 4),
		NameTH: fixtureThaiText("แนว", seq),
	}
	return newBuilder(m, model, func(row *entities.Genre) map[string]any {
		return map[string]any{"slug": row.Slug}
	})
}

// ANewUser creates a User builder for translator/reader fixtures.
func (m *MakeMe) ANewUser() *Builder[entities.User] {
	m.t.Helper()
	seq := m.next()
	model := &entities.User{
		Username:     fixtureCode("user_", seq, 6),
		Email:        fixtureCode("user_", seq, 6) + "@example.com",
		PasswordHash: defaultPasswordHash(),
		DisplayName:  fixtureThaiText("ผู้ใช้", seq),
		// Roles must be set explicitly: entities.User now maps the column, so
		// GORM sends it on INSERT and a nil value would violate NOT NULL rather
		// than falling back to the ARRAY['reader'] default.
		Roles:  pq.StringArray{entities.RoleReader},
		Status: "active",
	}
	return newBuilder(m, model, func(row *entities.User) map[string]any {
		return map[string]any{"username": row.Username}
	})
}

// ANewNovel creates a Novel builder with sane defaults; use With to set FKs.
func (m *MakeMe) ANewNovel() *Builder[entities.Novel] {
	m.t.Helper()
	seq := m.next()
	titleCN := fixtureCode("小说", seq, 4)
	author := fixtureThaiText("ผู้แต่ง", seq)
	desc := fixtureThaiText("เรื่องย่อ", seq)
	model := &entities.Novel{
		Slug:           fixtureCode("novel-", seq, 6),
		TitleTH:        fixtureThaiText("นิยาย", seq),
		TitleCN:        &titleCN,
		AuthorName:     &author,
		Description:    &desc,
		Status:         "ongoing",
		RatingAvg:      4.5,
		RatingCount:    100,
		FollowersCount: 1000,
		ChaptersCount:  10,
	}
	return newBuilder(m, model, func(row *entities.Novel) map[string]any {
		return map[string]any{"slug": row.Slug}
	})
}

// ANewNovelGenre creates a novel↔genre link builder.
func (m *MakeMe) ANewNovelGenre() *Builder[entities.NovelGenre] {
	m.t.Helper()
	model := &entities.NovelGenre{}
	return newBuilder(m, model, func(row *entities.NovelGenre) map[string]any {
		return map[string]any{"novel_id": row.NovelID, "genre_id": row.GenreID}
	})
}

// ANewArc creates an Arc builder. Set NovelID via With before Please.
func (m *MakeMe) ANewArc() *Builder[entities.Arc] {
	m.t.Helper()
	seq := m.next()
	model := &entities.Arc{
		ArcNo:         int16((seq % 20) + 1),
		Name:          fixtureThaiText("ภาค", seq),
		FromChapterNo: 1,
		ToChapterNo:   48,
	}
	return newBuilder(m, model, func(row *entities.Arc) map[string]any {
		return map[string]any{"novel_id": row.NovelID, "arc_no": row.ArcNo}
	})
}

// ANewChapter creates a Chapter builder. Set NovelID via With before Please.
func (m *MakeMe) ANewChapter() *Builder[entities.Chapter] {
	m.t.Helper()
	seq := m.next()
	published := timeForSequence(seq)
	model := &entities.Chapter{
		ChapterNo:   int(seq),
		Title:       fixtureThaiText("บท", seq),
		Status:      "published",
		PriceCoins:  0,
		WordCount:   3000,
		PublishedAt: &published,
	}
	return newBuilder(m, model, func(row *entities.Chapter) map[string]any {
		return map[string]any{"novel_id": row.NovelID, "chapter_no": row.ChapterNo}
	})
}

// ANewChapterBody creates a ChapterBody builder. Set ChapterID via With before Please.
func (m *MakeMe) ANewChapterBody() *Builder[entities.ChapterBody] {
	m.t.Helper()
	seq := m.next()
	model := &entities.ChapterBody{
		BodyHTML:    "<p>fixture body " + fixtureCode("", seq, 4) + "</p>",
		BodySource:  "fixture body source " + fixtureCode("", seq, 4),
		Revision:    1,
		GlossaryRev: 0,
		RenderedAt:  timeForSequence(seq),
		UpdatedAt:   timeForSequence(seq),
	}
	return newBuilder(m, model, func(row *entities.ChapterBody) map[string]any {
		return map[string]any{"chapter_id": row.ChapterID}
	})
}

// ANewGlossaryGroup creates a GlossaryGroup builder.
func (m *MakeMe) ANewGlossaryGroup() *Builder[entities.GlossaryGroup] {
	m.t.Helper()
	seq := m.next()
	model := &entities.GlossaryGroup{
		Name:   fixtureThaiText("หมวดอภิธาน", seq),
		SortNo: int16(seq % 100),
	}
	return newBuilder(m, model, func(row *entities.GlossaryGroup) map[string]any {
		return map[string]any{"novel_id": row.NovelID, "name": row.Name}
	})
}

// ANewGlossaryEntry creates a GlossaryEntry builder.
func (m *MakeMe) ANewGlossaryEntry() *Builder[entities.GlossaryEntry] {
	m.t.Helper()
	seq := m.next()
	titleCN := fixtureCode("术语", seq, 4)
	model := &entities.GlossaryEntry{
		TermKey: fixtureCode("term-", seq, 4),
		TitleTH: fixtureThaiText("ศัพท์", seq),
		TitleCN: &titleCN,
		Body:    fixtureThaiText("คำอธิบายศัพท์", seq),
	}
	return newBuilder(m, model, func(row *entities.GlossaryEntry) map[string]any {
		return map[string]any{"group_id": row.GroupID, "term_key": row.TermKey}
	})
}
