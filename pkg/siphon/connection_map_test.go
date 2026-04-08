package siphon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

func TestResolveConnectionMap_EmptyEntries(t *testing.T) {
	t.Parallel()
	result := ResolveConnectionMap(nil, "/home/user/project", "/home/user")
	require.Equal(t, "", result)
}

func TestResolveConnectionMap_PrefixMatch(t *testing.T) {
	t.Parallel()
	entries := []ConnectionMapEntry{
		{Prefix: strPtr("/home/user/myapp"), Connection: "dev"},
		{Prefix: strPtr("/srv/staging"), Connection: "staging"},
	}

	// Exact prefix match.
	result := ResolveConnectionMap(entries, "/home/user/myapp", "/home/user")
	require.Equal(t, "dev", result)

	// Subdirectory of prefix.
	result = ResolveConnectionMap(entries, "/home/user/myapp/src/main", "/home/user")
	require.Equal(t, "dev", result)

	// Different path, no match.
	result = ResolveConnectionMap(entries, "/home/user/other", "/home/user")
	require.Equal(t, "", result)

	// Second entry matches.
	result = ResolveConnectionMap(entries, "/srv/staging/deploy", "/home/user")
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

	result := ResolveConnectionMap(entries, "/home/user/myapp-staging", "/home/user")
	require.Equal(t, "staging", result)

	result = ResolveConnectionMap(entries, "/home/user/myapp-prod", "/home/user")
	require.Equal(t, "prod", result)

	result = ResolveConnectionMap(entries, "/home/user/myapp-dev", "/home/user")
	require.Equal(t, "", result)
}

func TestResolveConnectionMap_FirstMatchWins(t *testing.T) {
	t.Parallel()
	entries := []ConnectionMapEntry{
		{Prefix: strPtr("/home/user/myapp"), Connection: "first"},
		{Prefix: strPtr("/home/user/myapp"), Connection: "second"},
	}

	result := ResolveConnectionMap(entries, "/home/user/myapp/foo", "/home/user")
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
	result := ResolveConnectionMap(entries, "/home/user", "/home/user")
	require.Equal(t, "fallback", result)
}

func TestResolveConnectionMap_DefaultModeIsPrefix(t *testing.T) {
	t.Parallel()
	entries := []ConnectionMapEntry{
		{Prefix: strPtr("/srv/app"), Connection: "app-db"},
	}

	// Mode is nil, defaults to "prefix".
	result := ResolveConnectionMap(entries, "/srv/app/deploy", "/home/user")
	require.Equal(t, "app-db", result)
}

func TestResolveConnectionMap_EmptyPrefixSkipped(t *testing.T) {
	t.Parallel()
	entries := []ConnectionMapEntry{
		{Prefix: strPtr(""), Connection: "empty"},
		{Prefix: strPtr("/home"), Connection: "home"},
	}

	result := ResolveConnectionMap(entries, "/home/user", "/home/user")
	require.Equal(t, "home", result)
}

func TestResolveConnectionMap_TildeExpansion(t *testing.T) {
	t.Parallel()
	entries := []ConnectionMapEntry{
		{Prefix: strPtr("~/myapp"), Connection: "dev"},
	}

	result := ResolveConnectionMap(entries, "/home/user/myapp/src", "/home/user")
	require.Equal(t, "dev", result)

	result = ResolveConnectionMap(entries, "/other/myapp/src", "/home/user")
	require.Equal(t, "", result)
}
