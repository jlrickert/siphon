package siphon

import (
	"path/filepath"
	"strings"
)

// ExpandPath expands a leading ~ in a path to the given home directory.
// If the path does not start with ~, it is returned unchanged.
func ExpandPath(path, home string) string {
	if path == "" || home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// ExpandPtrPath expands a leading ~ in a *string path. Returns nil if the
// input is nil.
func ExpandPtrPath(p *string, home string) *string {
	if p == nil {
		return nil
	}
	expanded := ExpandPath(*p, home)
	return &expanded
}
