package glossaryrender

import (
	"reflect"
	"strings"
	"testing"
)

var terms = []Term{
	{EntryID: 1, Key: "qi", TitleTH: "ชี่"},
	{EntryID: 2, Key: "dantian", TitleTH: "ตันเถียน"},
	{EntryID: 3, Key: "ye", TitleTH: "เยี่ยหลิงเฟิง"},
}

func TestRender_BindsKnownTerms(t *testing.T) {
	got := Render("<p>{{qi}} ไหลออกจาก{{dantian}}</p>", terms)

	want := `<p><span data-k="qi">ชี่</span> ไหลออกจาก<span data-k="dantian">ตันเถียน</span></p>`
	if got.HTML != want {
		t.Fatalf("HTML =\n%s\nwant\n%s", got.HTML, want)
	}
	if !reflect.DeepEqual(got.EntryIDs, []int64{1, 2}) {
		t.Fatalf("EntryIDs = %v, want [1 2]", got.EntryIDs)
	}
	if len(got.Unknown) != 0 {
		t.Fatalf("Unknown = %v, want none", got.Unknown)
	}
}

// The default display text comes from the entry, which is what makes the
// seeded `<span data-k="ye">เยี่ยหลิงเฟิง</span>` output reproducible.
func TestRender_UsesTheEntryTitleAsDefaultDisplayText(t *testing.T) {
	got := Render("{{ye}}", terms)
	if got.HTML != `<span data-k="ye">เยี่ยหลิงเฟิง</span>` {
		t.Fatalf("HTML = %s", got.HTML)
	}
}

func TestRender_HonoursExplicitDisplayText(t *testing.T) {
	got := Render("{{ye|เขา}} ยกดาบขึ้น", terms)
	if got.HTML != `<span data-k="ye">เขา</span> ยกดาบขึ้น` {
		t.Fatalf("HTML = %s", got.HTML)
	}
	if !reflect.DeepEqual(got.EntryIDs, []int64{3}) {
		t.Fatalf("EntryIDs = %v, want [3]", got.EntryIDs)
	}
}

func TestRender_TrimsWhitespaceInsideMarkers(t *testing.T) {
	got := Render("{{  qi  |  พลัง  }}", terms)
	if got.HTML != `<span data-k="qi">พลัง</span>` {
		t.Fatalf("HTML = %s", got.HTML)
	}
}

// A typo must never delete the writer's text; it is passed through and
// reported so the editor can flag it.
func TestRender_LeavesUnknownMarkersVerbatim(t *testing.T) {
	got := Render("<p>{{qi}} และ {{unknown_term}}</p>", terms)

	if !strings.Contains(got.HTML, "{{unknown_term}}") {
		t.Fatalf("an unknown marker must survive verbatim, got %s", got.HTML)
	}
	if !strings.Contains(got.HTML, `<span data-k="qi">`) {
		t.Fatal("known terms in the same source must still bind")
	}
	if !reflect.DeepEqual(got.Unknown, []string{"unknown_term"}) {
		t.Fatalf("Unknown = %v, want [unknown_term]", got.Unknown)
	}
}

func TestRender_ReportsEachUnknownKeyOnce(t *testing.T) {
	got := Render("{{nope}} {{nope}} {{nope}}", terms)
	if !reflect.DeepEqual(got.Unknown, []string{"nope"}) {
		t.Fatalf("Unknown = %v, want a single entry", got.Unknown)
	}
}

func TestRender_MalformedMarkersArePassedThrough(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"unterminated marker", "<p>{{qi</p>"},
		{"empty marker", "<p>{{}}</p>"},
		{"key with illegal characters", "<p>{{ปลาดาบ}}</p>"},
		{"key with a space", "<p>{{two words}}</p>"},
		{"bare braces", "<p>{ qi }</p>"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Render(tc.source, terms)
			if got.HTML != tc.source {
				t.Fatalf("HTML = %q, want the source unchanged", got.HTML)
			}
			if len(got.EntryIDs) != 0 {
				t.Fatalf("EntryIDs = %v, want none", got.EntryIDs)
			}
		})
	}
}

// Display text is writer-controlled, so it must not be able to close the span
// or inject an attribute.
func TestRender_EscapesDisplayTextAndKeys(t *testing.T) {
	got := Render(`{{qi|<script>alert(1)</script>}}`, terms)

	if strings.Contains(got.HTML, "<script>") {
		t.Fatalf("display text was not escaped: %s", got.HTML)
	}
	if !strings.Contains(got.HTML, "&lt;script&gt;") {
		t.Fatalf("expected escaped display text, got %s", got.HTML)
	}

	quoted := Render(`{{qi|say "hi"}}`, terms)
	if strings.Contains(quoted.HTML, `data-k="qi">say "hi"`) {
		t.Fatalf("quotes in display text must be escaped: %s", quoted.HTML)
	}
}

func TestRender_DedupesEntryIDs(t *testing.T) {
	got := Render("{{qi}} {{qi}} {{dantian}} {{qi}}", terms)

	if !reflect.DeepEqual(got.EntryIDs, []int64{1, 2}) {
		t.Fatalf("EntryIDs = %v, want deduplicated and sorted [1 2]", got.EntryIDs)
	}
}

// Writers author HTML paragraphs, so everything outside a marker must survive
// byte for byte.
func TestRender_PassesSurroundingMarkupThrough(t *testing.T) {
	source := `<p class="x">ก่อน</p>
<hr/>
<p>หลัง &amp; ยัง</p>`
	got := Render(source, terms)
	if got.HTML != source {
		t.Fatalf("HTML =\n%s\nwant it unchanged", got.HTML)
	}
}

func TestRender_EmptyInputs(t *testing.T) {
	if got := Render("", terms); got.HTML != "" || len(got.EntryIDs) != 0 {
		t.Fatalf("empty source produced %+v", got)
	}
	// With no glossary at all, every marker is unknown and nothing is lost.
	got := Render("{{qi}}", nil)
	if got.HTML != "{{qi}}" {
		t.Fatalf("HTML = %q, want the marker preserved", got.HTML)
	}
	if !reflect.DeepEqual(got.Unknown, []string{"qi"}) {
		t.Fatalf("Unknown = %v, want [qi]", got.Unknown)
	}
}

func TestRender_AdjacentMarkers(t *testing.T) {
	got := Render("{{qi}}{{dantian}}", terms)
	want := `<span data-k="qi">ชี่</span><span data-k="dantian">ตันเถียน</span>`
	if got.HTML != want {
		t.Fatalf("HTML = %s, want %s", got.HTML, want)
	}
}
