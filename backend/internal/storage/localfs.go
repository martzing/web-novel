// Package storage holds the file-storage adapters behind the writer's Storage
// port. Only the local-disk adapter exists today; an object-store adapter drops
// in without touching the service.
package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalFS writes uploads to a directory served by the API.
type LocalFS struct {
	dir       string
	publicURL string
}

// NewLocalFS builds an adapter writing under dir and serving from publicBase.
func NewLocalFS(dir, publicBase string) *LocalFS {
	return &LocalFS{dir: dir, publicURL: strings.TrimRight(publicBase, "/")}
}

// Save writes the bytes and returns the public URL.
//
// The name is server-generated; it is still base-named here so a traversal
// sequence can never escape the upload directory.
func (l *LocalFS) Save(_ context.Context, name, _ string, data []byte) (string, error) {
	safe := filepath.Base(name)
	if safe == "." || safe == string(filepath.Separator) {
		return "", fmt.Errorf("storage: invalid file name %q", name)
	}

	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return "", fmt.Errorf("storage: create upload dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(l.dir, safe), data, 0o644); err != nil {
		return "", fmt.Errorf("storage: write file: %w", err)
	}
	return l.publicURL + "/" + safe, nil
}
