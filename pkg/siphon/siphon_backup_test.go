package siphon_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

// newTestSiphonWithMock creates a test Siphon with a mock adaptor factory.
// The mock adaptor's DumpFn writes "-- mock dump data" so that logical
// backup produces testable output.
func newTestSiphonWithMock(t *testing.T) (*siphon.Siphon, *sandbox.Sandbox) {
	t.Helper()

	sb := sandbox.NewSandbox(t, &sandbox.Options{
		Data: testdata,
		Home: "/home/testuser",
		User: "testuser",
	}, sandbox.WithFixture("testuser", "~"))

	s, err := siphon.New(siphon.SiphonOptions{
		Runtime: sb.Runtime(),
	})
	require.NoError(t, err)

	// Override the adaptor factory with a mock.
	s.AdaptorFactory = func(cfg *engine.ConnectionConfig) (engine.Adaptor, error) {
		mock := engine.NewMockAdaptor(cfg.Engine)
		mock.DumpFn = func(ctx context.Context, opts engine.DumpOptions) error {
			if opts.Output != nil {
				_, _ = opts.Output.Write([]byte("-- mock dump data\n"))
			}
			return nil
		}
		mock.CopyBackupFn = func(ctx context.Context, opts engine.CopyBackupOptions) error {
			// Write mock data at the destination.
			return sb.WriteFile(opts.DestPath, []byte("mock sqlite data"), 0o644)
		}
		return mock, nil
	}

	return s, sb
}

func TestBackup_LogicalDump(t *testing.T) {
	t.Parallel()
	s, sb := newTestSiphonWithMock(t)
	ctx := context.Background()

	// Create a backup directory we can inspect.
	desc, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Compress:   "none",
	})
	require.NoError(t, err)
	require.NotNil(t, desc)
	require.Equal(t, "mariadb", string(desc.Engine))
	require.Equal(t, "logical", desc.BackupType)
	require.Equal(t, "dev", desc.Connection)
	require.Equal(t, "testdb", desc.Database)
	require.Greater(t, desc.Size, int64(0))
	require.NotEmpty(t, desc.Checksum)
	require.NotEmpty(t, desc.ID)

	// Verify the manifest was written.
	wd, _ := sb.Getwd()
	manifestPath := filepath.Join(wd, desc.ID, siphon.ManifestFilename)
	manifest, err := siphon.ReadManifest(sb.Runtime(), manifestPath)
	require.NoError(t, err)
	require.Equal(t, "1", manifest.Version)
	require.Equal(t, "mariadb", manifest.Engine)
	require.Equal(t, "logical", manifest.Type)
	require.Len(t, manifest.Files, 1)
	require.Equal(t, "dump.sql", manifest.Files[0].Path)
}

func TestBackup_LogicalDumpWithCompression(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphonWithMock(t)
	ctx := context.Background()

	desc, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Compress:   "gzip",
	})
	require.NoError(t, err)
	require.NotNil(t, desc)
	require.Greater(t, desc.Size, int64(0))
}

func TestBackup_FileBackup(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphonWithMock(t)
	ctx := context.Background()

	desc, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "local",
		BackupType: "file",
	})
	require.NoError(t, err)
	require.NotNil(t, desc)
	require.Equal(t, "sqlite", string(desc.Engine))
	require.Equal(t, "file", desc.BackupType)
	require.Greater(t, desc.Size, int64(0))
}

func TestBackup_WithCustomName(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphonWithMock(t)
	ctx := context.Background()

	desc, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Name:       "my-custom-backup",
	})
	require.NoError(t, err)
	require.Equal(t, "my-custom-backup", desc.ID)
}

func TestBackup_WithRepo(t *testing.T) {
	t.Parallel()
	s, sb := newTestSiphonWithMock(t)
	ctx := context.Background()

	// Create repo directory.
	sb.Mkdir("/home/testuser/backups/nightly", true)

	desc, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Repo:       "nightly",
	})
	require.NoError(t, err)
	require.NotNil(t, desc)
}

func TestBackup_ConnectionNotFound(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphonWithMock(t)
	ctx := context.Background()

	_, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "nonexistent",
		BackupType: "logical",
	})
	require.ErrorIs(t, err, siphon.ErrConnectionNotFound)
}

func TestBackup_UnsupportedCapability(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphonWithMock(t)
	ctx := context.Background()

	// SQLite does not support physical backup in production, but mock does.
	// Let's test with a mock that has the engine type but we request an
	// unsupported type. Actually the mock implements all interfaces. Let me
	// test the checkBackupCapability path directly by using the factory
	// to return an adaptor that doesn't implement the required interface.
	s.AdaptorFactory = func(cfg *engine.ConnectionConfig) (engine.Adaptor, error) {
		// Return a real SQLite adaptor which doesn't support logical backup.
		return engine.NewSQLiteAdaptor(cfg), nil
	}

	_, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "local",
		BackupType: "logical", // SQLite doesn't support logical
	})
	require.ErrorIs(t, err, siphon.ErrUnsupportedCapability)
}

func TestBackupInfo(t *testing.T) {
	t.Parallel()
	s, sb := newTestSiphonWithMock(t)
	ctx := context.Background()

	// First create a backup.
	desc, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Label:      "test-label",
	})
	require.NoError(t, err)

	// Now retrieve its info.
	wd, _ := sb.Getwd()
	backupPath := filepath.Join(wd, desc.ID)
	info, err := s.BackupInfo(ctx, &siphon.BackupInfoOptions{
		Path: backupPath,
	})
	require.NoError(t, err)
	require.Equal(t, desc.ID, info.ID)
	require.Equal(t, "test-label", info.Label)
	require.Equal(t, "mariadb", string(info.Engine))
	require.Equal(t, "testdb", info.Database)
}

