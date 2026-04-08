package siphon

import (
	"regexp"
	"strings"
)

// ResolveConnectionMap finds the first connection map entry matching the given
// working directory. Returns the connection name, or empty string if no match.
//
// Resolution rules:
//   - Mode "prefix" (default): checks if workDir starts with the entry's Prefix.
//   - Mode "regex": checks if workDir matches the entry's Match regex pattern.
//
// Entries are evaluated in order; first match wins.
func ResolveConnectionMap(entries []ConnectionMapEntry, workDir, home string) string {
	for _, entry := range entries {
		mode := "prefix"
		if entry.Mode != nil && *entry.Mode != "" {
			mode = *entry.Mode
		}

		switch mode {
		case "prefix":
			if entry.Prefix == nil || *entry.Prefix == "" {
				continue
			}
			prefix := ExpandPath(*entry.Prefix, home)
			// Normalize: ensure prefix ends with / for proper directory matching.
			if !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}
			if strings.HasPrefix(workDir, prefix) || workDir == strings.TrimSuffix(prefix, "/") {
				return entry.Connection
			}

		case "regex":
			if entry.Match == nil || *entry.Match == "" {
				continue
			}
			re, err := regexp.Compile(*entry.Match)
			if err != nil {
				// Invalid regex: skip this entry.
				continue
			}
			if re.MatchString(workDir) {
				return entry.Connection
			}
		}
	}
	return ""
}
