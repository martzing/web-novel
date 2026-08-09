package writer_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/test/apitest"
)

// glossaryFixture is a translator's novel with one glossary group holding one
// term, which is the smallest shape that can exercise every delete rule.
type glossaryFixture struct {
	*writerFixture
	group *entities.GlossaryGroup
	entry *entities.GlossaryEntry
}

func newGlossaryFixture(t *testing.T) *glossaryFixture {
	t.Helper()
	f := newWriterFixture(t)
	m := f.env.MakeMe

	group := m.ANewGlossaryGroup().With(func(g *entities.GlossaryGroup) {
		g.NovelID = f.novel.ID
		g.Name = "ศัพท์การบำเพ็ญ"
	}).Please()

	entry := m.ANewGlossaryEntry().With(func(e *entities.GlossaryEntry) {
		e.GroupID = group.ID
		e.TermKey = "qi"
		e.TitleTH = "ชี่"
	}).Please()

	return &glossaryFixture{writerFixture: f, group: group, entry: entry}
}

func (f *glossaryFixture) novelGlossaryRev(t *testing.T) int {
	t.Helper()
	var n entities.Novel
	if err := f.env.MakeMe.DB.Where("id = ?", f.novel.ID).Take(&n).Error; err != nil {
		t.Fatalf("load novel: %v", err)
	}
	return n.GlossaryRev
}

// I-WR-05 — deleting a term removes it, bumps glossary_rev and takes its
// chapter bindings with it.
//
// The rev bump is what makes the re-render worker rewrite every body that bound
// the term; without it a deleted word would keep rendering as a live span.
func TestDeleteGlossaryEntry_BumpsGlossaryRevAndClearsBindings(t *testing.T) {
	f := newGlossaryFixture(t)

	chapter := f.createChapter(t, 87, "ดาบแรกใต้ฟ้าหมอก", "<p>{{qi}} ไหลเวียน</p>", 5)
	rec := f.env.Do(apitest.Request{
		Method: http.MethodPost,
		Path:   "/api/v1/writer/chapters/" + chapter.ID + "/publish",
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusOK)

	before := f.novelGlossaryRev(t)

	rec = f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   fmt.Sprintf("/api/v1/writer/glossary-entries/%d", f.entry.ID),
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusNoContent)

	var remaining int64
	if err := f.env.MakeMe.DB.Model(&entities.GlossaryEntry{}).
		Where("id = ?", f.entry.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("count entries: %v", err)
	}
	if remaining != 0 {
		t.Fatal("the entry should be gone")
	}

	if after := f.novelGlossaryRev(t); after <= before {
		t.Fatalf("glossary_rev = %d, want it bumped past %d so the worker re-renders", after, before)
	}

	// chapter_glossary_refs.entry_id is ON DELETE CASCADE, so the binding goes
	// with the term rather than dangling.
	var refs int64
	if err := f.env.MakeMe.DB.Model(&entities.ChapterGlossaryRef{}).
		Where("entry_id = ?", f.entry.ID).Count(&refs).Error; err != nil {
		t.Fatalf("count refs: %v", err)
	}
	if refs != 0 {
		t.Fatalf("glossary refs = %d, want the cascade to have removed them", refs)
	}
}

// I-WR-06 — a translator cannot delete another translator's term.
func TestDeleteGlossaryEntry_ForbiddenForAnotherTranslator(t *testing.T) {
	f := newGlossaryFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   fmt.Sprintf("/api/v1/writer/glossary-entries/%d", f.entry.ID),
		Token:  f.strToken,
	})
	apitest.AssertStatus(t, rec, http.StatusForbidden)

	var remaining int64
	if err := f.env.MakeMe.DB.Model(&entities.GlossaryEntry{}).
		Where("id = ?", f.entry.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("count entries: %v", err)
	}
	if remaining != 1 {
		t.Fatal("a forbidden delete must not remove the entry")
	}
}

// I-WR-07 — a group holding terms refuses to be deleted.
//
// Cascading here would destroy a translator's work behind one click on a
// container they may only have meant to rename.
func TestDeleteGlossaryGroup_RefusesWhileItStillHoldsTerms(t *testing.T) {
	f := newGlossaryFixture(t)

	rec := f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   fmt.Sprintf("/api/v1/writer/glossary-groups/%d", f.group.ID),
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusConflict)
	if !strings.Contains(rec.Body.String(), "GROUP_NOT_EMPTY") {
		t.Fatalf("body = %s, want GROUP_NOT_EMPTY", rec.Body.String())
	}

	// Emptying it first makes the delete succeed.
	rec = f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   fmt.Sprintf("/api/v1/writer/glossary-entries/%d", f.entry.ID),
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusNoContent)

	rec = f.env.Do(apitest.Request{
		Method: http.MethodDelete,
		Path:   fmt.Sprintf("/api/v1/writer/glossary-groups/%d", f.group.ID),
		Token:  f.token,
	})
	apitest.AssertStatus(t, rec, http.StatusNoContent)
}
