package siphon

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandPath(t *testing.T) {
	home := "/home/testuser"

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"absolute", "/usr/local/bin", "/usr/local/bin"},
		{"relative", "data/local.db", "data/local.db"},
		{"tilde only", "~", home},
		{"tilde slash", "~/backups/nightly", filepath.Join(home, "backups/nightly")},
		{"tilde no slash", "~user", "~user"},
		{"dollar not expanded", "$HOME/backups", "$HOME/backups"},
		{"empty home", "~/data", "~/data"}, // empty home = no expansion
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := home
			if tt.name == "empty home" {
				h = ""
			}
			got := ExpandPath(tt.in, h)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestExpandPtrPath(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.Nil(t, ExpandPtrPath(nil, "/home/testuser"))
	})

	t.Run("expands tilde", func(t *testing.T) {
		home := "/home/testuser"
		in := "~/data"
		result := ExpandPtrPath(&in, home)
		require.NotNil(t, result)
		require.Equal(t, filepath.Join(home, "data"), *result)
	})
}
