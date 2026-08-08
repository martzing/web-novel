// Package page carries the cursor-pagination request shape across domain
// packages. It is stdlib-only and depends on nothing, so every bounded context
// may import it without breaking the dependency rule.
package page

// Page is one page request: how many rows, starting after which opaque cursor.
type Page struct {
	Limit  int
	Cursor string
}

// Normalize clamps Limit into (0, max] falling back to def.
func (p Page) Normalize(def, max int) Page {
	if p.Limit <= 0 || p.Limit > max {
		p.Limit = def
	}
	return p
}
