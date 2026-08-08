package makeme_test

import (
	"context"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/makeme"
)

func TestMakeMe_Genre(t *testing.T) {
	m := makeme.New(t)

	genre := m.ANewGenre().
		With(func(row *entities.Genre) {
			row.Slug = "xianxia-fixture"
			row.NameTH = "เซียน"
		}).
		Please()

	if genre.ID == 0 {
		t.Fatalf("expected id to be populated, got %d", genre.ID)
	}

	found := m.ANewGenre().Find(makeme.QueryCriteria{
		Where: map[string]any{"slug": "xianxia-fixture"},
	})
	if found.NameTH != "เซียน" {
		t.Fatalf("expected NameTH=เซียน, got %q", found.NameTH)
	}
}

func TestMakeMe_NovelWithArcAndChapter(t *testing.T) {
	m := makeme.New(t)

	translator := m.ANewUser().Please()

	novel := m.ANewNovel().
		With(func(row *entities.Novel) {
			row.Slug = "nine-streams"
			row.TitleTH = "เซียนดาบเก้าสายธาร"
			row.PrimaryTranslatorID = &translator.ID
		}).
		Please()

	arc := m.ANewArc().
		With(func(row *entities.Arc) {
			row.NovelID = novel.ID
			row.Name = "สำนักเมฆาวสันต์"
			row.FromChapterNo = 49
			row.ToChapterNo = 120
		}).
		Please()

	chapter := m.ANewChapter().
		With(func(row *entities.Chapter) {
			row.NovelID = novel.ID
			row.ArcID = &arc.ID
			row.ChapterNo = 87
			row.Title = "ดาบแรกใต้ฟ้าหมอก"
		}).
		Please()

	m.ANewChapterBody().
		With(func(row *entities.ChapterBody) {
			row.ChapterID = chapter.ID
			row.BodyHTML = "<p>หิมะบางโปรยลงเหนือหุบเขาเมฆาวสันต์...</p>"
			row.BodySource = "หิมะบางโปรยลงเหนือหุบเขาเมฆาวสันต์..."
		}).
		Please()

	chapters := m.ANewChapter().List(makeme.QueryCriteria{
		Where: "novel_id = ?",
		Args:  []any{novel.ID},
		Order: "chapter_no",
	})
	if len(chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(chapters))
	}
	if chapters[0].Title != "ดาบแรกใต้ฟ้าหมอก" {
		t.Fatalf("unexpected chapter title: %s", chapters[0].Title)
	}

	_ = context.Background()
}

func TestMakeMe_ResetBetweenSubtests(t *testing.T) {
	m := makeme.New(t)

	t.Run("case one", func(t *testing.T) {
		m.Reset()
		m.ANewGenre().Please()
	})

	t.Run("case two", func(t *testing.T) {
		m.Reset()
		list := m.ANewGenre().List(makeme.QueryCriteria{Order: "id"})
		if len(list) != 0 {
			t.Fatalf("expected empty genres after reset, got %d rows", len(list))
		}
	})
}
