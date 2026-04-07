package parity_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

// setupRestoreParityEnv creates a parity env with mock adaptor factory that
// supports both backup creation and restore operations.
func setupRestoreParityEnv(t *testing.T) *parityEnv {
	t.Helper()
	env := newParityEnv(t)

	env.siphon.AdaptorFactory = func(cfg *engine.ConnectionConfig) (engine.Adaptor, error) {
		mock := engine.NewMockAdaptor(cfg.Engine)
		mock.DumpFn = func(ctx context.Context, opts engine.DumpOptions) error {
			if opts.Output != nil {
				_, _ = opts.Output.Write([]byte("-- parity restore dump\n"))
			}
			return nil
		}
		mock.CopyBackupFn = func(ctx context.Context, opts engine.CopyBackupOptions) error {
			return env.sb.WriteFile(opts.DestPath, []byte("mock sqlite parity data"), 0o644)
		}
		// Restore hooks (no-op for parity validation).
		mock.LoadDumpFn = func(ctx context.Context, opts engine.LoadDumpOptions) error {
			return nil
		}
		mock.CopyRestoreFn = func(ctx context.Context, opts engine.CopyRestoreOptions) error {
			return nil
		}
		mock.PrepareFn = func(ctx context.Context, opts engine.PrepareOptions) error {
			return nil
		}
		mock.CopyBackFn = func(ctx context.Context, opts engine.CopyBackOptions) error {
			return nil
		}
		return mock, nil
	}

	return env
}

func TestParity_Restore(t *testing.T) {
	t.Parallel()

	t.Run("CLI restores from logical backup", func(t *testing.T) {
		t.Parallel()
		env := setupRestoreParityEnv(t)

		// Create a backup first.
		desc, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Name:       "cli-restore-test",
		})
		require.NoError(t, err)

		wd, _ := env.sb.Getwd()
		backupPath := filepath.Join(wd, desc.ID)

		out, err := runCLIWithSiphon(t, env, "restore", "dev", backupPath, "--force")
		require.NoError(t, err)
		require.Contains(t, out, "Restore completed")
	})

	t.Run("MCP restores from logical backup", func(t *testing.T) {
		t.Parallel()
		env := setupRestoreParityEnv(t)

		desc, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Name:       "mcp-restore-test",
		})
		require.NoError(t, err)

		wd, _ := env.sb.Getwd()
		backupPath := filepath.Join(wd, desc.ID)

		out, err := env.runMCP("restore", map[string]any{
			"connection": "dev",
			"path":       backupPath,
			"force":      true,
			"confirm":    true,
		})
		require.NoError(t, err)
		require.Contains(t, out, "restore completed")
	})

	t.Run("CLI requires --force", func(t *testing.T) {
		t.Parallel()
		env := setupRestoreParityEnv(t)

		desc, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Name:       "force-test",
		})
		require.NoError(t, err)

		wd, _ := env.sb.Getwd()
		backupPath := filepath.Join(wd, desc.ID)

		_, err = runCLIWithSiphon(t, env, "restore", "dev", backupPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "force")
	})

	t.Run("MCP requires force", func(t *testing.T) {
		t.Parallel()
		env := setupRestoreParityEnv(t)

		desc, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Name:       "mcp-force-test",
		})
		require.NoError(t, err)

		wd, _ := env.sb.Getwd()
		backupPath := filepath.Join(wd, desc.ID)

		_, err = env.runMCP("restore", map[string]any{
			"connection": "dev",
			"path":       backupPath,
			"confirm":    true,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "force")
	})
}
