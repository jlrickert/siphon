package siphon_test

import (
	"strings"
	"testing"

	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

func TestGeneratePlist_ValidXML(t *testing.T) {
	t.Parallel()

	cfg := siphon.LaunchdConfig{
		Label:   "com.siphon.backup.nightly",
		Program: "/usr/local/bin/siphon",
		Arguments: []string{
			"backup", "create", "dev",
			"--repo", "nightly",
		},
		StartCalendarInterval: map[string]int{
			"Hour":   2,
			"Minute": 0,
		},
		StandardOutPath:   "/tmp/siphon-nightly.out.log",
		StandardErrorPath: "/tmp/siphon-nightly.err.log",
	}

	data, err := siphon.GeneratePlist(cfg)
	require.NoError(t, err)

	xml := string(data)

	// Verify it is valid plist structure.
	require.Contains(t, xml, `<?xml version="1.0" encoding="UTF-8"?>`)
	require.Contains(t, xml, `<!DOCTYPE plist`)
	require.Contains(t, xml, `<plist version="1.0">`)
	require.Contains(t, xml, `</plist>`)

	// Verify label.
	require.Contains(t, xml, `<string>com.siphon.backup.nightly</string>`)

	// Verify program arguments.
	require.Contains(t, xml, `<string>/usr/local/bin/siphon</string>`)
	require.Contains(t, xml, `<string>backup</string>`)
	require.Contains(t, xml, `<string>create</string>`)
	require.Contains(t, xml, `<string>dev</string>`)
	require.Contains(t, xml, `<string>--repo</string>`)
	require.Contains(t, xml, `<string>nightly</string>`)

	// Verify calendar interval.
	require.Contains(t, xml, `<key>Hour</key>`)
	require.Contains(t, xml, `<integer>2</integer>`)
	require.Contains(t, xml, `<key>Minute</key>`)
	require.Contains(t, xml, `<integer>0</integer>`)

	// Verify log paths.
	require.Contains(t, xml, `<key>StandardOutPath</key>`)
	require.Contains(t, xml, `<string>/tmp/siphon-nightly.out.log</string>`)
	require.Contains(t, xml, `<key>StandardErrorPath</key>`)
	require.Contains(t, xml, `<string>/tmp/siphon-nightly.err.log</string>`)
}

func TestGeneratePlist_NoLogPaths(t *testing.T) {
	t.Parallel()

	cfg := siphon.LaunchdConfig{
		Label:   "com.siphon.backup.simple",
		Program: "/usr/local/bin/siphon",
		Arguments: []string{
			"backup", "create", "dev",
		},
		StartCalendarInterval: map[string]int{
			"Minute": 30,
		},
	}

	data, err := siphon.GeneratePlist(cfg)
	require.NoError(t, err)

	xml := string(data)

	// Should NOT contain log path keys when not set.
	require.False(t, strings.Contains(xml, "StandardOutPath"))
	require.False(t, strings.Contains(xml, "StandardErrorPath"))
}

func TestPlistFilename(t *testing.T) {
	t.Parallel()
	require.Equal(t, "com.siphon.backup.nightly.plist", siphon.PlistFilename("nightly"))
}

func TestPlistLabel(t *testing.T) {
	t.Parallel()
	require.Equal(t, "com.siphon.backup.nightly", siphon.PlistLabel("nightly"))
}
