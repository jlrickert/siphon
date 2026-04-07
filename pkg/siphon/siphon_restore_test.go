package siphon_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// newTestSiphonWithRestoreMock creates a test Siphon where both backup and
// restore mock hooks are wired. The mock adaptor's DumpFn writes test data,
// and LoadDumpFn / CopyRestoreFn / PrepareFn / CopyBackFn all track calls.
func newTestSiphonWithRestoreMock(t *testing.T) (*siphon.Siphon, *sandbox.Sandbox, *restoreCalls) {
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

	calls := &restoreCalls{}

	s.AdaptorFactory = func(cfg *engine.ConnectionConfig) (engine.Adaptor, error) {
		mock := engine.NewMockAdaptor(cfg.Engine)

		// Backup hooks (for creating test backups).
		mock.DumpFn = func(ctx context.Context, opts engine.DumpOptions) error {
			if opts.Output != nil {
				_, _ = opts.Output.Write([]byte("-- mock dump data\n"))
			}
			return nil
		}
		mock.CopyBackupFn = func(ctx context.Context, opts engine.CopyBackupOptions) error {
			return sb.WriteFile(opts.DestPath, []byte("mock sqlite data"), 0o644)
		}

		// Restore hooks.
		mock.LoadDumpFn = func(ctx context.Context, opts engine.LoadDumpOptions) error {
			calls.LoadDumpCalled = true
			calls.LoadDumpDatabase = opts.Database
			return nil
		}
		mock.CopyRestoreFn = func(ctx context.Context, opts engine.CopyRestoreOptions) error {
			calls.CopyRestoreCalled = true
			calls.CopyRestoreSource = opts.SourcePath
			calls.CopyRestoreDest = opts.DestPath
			calls.CopyRestoreForce = opts.Force
			return nil
		}
		mock.PrepareFn = func(ctx context.Context, opts engine.PrepareOptions) error {
			calls.PrepareCalled = true
			calls.PrepareDir = opts.TargetDir
			return nil
		}
		mock.CopyBackFn = func(ctx context.Context, opts engine.CopyBackOptions) error {
			calls.CopyBackCalled = true
			calls.CopyBackSource = opts.SourceDir
			calls.CopyBackDataDir = opts.DataDir
			return nil
		}

		return mock, nil
	}

	return s, sb, calls
}

// restoreCalls tracks which mock methods were called and with what arguments.
type restoreCalls struct {
	LoadDumpCalled   bool
	LoadDumpDatabase string

	CopyRestoreCalled bool
	CopyRestoreSource string
	CopyRestoreDest   string
	CopyRestoreForce  bool

	PrepareCalled bool
	PrepareDir    string

	CopyBackCalled  bool
	CopyBackSource  string
	CopyBackDataDir string
}

// createTestBackup creates a backup in the sandbox and returns the path.
func createTestBackup(t *testing.T, s *siphon.Siphon, sb *sandbox.Sandbox, opts *siphon.BackupOptions) string {
	t.Helper()
	ctx := context.Background()

	desc, err := s.Backup(ctx, opts)
	require.NoError(t, err)

	wd, _ := sb.Getwd()
	return filepath.Join(wd, desc.ID)
}

func TestRestore_LogicalBackup(t *testing.T) {
	t.Parallel()
	s, sb, calls := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	// Create a logical backup first.
	backupPath := createTestBackup(t, s, sb, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Name:       "restore-test",
	})

	// Restore it.
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "dev",
		Path:       backupPath,
		Force:      true,
	})
	require.NoError(t, err)
	require.True(t, calls.LoadDumpCalled, "LoadDump should have been called")
	require.Equal(t, "testdb", calls.LoadDumpDatabase)
}

func TestRestore_RequiresForce(t *testing.T) {
	t.Parallel()
	s, sb, _ := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	backupPath := createTestBackup(t, s, sb, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Name:       "force-test",
	})

	// Attempt restore without --force.
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "dev",
		Path:       backupPath,
		Force:      false,
	})
	require.ErrorIs(t, err, siphon.ErrForceRequired)
}

func TestRestore_DatabaseRedirect(t *testing.T) {
	t.Parallel()
	s, sb, calls := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	backupPath := createTestBackup(t, s, sb, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Name:       "redirect-test",
	})

	// Restore to a different database.
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "dev",
		Path:       backupPath,
		Force:      true,
		Database:   "otherdb",
	})
	require.NoError(t, err)
	require.True(t, calls.LoadDumpCalled)
	require.Equal(t, "otherdb", calls.LoadDumpDatabase)
}

func TestRestore_TablesPartialRestore(t *testing.T) {
	t.Parallel()
	s, sb, calls := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	backupPath := createTestBackup(t, s, sb, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Name:       "tables-test",
	})

	// Restore with --tables should succeed for logical backups.
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "dev",
		Path:       backupPath,
		Force:      true,
		Tables:     []string{"users", "orders"},
	})
	require.NoError(t, err)
	require.True(t, calls.LoadDumpCalled)
}

func TestRestore_ManifestDiscovery(t *testing.T) {
	t.Parallel()
	s, sb, calls := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	// Create a backup (manifest is auto-generated).
	backupPath := createTestBackup(t, s, sb, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Name:       "manifest-test",
	})

	// Restore without --type; should auto-detect "logical" from manifest.
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "dev",
		Path:       backupPath,
		Force:      true,
	})
	require.NoError(t, err)
	require.True(t, calls.LoadDumpCalled)
}

