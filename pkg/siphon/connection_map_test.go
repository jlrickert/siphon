package siphon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

func TestResolveConnectionMap_EmptyEntries(t *testing.T) {
	t.Parallel()
	result := ResolveConnectionMap(nil, "/home/user/project")
	require.Equal(t, "", result)
}

func TestResolveConnectionMap_PrefixMatch(t *testing.T) {
	t.Parallel()
	entries := []ConnectionMapEntry{
		{Prefix: strPtr("/home/user/myapp"), Connection: "dev"},
		{Prefix: strPtr("/srv/staging"), Connection: "staging"},
	}

	// Exact prefix match.
	result := ResolveConnectionMap(entries, "/home/user/myapp")
	require.Equal(t, "dev", result)

	// Subdirectory of prefix.
	result = ResolveConnectionMap(entries, "/home/user/myapp/src/main")
	require.Equal(t, "dev", result)

	// Different path, no match.
	result = ResolveConnectionMap(entries, "/home/user/other")
	require.Equal(t, "", result)

	// Second entry matches.
	result = ResolveConnectionMap(entries, "/srv/staging/deploy")
	require.Equal(t, "staging", result)
}

func TestResolveConnectionMap_RegexMatch(t *testing.T) {
	t.Parallel()
	regexMode := "regex"
	entries := []ConnectionMapEntry{
		{
			Match:      strPtr(`/home/user/.*-staging`),
			Mode:       &regexMode,
			Connection: "staging",
		},
		{
			Match:      strPtr(`/home/user/.*-prod`),
			Mode:       &regexMode,
			Connection: "prod",
		},
	}

	result := ResolveConnectionMap(entries, "/home/user/myapp-staging")
	require.Equal(t, "staging", result)

	result = ResolveConnectionMap(entries, "/home/user/myapp-prod")
	require.Equal(t, "prod", result)

	result = ResolveConnectionMap(entries, "/home/user/myapp-dev")
	require.Equal(t, "", result)
}

func TestResolveConnectionMap_FirstMatchWins(t *testing.T) {
	t.Parallel()
	entries := []ConnectionMapEntry{
		{Prefix: strPtr("/home/user/myapp"), Connection: "first"},
		{Prefix: strPtr("/home/user/myapp"), Connection: "second"},
	}

	result := ResolveConnectionMap(entries, "/home/user/myapp/foo")
	require.Equal(t, "first", result)
}

func TestResolveConnectionMap_InvalidRegex(t *testing.T) {
	t.Parallel()
	regexMode := "regex"
	entries := []ConnectionMapEntry{
		{
			Match:      strPtr("[invalid"),
			Mode:       &regexMode,
			Connection: "bad",
		},
		{Prefix: strPtr("/home"), Connection: "fallback"},
	}

	// Invalid regex is skipped, fallback prefix matches.
	result := ResolveConnectionMap(entries, "/home/user")
	require.Equal(t, "fallback", result)
}

func TestResolveConnectionMap_DefaultModeIsPrefix(t *testing.T) {
	t.Parallel()
	entries := []ConnectionMapEntry{
		{Prefix: strPtr("/srv/app"), Connection: "app-db"},
	}

	// Mode is nil, defaults to "prefix".
	result := ResolveConnectionMap(entries, "/srv/app/deploy")
	require.Equal(t, "app-db", result)
}

func TestResolveConnectionMap_EmptyPrefixSkipped(t *testing.T) {
	t.Parallel()
	entries := []ConnectionMapEntry{
		{Prefix: strPtr(""), Connection: "empty"},
		{Prefix: strPtr("/home"), Connection: "home"},
	}

	result := ResolveConnectionMap(entries, "/home/user")
	require.Equal(t, "home", result)
}
