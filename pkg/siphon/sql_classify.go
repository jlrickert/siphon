package siphon

import "strings"

// writeKeywords are SQL keywords that indicate a write (mutating) operation.
var writeKeywords = map[string]bool{
	"INSERT":   true,
	"UPDATE":   true,
	"DELETE":   true,
	"DROP":     true,
	"ALTER":    true,
	"CREATE":   true,
	"TRUNCATE": true,
	"REPLACE":  true,
	"RENAME":   true,
	"GRANT":    true,
	"REVOKE":   true,
}

// IsWriteQuery returns true if the SQL appears to be a write operation.
// It checks the first meaningful keyword after stripping leading whitespace
// and SQL comments. This is a best-effort classifier, not a SQL parser.
func IsWriteQuery(sql string) bool {
	trimmed := stripLeadingComments(sql)
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return false
	}

	// Extract first word.
	firstWord := trimmed
	if idx := strings.IndexAny(trimmed, " \t\r\n("); idx >= 0 {
		firstWord = trimmed[:idx]
	}

	return writeKeywords[strings.ToUpper(firstWord)]
}

// stripLeadingComments removes leading SQL comments (-- and /* */) from
// the input string.
func stripLeadingComments(s string) string {
	for {
		s = strings.TrimSpace(s)
		if strings.HasPrefix(s, "--") {
			// Single-line comment: skip to end of line.
			if idx := strings.IndexByte(s, '\n'); idx >= 0 {
				s = s[idx+1:]
			} else {
				return ""
			}
		} else if strings.HasPrefix(s, "/*") {
			// Block comment: skip to closing */.
			if idx := strings.Index(s, "*/"); idx >= 0 {
				s = s[idx+2:]
			} else {
				return ""
			}
		} else {
			return s
		}
	}
}
