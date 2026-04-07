package parity_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/jlrickert/siphon/pkg/cli"
	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

// setupBackupParityEnv creates a parity env with mock adaptor factory and
// a pre-created backup for tests that need one.
func setupBackupParityEnv(t *testing.T) *parityEnv {
	t.Helper()
	env := newParityEnv(t)

	// Override the adaptor factory with a mock.
	env.siphon.AdaptorFactory = func(cfg *engine.ConnectionConfig) (engine.Adaptor, error) {
		mock := engine.NewMockAdaptor(cfg.Engine)
		mock.DumpFn = func(ctx context.Context, opts engine.DumpOptions) error {
			if opts.Output != nil {
				_, _ = opts.Output.Write([]byte("-- parity mock dump\n"))
			}
			return nil
		}
		mock.CopyBackupFn = func(ctx context.Context, opts engine.CopyBackupOptions) error {
			return env.sb.WriteFile(opts.DestPath, []byte("mock sqlite parity data"), 0o644)
		}
		return mock, nil
	}

	return env
}

// runCLIWithSiphon runs a CLI command through Cobra using a pre-configured
// Siphon instance (with mock factory). This avoids creating a fresh Siphon
// which would try to connect to real databases.
func runCLIWithSiphon(t *testing.T, env *parityEnv, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	deps := &cli.Deps{
		Runtime: env.sb.Runtime(),
		Siphon:  env.siphon,
	}
	cmd := cli.NewRootCmd(deps)
	cmd.SetArgs(args)
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	err := cmd.ExecuteContext(env.ctx)
	return buf.String(), err
}

func TestParity_BackupCreate(t *testing.T) {
	t.Parallel()

	t.Run("CLI creates backup", func(t *testing.T) {
		t.Parallel()
		env := setupBackupParityEnv(t)

		out, err := runCLIWithSiphon(t, env, "backup", "create", "dev", "--type", "logical", "--name", "cli-backup")
		require.NoError(t, err)
		require.Contains(t, out, "Backup created")
		require.Contains(t, out, "cli-backup")
	})

	t.Run("MCP creates backup", func(t *testing.T) {
		t.Parallel()
		env := setupBackupParityEnv(t)

		out, err := env.runMCP("backup_create", map[string]any{
			"connection":  "dev",
			"backup_type": "logical",
			"name":        "mcp-backup",
		})
		require.NoError(t, err)

		var desc siphon.BackupDescriptor
		require.NoError(t, json.Unmarshal([]byte(out), &desc))
		require.Equal(t, "mcp-backup", desc.ID)
		require.Equal(t, "mariadb", string(desc.Engine))
	})
}

func TestParity_BackupInfo(t *testing.T) {
	t.Parallel()

	t.Run("CLI shows info", func(t *testing.T) {
		t.Parallel()
		env := setupBackupParityEnv(t)

		desc, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Name:       "info-test",
		})
		require.NoError(t, err)

		wd, _ := env.sb.Getwd()
		backupPath := filepath.Join(wd, desc.ID)

		out, err := runCLIWithSiphon(t, env, "backup", "info", backupPath)
		require.NoError(t, err)
		require.Contains(t, out, "mariadb")
		require.Contains(t, out, "logical")
	})

	t.Run("MCP shows info", func(t *testing.T) {
		t.Parallel()
		env := setupBackupParityEnv(t)

		desc, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Name:       "info-test",
		})
		require.NoError(t, err)

		wd, _ := env.sb.Getwd()
		backupPath := filepath.Join(wd, desc.ID)

		out, err := env.runMCP("backup_info", map[string]any{
			"path": backupPath,
		})
		require.NoError(t, err)

		var info siphon.BackupDescriptor
		require.NoError(t, json.Unmarshal([]byte(out), &info))
		require.Equal(t, "info-test", info.ID)
	})
}

func TestParity_BackupList(t *testing.T) {
	t.Parallel()

	t.Run("CLI lists backups", func(t *testing.T) {
		t.Parallel()
		env := setupBackupParityEnv(t)

		env.sb.Mkdir("/home/testuser/backups/nightly", true)

		_, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Repo:       "nightly",
			Name:       "list-cli-backup",
		})
		require.NoError(t, err)

		out, err := runCLIWithSiphon(t, env, "backup", "list", "--repo", "nightly")
		require.NoError(t, err)
		require.Contains(t, out, "list-cli-backup")
	})

	t.Run("MCP lists backups", func(t *testing.T) {
		t.Parallel()
		env := setupBackupParityEnv(t)

		env.sb.Mkdir("/home/testuser/backups/nightly", true)

		_, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Repo:       "nightly",
			Name:       "list-mcp-backup",
		})
		require.NoError(t, err)

		out, err := env.runMCP("backup_list", map[string]any{
			"repo": "nightly",
		})
		require.NoError(t, err)

		var backups []siphon.BackupDescriptor
		require.NoError(t, json.Unmarshal([]byte(out), &backups))
		require.Len(t, backups, 1)
		require.Equal(t, "list-mcp-backup", backups[0].ID)
	})
}

func TestParity_BackupVerify(t *testing.T) {
	t.Parallel()

	t.Run("CLI verifies backup", func(t *testing.T) {
		t.Parallel()
		env := setupBackupParityEnv(t)

		desc, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Name:       "verify-test",
		})
		require.NoError(t, err)

		wd, _ := env.sb.Getwd()
		backupPath := filepath.Join(wd, desc.ID)

		out, err := runCLIWithSiphon(t, env, "backup", "verify", backupPath)
		require.NoError(t, err)
		require.Contains(t, out, "OK")
	})

	t.Run("MCP verifies backup", func(t *testing.T) {
		t.Parallel()
		env := setupBackupParityEnv(t)

		desc, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Name:       "verify-test",
		})
		require.NoError(t, err)

		wd, _ := env.sb.Getwd()
		backupPath := filepath.Join(wd, desc.ID)

		out, err := env.runMCP("backup_verify", map[string]any{
			"path": backupPath,
		})
		require.NoError(t, err)
		require.Contains(t, out, "verified")
	})
}

func TestParity_BackupLabel(t *testing.T) {
	t.Parallel()

	t.Run("CLI labels backup", func(t *testing.T) {
		t.Parallel()
		env := setupBackupParityEnv(t)

		desc, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Name:       "label-test",
		})
		require.NoError(t, err)

		wd, _ := env.sb.Getwd()
		backupPath := filepath.Join(wd, desc.ID)

		out, err := runCLIWithSiphon(t, env, "backup", "label", backupPath, "production")
		require.NoError(t, err)
		require.Contains(t, out, "production")
	})

	t.Run("MCP labels backup", func(t *testing.T) {
		t.Parallel()
		env := setupBackupParityEnv(t)

		desc, err := env.siphon.Backup(env.ctx, &siphon.BackupOptions{
			Connection: "dev",
			Database:   "testdb",
			BackupType: "logical",
			Name:       "label-test",
		})
		require.NoError(t, err)

		wd, _ := env.sb.Getwd()
		backupPath := filepath.Join(wd, desc.ID)

		out, err := env.runMCP("backup_label", map[string]any{
			"backup_id": backupPath,
			"label":     "staging",
		})
		require.NoError(t, err)
		require.Contains(t, out, "updated")
	})
}
