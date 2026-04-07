package siphon_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

func TestManifest_WriteReadRoundTrip(t *testing.T) {
	t.Parallel()

	sb := sandbox.NewSandbox(t, &sandbox.Options{
		Home: "/home/testuser",
		User: "testuser",
	})

	original := &siphon.BackupManifest{
		Version:    "1",
		Engine:     "mariadb",
		Type:       "logical",
		Connection: "dev",
		Database:   "testdb",
		Timestamp:  time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC),
		Checksum:   "abc123",
		Compress:   "gzip",
		Label:      "nightly",
		Message:    "scheduled backup",
		Files: []siphon.BackupFileEntry{
			{Path: "dump.sql.gz", Size: 1024, Checksum: "def456"},
			{Path: "schema.sql.gz", Size: 512, Checksum: "ghi789"},
		},
		Size: 1536,
	}

	dir := filepath.Join(sb.GetJail(), "backups", "test-backup")
	sb.Mkdir(dir, true)
	path := filepath.Join(dir, siphon.ManifestFilename)

	err := siphon.WriteManifest(sb.Runtime(), path, original)
	require.NoError(t, err)

	loaded, err := siphon.ReadManifest(sb.Runtime(), path)
	require.NoError(t, err)

	require.Equal(t, original.Version, loaded.Version)
	require.Equal(t, original.Engine, loaded.Engine)
	require.Equal(t, original.Type, loaded.Type)
	require.Equal(t, original.Connection, loaded.Connection)
	require.Equal(t, original.Database, loaded.Database)
	require.Equal(t, original.Checksum, loaded.Checksum)
	require.Equal(t, original.Compress, loaded.Compress)
	require.Equal(t, original.Label, loaded.Label)
	require.Equal(t, original.Message, loaded.Message)
	require.Equal(t, original.Size, loaded.Size)
	require.True(t, original.Timestamp.Equal(loaded.Timestamp), "timestamps should match")
	require.Len(t, loaded.Files, 2)
	require.Equal(t, original.Files[0].Path, loaded.Files[0].Path)
	require.Equal(t, original.Files[0].Size, loaded.Files[0].Size)
	require.Equal(t, original.Files[0].Checksum, loaded.Files[0].Checksum)
	require.Equal(t, original.Files[1].Path, loaded.Files[1].Path)
}

func TestManifest_ReadNotFound(t *testing.T) {
	t.Parallel()

	sb := sandbox.NewSandbox(t, &sandbox.Options{
		Home: "/home/testuser",
		User: "testuser",
	})

	_, err := siphon.ReadManifest(sb.Runtime(), "/nonexistent/backup.meta.yaml")
	require.Error(t, err)
}

func TestManifestFilename(t *testing.T) {
	require.Equal(t, "backup.meta.yaml", siphon.ManifestFilename)
}
