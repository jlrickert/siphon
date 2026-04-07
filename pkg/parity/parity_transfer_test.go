package parity_test

import (
	"context"
	"testing"

	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/stretchr/testify/require"
)

// setupTransferParityEnv creates a parity env with mock adaptor factory that
// supports transfer operations.
func setupTransferParityEnv(t *testing.T) (*parityEnv, *transferParityRecord) {
	t.Helper()
	env := newParityEnv(t)

	rec := &transferParityRecord{}

	env.siphon.AdaptorFactory = func(cfg *engine.ConnectionConfig) (engine.Adaptor, error) {
		mock := engine.NewMockAdaptor(cfg.Engine)
		mock.Tables = []string{"users", "orders", "products"}
		mock.ForeignKeys = []engine.ForeignKey{
			{Table: "orders", ReferencedTable: "users"},
		}
		mock.TransferTablesFn = func(ctx context.Context, opts engine.TransferOptions) error {
			rec.called = true
			rec.opts = opts
			return nil
		}
		return mock, nil
	}

	return env, rec
}

type transferParityRecord struct {
	called bool
	opts   engine.TransferOptions
}

func TestParity_Transfer(t *testing.T) {
	t.Parallel()

	t.Run("CLI transfer completes", func(t *testing.T) {
		t.Parallel()
		env, rec := setupTransferParityEnv(t)

		out, err := runCLIWithSiphon(t, env, "transfer", "dev:testdb", "local:localdb",
			"--tables", "users,orders",
			"--on-conflict", "overwrite")
		require.NoError(t, err)
		require.Contains(t, out, "Transfer completed")
		require.True(t, rec.called)
		require.Equal(t, "overwrite", rec.opts.OnConflict)
	})

	t.Run("MCP transfer completes", func(t *testing.T) {
		t.Parallel()
		env, rec := setupTransferParityEnv(t)

		out, err := env.runMCP("transfer", map[string]any{
			"source":      "dev:testdb",
			"target":      "local:localdb",
			"tables":      []any{"users", "orders"},
			"on_conflict": "skip",
			"confirm":     true,
		})
		require.NoError(t, err)
		require.Contains(t, out, "transfer completed")
		require.True(t, rec.called)
		require.Equal(t, "skip", rec.opts.OnConflict)
	})

	t.Run("CLI and MCP both error on missing source", func(t *testing.T) {
		t.Parallel()
		env, _ := setupTransferParityEnv(t)

		_, cliErr := runCLIWithSiphon(t, env, "transfer", "nonexistent:db", "local:localdb")
		require.Error(t, cliErr)
		require.Contains(t, cliErr.Error(), "not found")

		_, mcpErr := env.runMCP("transfer", map[string]any{
			"source":  "nonexistent:db",
			"target":  "local:localdb",
			"confirm": true,
		})
		require.Error(t, mcpErr)
		require.Contains(t, mcpErr.Error(), "not found")
	})

	t.Run("CLI transfer with all-tables flag", func(t *testing.T) {
		t.Parallel()
		env, rec := setupTransferParityEnv(t)

		out, err := runCLIWithSiphon(t, env, "transfer", "dev:testdb", "local:localdb",
			"--all-tables")
		require.NoError(t, err)
		require.Contains(t, out, "Transfer completed")
		require.True(t, rec.called)
		// Should include all tables.
		require.ElementsMatch(t, []string{"users", "orders", "products"}, rec.opts.Tables)
	})

	t.Run("MCP transfer with all_tables", func(t *testing.T) {
		t.Parallel()
		env, rec := setupTransferParityEnv(t)

		out, err := env.runMCP("transfer", map[string]any{
			"source":     "dev:testdb",
			"target":     "local:localdb",
			"all_tables": true,
			"confirm":    true,
		})
		require.NoError(t, err)
		require.Contains(t, out, "transfer completed")
		require.True(t, rec.called)
		require.ElementsMatch(t, []string{"users", "orders", "products"}, rec.opts.Tables)
	})

	t.Run("Transfer respects FK ordering", func(t *testing.T) {
		t.Parallel()
		env, rec := setupTransferParityEnv(t)

		_, err := runCLIWithSiphon(t, env, "transfer", "dev:testdb", "local:localdb",
			"--tables", "users,orders")
		require.NoError(t, err)
		require.True(t, rec.called)

		// users must come before orders (orders depends on users).
		indexOf := make(map[string]int)
		for i, table := range rec.opts.Tables {
			indexOf[table] = i
		}
		require.Less(t, indexOf["users"], indexOf["orders"])
	})
}

