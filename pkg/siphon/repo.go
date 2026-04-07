package siphon

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveRepoPath resolves a backup path that may use @repo syntax.
// For example, "@nightly/mydb-2026-04-02" resolves the "nightly" repo
// from config and appends the subpath.
func ResolveRepoPath(cfg *Config, raw string) (string, error) {
	if !strings.HasPrefix(raw, "@") {
		return raw, nil
	}

	// Strip the @ prefix and split on first /.
	rest := raw[1:]
	repoName := rest
	subpath := ""
	if idx := strings.Index(rest, "/"); idx >= 0 {
		repoName = rest[:idx]
		subpath = rest[idx+1:]
	}

	if cfg == nil || cfg.Repos == nil {
		return "", fmt.Errorf("repo %q not configured (no repos defined)", repoName)
	}

	repo, ok := cfg.Repos[repoName]
	if !ok {
		return "", fmt.Errorf("repo %q not configured", repoName)
	}

	if repo.Path == nil || *repo.Path == "" {
		return "", fmt.Errorf("repo %q has no path configured", repoName)
	}

	if subpath == "" {
		return *repo.Path, nil
	}
	return filepath.Join(*repo.Path, subpath), nil
}