func TestListBackups(t *testing.T) {
	t.Parallel()
	s, sb := newTestSiphonWithMock(t)
	ctx := context.Background()

	// Create repo dir.
	sb.Mkdir("/home/testuser/backups/nightly", true)

	// Create two backups in the repo.
	_, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Repo:       "nightly",
		Name:       "backup-one",
	})
	require.NoError(t, err)

	// Advance clock so second backup has different timestamp.
	sb.Advance(60_000_000_000) // 1 minute

	_, err = s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Repo:       "nightly",
		Name:       "backup-two",
	})
	require.NoError(t, err)

	// List all backups in repo.
	backups, err := s.ListBackups(ctx, &siphon.ListBackupsOptions{
		Repo: "nightly",
	})
	require.NoError(t, err)
	require.Len(t, backups, 2)
	// Should be sorted newest first.
	require.Equal(t, "backup-two", backups[0].ID)
	require.Equal(t, "backup-one", backups[1].ID)
}

func TestListBackups_FilterByConnection(t *testing.T) {
	t.Parallel()
	s, sb := newTestSiphonWithMock(t)
	ctx := context.Background()

	sb.Mkdir("/home/testuser/backups/nightly", true)

	_, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Repo:       "nightly",
		Name:       "dev-backup",
	})
	require.NoError(t, err)

	_, err = s.Backup(ctx, &siphon.BackupOptions{
		Connection: "local",
		BackupType: "file",
		Repo:       "nightly",
		Name:       "local-backup",
	})
	require.NoError(t, err)

	backups, err := s.ListBackups(ctx, &siphon.ListBackupsOptions{
		Repo:       "nightly",
		Connection: "dev",
	})
	require.NoError(t, err)
	require.Len(t, backups, 1)
	require.Equal(t, "dev", backups[0].Connection)
}

func TestListBackups_Limit(t *testing.T) {
	t.Parallel()
	s, sb := newTestSiphonWithMock(t)
	ctx := context.Background()

	sb.Mkdir("/home/testuser/backups/nightly", true)

	for i := 0; i < 3; i++ {
		_, err := s.Backup(ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Repo:       "nightly",
			Name:       "backup-" + string(rune('a'+i)),
		})
		require.NoError(t, err)
	}

	backups, err := s.ListBackups(ctx, &siphon.ListBackupsOptions{
		Repo:  "nightly",
		Limit: 2,
	})
	require.NoError(t, err)
	require.Len(t, backups, 2)
}

func TestVerifyBackup_Success(t *testing.T) {
	t.Parallel()
	s, sb := newTestSiphonWithMock(t)
	ctx := context.Background()

	desc, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
	})
	require.NoError(t, err)

	wd, _ := sb.Getwd()
	backupPath := filepath.Join(wd, desc.ID)

	err = s.VerifyBackup(ctx, &siphon.VerifyBackupOptions{
		Path: backupPath,
	})
	require.NoError(t, err)
}

func TestVerifyBackup_CorruptedFile(t *testing.T) {
	t.Parallel()
	s, sb := newTestSiphonWithMock(t)
	ctx := context.Background()

	desc, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
	})
	require.NoError(t, err)

	wd, _ := sb.Getwd()
	backupPath := filepath.Join(wd, desc.ID)

	// Corrupt the dump file.
	dumpPath := filepath.Join(backupPath, "dump.sql")
	sb.MustWriteFile(dumpPath, []byte("corrupted data"), 0o644)

	err = s.VerifyBackup(ctx, &siphon.VerifyBackupOptions{
		Path: backupPath,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksum mismatch")
}

func TestVerifyBackup_MissingManifest(t *testing.T) {
	t.Parallel()
	s, sb := newTestSiphonWithMock(t)
	ctx := context.Background()

	sb.Mkdir("/home/testuser/empty-backup", true)

	err := s.VerifyBackup(ctx, &siphon.VerifyBackupOptions{
		Path: "/home/testuser/empty-backup",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, siphon.ErrManifestNotFound)
}

func TestLabelBackup(t *testing.T) {
	t.Parallel()
	s, sb := newTestSiphonWithMock(t)
	ctx := context.Background()

	desc, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
	})
	require.NoError(t, err)

	wd, _ := sb.Getwd()
	backupPath := filepath.Join(wd, desc.ID)

	// Set label.
	err = s.LabelBackup(ctx, &siphon.LabelBackupOptions{
		BackupID: backupPath,
		Label:    "production",
	})
	require.NoError(t, err)

	// Verify label was set.
	info, err := s.BackupInfo(ctx, &siphon.BackupInfoOptions{
		Path: backupPath,
	})
	require.NoError(t, err)
	require.Equal(t, "production", info.Label)
}

func TestLabelBackup_UpdateExisting(t *testing.T) {
	t.Parallel()
	s, sb := newTestSiphonWithMock(t)
	ctx := context.Background()

	desc, err := s.Backup(ctx, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Label:      "initial",
	})
	require.NoError(t, err)

	wd, _ := sb.Getwd()
	backupPath := filepath.Join(wd, desc.ID)

	// Update label.
	err = s.LabelBackup(ctx, &siphon.LabelBackupOptions{
		BackupID: backupPath,
		Label:    "updated",
	})
	require.NoError(t, err)

	info, err := s.BackupInfo(ctx, &siphon.BackupInfoOptions{
		Path: backupPath,
	})
	require.NoError(t, err)
	require.Equal(t, "updated", info.Label)
}

// Helper: compute sha256 hex for verifying test expectations.
func testSHA256(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