func TestParity_Transfer_DefaultOnConflict(t *testing.T) {
	t.Parallel()
	env, rec := setupTransferParityEnv(t)

	// No --on-conflict flag: should default to "skip".
	_, err := runCLIWithSiphon(t, env, "transfer", "dev:testdb", "local:localdb",
		"--tables", "users")
	require.NoError(t, err)
	require.True(t, rec.called)
	require.Equal(t, "skip", rec.opts.OnConflict)
}

func TestParity_Transfer_PolicyDenied(t *testing.T) {
	t.Parallel()

	t.Run("CLI denied by policy", func(t *testing.T) {
		t.Parallel()
		env, _ := setupTransferParityEnv(t)

		// Write a config that denies the local connection for CLI.
		configDir := "/home/testuser/.config/siphon"
		env.sb.WriteFile(configDir+"/config.yaml", []byte(`
version: "1"
default_connection: dev
connections:
  dev:
    name: dev
    engine: mariadb
    host: localhost
    database: testdb
  local:
    name: local
    engine: sqlite
    path: /home/testuser/data/local.db
policies:
  local:
    allow_transfer: false
repos:
  nightly:
    path: /home/testuser/backups/nightly
`), 0o644)
		env.siphon.ConfigService.ResetCache()

		_, err := runCLIWithSiphon(t, env, "transfer", "dev:testdb", "local:localdb")
		require.Error(t, err)
		require.Contains(t, err.Error(), "denied")
	})

	t.Run("MCP denied by policy", func(t *testing.T) {
		t.Parallel()
		env, _ := setupTransferParityEnv(t)

		configDir := "/home/testuser/.config/siphon"
		env.sb.WriteFile(configDir+"/config.yaml", []byte(`
version: "1"
default_connection: dev
connections:
  dev:
    name: dev
    engine: mariadb
    host: localhost
    database: testdb
  local:
    name: local
    engine: sqlite
    path: /home/testuser/data/local.db
policies:
  local:
    allow_transfer: false
repos:
  nightly:
    path: /home/testuser/backups/nightly
`), 0o644)
		env.siphon.ConfigService.ResetCache()

		_, err := env.runMCP("transfer", map[string]any{
			"source":  "dev:testdb",
			"target":  "local:localdb",
			"confirm": true,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "denied")
	})
}

func TestParity_Transfer_MCPConfirmRequired(t *testing.T) {
	t.Parallel()

	env, _ := setupTransferParityEnv(t)

	// Write a config that requires confirmation for MCP on local.
	configDir := "/home/testuser/.config/siphon"
	env.sb.WriteFile(configDir+"/config.yaml", []byte(`
version: "1"
default_connection: dev
connections:
  dev:
    name: dev
    engine: mariadb
    host: localhost
    database: testdb
  local:
    name: local
    engine: sqlite
    path: /home/testuser/data/local.db
policies:
  local:
    mcp: confirm
repos:
  nightly:
    path: /home/testuser/backups/nightly
`), 0o644)
	env.siphon.ConfigService.ResetCache()

	// Without confirm.
	_, err := env.runMCP("transfer", map[string]any{
		"source": "dev:testdb",
		"target": "local:localdb",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "confirmation required")

	// With confirm.
	out, err := env.runMCP("transfer", map[string]any{
		"source":  "dev:testdb",
		"target":  "local:localdb",
		"confirm": true,
	})
	require.NoError(t, err)
	require.Contains(t, out, "transfer completed")
}
