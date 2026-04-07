package siphon

import "strings"

// ParseColonSyntax splits a "CONN:DATABASE" string into its connection and
// database parts. If no colon is present, the entire string is treated as
// the connection name and database is empty (meaning use the connection's
// default database).
func ParseColonSyntax(s string) (connection, database string) {
	idx := strings.Index(s, ":")
	if idx < 0 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
}
