// Package glossaryrender turns a chapter's authored source into the rendered
// HTML the reader receives, baking glossary terms into `<span data-k>` spans.
//
// It is deliberately pure and stdlib-only: rendering is the one step that must
// behave identically whether it runs during a publish or in the background
// re-render worker, so it is fully unit-testable without a database.
package glossaryrender

import (
	"html"
	"slices"
	"strings"
)

// Term is one glossary entry the renderer may bind.
type Term struct {
	EntryID int64
	Key     string
	TitleTH string
}

// Result is the rendered HTML plus the entries that were actually bound.
type Result struct {
	HTML string
	// EntryIDs is sorted and deduplicated, ready for chapter_glossary_refs.
	EntryIDs []int64
	// Unknown lists markers with no matching term, surfaced to the writer
	// rather than silently dropped.
	Unknown []string
}

// Render replaces every `{{key}}` or `{{key|display text}}` marker in source
// with `<span data-k="key">display</span>`.
//
// Unknown or malformed markers are left verbatim and reported in Unknown: a
// typo must never delete a translator's prose. Text outside markers passes
// through unchanged, because writers author HTML paragraphs directly.
func Render(source string, terms []Term) Result {
	byKey := make(map[string]Term, len(terms))
	for _, t := range terms {
		byKey[t.Key] = t
	}

	var (
		out     strings.Builder
		bound   = make(map[int64]bool)
		unknown []string
	)
	out.Grow(len(source) + len(source)/8)

	for i := 0; i < len(source); {
		start := strings.Index(source[i:], "{{")
		if start < 0 {
			out.WriteString(source[i:])
			break
		}
		start += i
		out.WriteString(source[i:start])

		end := strings.Index(source[start:], "}}")
		if end < 0 {
			// An unterminated marker is just text.
			out.WriteString(source[start:])
			break
		}
		end += start

		marker := source[start+2 : end]
		key, display := splitMarker(marker)

		term, known := byKey[key]
		switch {
		case !validKey(key):
			out.WriteString(source[start : end+2])
		case !known:
			out.WriteString(source[start : end+2])
			if !slices.Contains(unknown, key) {
				unknown = append(unknown, key)
			}
		default:
			if display == "" {
				display = term.TitleTH
			}
			out.WriteString(`<span data-k="`)
			out.WriteString(html.EscapeString(key))
			out.WriteString(`">`)
			out.WriteString(html.EscapeString(display))
			out.WriteString(`</span>`)
			bound[term.EntryID] = true
		}
		i = end + 2
	}

	entryIDs := make([]int64, 0, len(bound))
	for id := range bound {
		entryIDs = append(entryIDs, id)
	}
	slices.Sort(entryIDs)

	return Result{HTML: out.String(), EntryIDs: entryIDs, Unknown: unknown}
}

// splitMarker separates `key|display` into its parts.
func splitMarker(marker string) (key, display string) {
	key, display, _ = strings.Cut(marker, "|")
	return strings.TrimSpace(key), strings.TrimSpace(display)
}

// validKey mirrors the charset allowed for glossary_entries.term_key.
func validKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}
