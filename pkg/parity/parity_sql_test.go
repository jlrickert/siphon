package parity_test

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/jlrickert/siphon/pkg/cli"
	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/stretchr/testify/require"
)

// setupSQLParityEnv creates a parity env with mock adaptor factory that
// supports query operations.
func setupSQLParityEnv(t *testing.T) *parityEnv {
	t.Helper()
	env := newParityEnv(t)

	env.siphon.AdaptorFactory = func(cfg *engine.ConnectionConfig) (engine.Adaptor, error) {
		mock := engine.NewMockAdaptor(cfg.Engine)
		mock.QueryFn = func(ctx context.Context, query string, args ...any) (*engine.QueryResult, error) {
			return &engine.QueryResult{
				Columns: []string{"id", "name"},
				Rows: [][]string{
					{"1", "Alice"},
					{"2", "Bob"},
				},
			}, nil
		}
		mock.ExecuteFn = func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			return &sqlParityResult{affected: 1}, nil
		}
		return mock, nil
	}

	return env
}

type sqlParityResult struct {
	affected int64
}

func (r *sqlParityResult) LastInsertId() (int64, error) { return 0, nil }
func (r *sqlParityResult) RowsAffected() (int64, error) { return r.affected, nil }

var _ driver.Result = (*sqlParityResult)(nil)

func TestParity_SQL_Select(t *testing.T) {
	t.Parallel()

	t.Run("CLI SELECT returns rows", func(t *testing.T) {
		t.Parallel()
		env := setupSQLParityEnv(t)

		out, err := runSQLCLIWithSiphon(t, env, "sql", "dev", "SELECT * FROM users", "--format", "csv")
		require.NoError(t, err)
		require.Contains(t, out, "id,name")
		require.Contains(t, out, "1,Alice")
		require.Contains(t, out, "2,Bob")
	})

	t.Run("MCP SELECT returns rows", func(t *testing.T) {
		t.Parallel()
		env := setupSQLParityEnv(t)

		out, err := env.runMCP("sql_execute", map[string]any{
			"connection": "dev",
			"query":      "SELECT * FROM users",
			"format":     "csv",
		})
		require.NoError(t, err)
		require.Contains(t, out, "id,name")
		require.Contains(t, out, "1,Alice")
		require.Contains(t, out, "2,Bob")
	})

	t.Run("CLI and MCP produce equivalent CSV output", func(t *testing.T) {
		t.Parallel()
		env := setupSQLParityEnv(t)

		cliOut, cliErr := runSQLCLIWithSiphon(t, env, "sql", "dev", "SELECT * FROM users", "--format", "csv")
		require.NoError(t, cliErr)

		mcpOut, mcpErr := env.runMCP("sql_execute", map[string]any{
			"connection": "dev",
			"query":      "SELECT * FROM users",
			"format":     "csv",
		})
		require.NoError(t, mcpErr)

		// CSV output should be identical.
		require.Equal(t, strings.TrimSpace(cliOut), strings.TrimSpace(mcpOut))
	})
}

func TestParity_SQL_Write(t *testing.T) {
	t.Parallel()

	t.Run("CLI INSERT returns affected rows", func(t *testing.T) {
		t.Parallel()
		env := setupSQLParityEnv(t)

		out, err := runSQLCLIWithSiphon(t, env, "sql", "dev", "INSERT INTO users VALUES (3, 'Charlie')", "--format", "csv")
		require.NoError(t, err)
		require.Contains(t, out, "rows_affected")
		require.Contains(t, out, "1")
	})

	t.Run("MCP INSERT returns affected rows", func(t *testing.T) {
		t.Parallel()
		env := setupSQLParityEnv(t)

		out, err := env.runMCP("sql_execute", map[string]any{
			"connection": "dev",
			"query":      "INSERT INTO users VALUES (3, 'Charlie')",
			"format":     "csv",
			"confirm":    true,
		})
		require.NoError(t, err)
		require.Contains(t, out, "rows_affected")
		require.Contains(t, out, "1")
	})
}

func TestParity_SQL_ConnectionNotFound(t *testing.T) {
	t.Parallel()

	t.Run("CLI errors on missing connection", func(t *testing.T) {
		t.Parallel()
		env := setupSQLParityEnv(t)

		_, err := runSQLCLIWithSiphon(t, env, "sql", "nonexistent", "SELECT 1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("MCP errors on missing connection", func(t *testing.T) {
		t.Parallel()
		env := setupSQLParityEnv(t)

		_, err := env.runMCP("sql_execute", map[string]any{
			"connection": "nonexistent",
			"query":      "SELECT 1",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}

func TestParity_SQL_JSONFormat(t *testing.T) {
	t.Parallel()

	t.Run("CLI JSON format", func(t *testing.T) {
		t.Parallel()
		env := setupSQLParityEnv(t)

		out, err := runSQLCLIWithSiphon(t, env, "sql", "dev", "SELECT * FROM users", "--format", "json")
		require.NoError(t, err)
		require.Contains(t, out, `"id"`)
		require.Contains(t, out, `"Alice"`)
	})

	t.Run("MCP JSON format", func(t *testing.T) {
		t.Parallel()
		env := setupSQLParityEnv(t)

		out, err := env.runMCP("sql_execute", map[string]any{
			"connection": "dev",
			"query":      "SELECT * FROM users",
			"format":     "json",
		})
		require.NoError(t, err)
		require.Contains(t, out, `"id"`)
		require.Contains(t, out, `"Alice"`)
	})
}

// runSQLCLIWithSiphon executes a CLI command using the parity env's siphon.
func runSQLCLIWithSiphon(t *testing.T, env *parityEnv, args ...string) (string, error) {
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
