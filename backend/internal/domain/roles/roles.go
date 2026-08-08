// Package roles names the values stored in users.roles. It is stdlib-only and
// dependency-free, so any domain package may import it.
package roles

const (
	Reader     = "reader"
	Translator = "translator"
	Admin      = "admin"
)
