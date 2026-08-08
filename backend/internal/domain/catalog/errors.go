package catalog

import "errors"

// ErrNotFound is returned when a domain lookup finds no matching aggregate.
var ErrNotFound = errors.New("catalog: not found")
