package httpx

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// ErrBadCursor is returned when a cursor cannot be decoded or belongs to a
// different ordering than the request asks for.
var ErrBadCursor = errors.New("httpx: bad cursor")

// Cursor is an opaque keyset position. Keys hold the ordered key parts of the
// last row on the previous page, e.g. ["2026-01-02T03:04:05Z", "1832"].
//
// Keyset pagination (never OFFSET) keeps results stable while rows are being
// inserted concurrently, which matters for the ledger and comment feeds.
type Cursor struct {
	Sort string   `json:"s"`
	Keys []string `json:"k"`
}

// EncodeCursor renders a cursor as unpadded base64url JSON.
func EncodeCursor(c Cursor) string {
	raw, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// DecodeCursor parses a cursor produced by EncodeCursor. An empty string yields
// the zero cursor and no error, meaning "first page".
func DecodeCursor(s string) (Cursor, error) {
	if s == "" {
		return Cursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrBadCursor
	}
	var c Cursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return Cursor{}, ErrBadCursor
	}
	// Every cursor we mint carries at least one key part. Rejecting key-less
	// payloads means `{}` and `null` are treated as corruption rather than
	// silently restarting the caller at page one.
	if len(c.Keys) == 0 {
		return Cursor{}, ErrBadCursor
	}
	return c, nil
}

// DecodeCursorFor decodes a cursor and additionally rejects one minted under a
// different sort order, rather than silently mixing two orderings.
func DecodeCursorFor(s, sort string) (Cursor, error) {
	c, err := DecodeCursor(s)
	if err != nil {
		return Cursor{}, err
	}
	if s != "" && c.Sort != sort {
		return Cursor{}, ErrBadCursor
	}
	return c, nil
}