func TestRestore_TypeOverride(t *testing.T) {
	t.Parallel()
	s, sb, _ := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	// Create a backup directory manually with no manifest but with a dump file.
	wd, _ := sb.Getwd()
	backupDir := filepath.Join(wd, "override-test")
	sb.Mkdir(backupDir, true)
	sb.MustWriteFile(filepath.Join(backupDir, "dump.sql"), []byte("-- data\n"), 0o644)

	// Restore with explicit --type (no manifest).
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "dev",
		Path:       backupDir,
		Force:      true,
		Type:       "logical",
	})
	require.NoError(t, err)
}

func TestRestore_PhysicalTablesRejected(t *testing.T) {
	t.Parallel()
	s, sb, _ := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	// Create a physical backup.
	backupPath := createTestBackup(t, s, sb, &siphon.BackupOptions{
		Connection: "dev",
		BackupType: "physical",
		Name:       "physical-tables-test",
	})

	// Attempting partial restore of physical backup should fail.
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "dev",
		Path:       backupPath,
		Force:      true,
		Tables:     []string{"users"},
	})
	require.ErrorIs(t, err, siphon.ErrUnsupportedRestoreTarget)
}

func TestRestore_CrossConnection(t *testing.T) {
	t.Parallel()
	s, sb, calls := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	// Create a backup from the "dev" connection.
	backupPath := createTestBackup(t, s, sb, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Name:       "cross-conn-test",
	})

	// Restore to the "local" connection using --type override since SQLite
	// doesn't natively support logical. But our mock implements all interfaces.
	// We use the mock adaptor which implements LogicalBackupAdaptor.
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "local",
		Path:       backupPath,
		Force:      true,
	})
	require.NoError(t, err)
	require.True(t, calls.LoadDumpCalled)
}

func TestRestore_PolicyDeny(t *testing.T) {
	t.Parallel()
	s, sb, _ := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	backupPath := createTestBackup(t, s, sb, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Name:       "policy-test",
	})

	// Write a policy config that denies restore on "dev".
	policyConfig := map[string]interface{}{
		"version":            "1",
		"default_connection": "dev",
		"connections": map[string]interface{}{
			"dev": map[string]interface{}{
				"name":     "dev",
				"engine":   "mariadb",
				"host":     "localhost",
				"port":     3306,
				"user":     "root",
				"database": "testdb",
			},
			"local": map[string]interface{}{
				"name":   "local",
				"engine": "sqlite",
				"path":   "/home/testuser/data/local.db",
			},
		},
		"repos": map[string]interface{}{
			"nightly": map[string]interface{}{
				"path": "/home/testuser/backups/nightly",
			},
		},
		"policies": map[string]interface{}{
			"dev": map[string]interface{}{
				"cli": "deny",
			},
		},
	}
	configData, _ := yaml.Marshal(policyConfig)
	sb.MustWriteFile("/home/testuser/.config/siphon/config.yaml", configData, 0o644)

	// Reset config cache to pick up the new policy.
	s.ConfigService.ResetCache()

	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "dev",
		Path:       backupPath,
		Force:      true,
		Surface:    siphon.SurfaceCLI,
	})
	require.ErrorIs(t, err, siphon.ErrOperationDenied)
}

func TestRestore_FileBackup(t *testing.T) {
	t.Parallel()
	s, sb, calls := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	// Create a file backup from the "local" connection.
	backupPath := createTestBackup(t, s, sb, &siphon.BackupOptions{
		Connection: "local",
		BackupType: "file",
		Name:       "file-restore-test",
	})

	// Restore it.
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "local",
		Path:       backupPath,
		Force:      true,
	})
	require.NoError(t, err)
	require.True(t, calls.CopyRestoreCalled)
	require.Equal(t, "/home/testuser/data/local.db", calls.CopyRestoreDest)
	require.True(t, calls.CopyRestoreForce)
}

func TestRestore_BackupTypeMismatch(t *testing.T) {
	t.Parallel()
	s, sb, _ := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	// Create a logical backup.
	backupPath := createTestBackup(t, s, sb, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Name:       "mismatch-test",
	})

	// Attempt restore with --type=physical (mismatches manifest).
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "dev",
		Path:       backupPath,
		Force:      true,
		Type:       "physical",
	})
	require.ErrorIs(t, err, siphon.ErrBackupTypeMismatch)
}

func TestRestore_CompressedBackup(t *testing.T) {
	t.Parallel()
	s, sb, calls := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	// Create a compressed backup.
	backupPath := createTestBackup(t, s, sb, &siphon.BackupOptions{
		Connection: "dev",
		Database:   "testdb",
		BackupType: "logical",
		Compress:   "gzip",
		Name:       "compressed-test",
	})

	// Restore should decompress automatically.
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "dev",
		Path:       backupPath,
		Force:      true,
	})
	require.NoError(t, err)
	require.True(t, calls.LoadDumpCalled)
}

func TestRestore_MissingManifestNoType(t *testing.T) {
	t.Parallel()
	s, sb, _ := newTestSiphonWithRestoreMock(t)
	ctx := context.Background()

	// Create a directory with no manifest.
	wd, _ := sb.Getwd()
	backupDir := filepath.Join(wd, "no-manifest-test")
	sb.Mkdir(backupDir, true)
	sb.MustWriteFile(filepath.Join(backupDir, "dump.sql"), []byte("-- data\n"), 0o644)

	// Restore without --type should fail because no manifest to discover type.
	err := s.Restore(ctx, &siphon.RestoreOptions{
		Connection: "dev",
		Path:       backupDir,
		Force:      true,
	})
	require.ErrorIs(t, err, siphon.ErrManifestNotFound)
}
