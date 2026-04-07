package siphon

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/stretchr/testify/require"
)

// newSQLTestSiphon creates a sandboxed Siphon with a mock factory that
// supports query operations.
func newSQLTestSiphon(t *testing.T) (*Siphon, *sandbox.Sandbox) {
	t.Helper()

	sb := sandbox.NewSandbox(t, nil)
	rt := sb.Runtime()

	configDir := "/home/testuser/.config/siphon"
	sb.Mkdir(configDir, true)
	sb.WriteFile(configDir+"/config.yaml", []byte(`
version: "1"
default_connection: mydb
connections:
  mydb:
    name: mydb
    engine: mariadb
    host: localhost
    database: testdb
`), 0o644)

	s, err := New(SiphonOptions{Runtime: rt})
	require.NoError(t, err)

	s.AdaptorFactory = func(cfg *engine.ConnectionConfig) (engine.Adaptor, error) {
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
			return &testSQLResult{affected: 3}, nil
		}
		return mock, nil
	}

	return s, sb
}

type testSQLResult struct {
	affected int64
}

func (r *testSQLResult) LastInsertId() (int64, error) { return 0, nil }
func (r *testSQLResult) RowsAffected() (int64, error) { return r.affected, nil }

var _ driver.Result = (*testSQLResult)(nil)

func TestExecuteSQL_BasicSelect(t *testing.T) {
	t.Parallel()
	s, _ := newSQLTestSiphon(t)

	result, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "mydb",
		Query:      "SELECT * FROM users",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"id", "name"}, result.Columns)
	require.Len(t, result.Rows, 2)
	require.Equal(t, "Alice", result.Rows[0][1])
}

func TestExecuteSQL_WriteQuery(t *testing.T) {
	t.Parallel()
	s, _ := newSQLTestSiphon(t)

	result, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "mydb",
		Query:      "INSERT INTO users (name) VALUES ('Charlie')",
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), result.RowsAffected)
	require.Equal(t, []string{"rows_affected"}, result.Columns)
	require.Equal(t, "3", result.Rows[0][0])
}

func TestExecuteSQL_ReadonlyPolicyBlocksWrites(t *testing.T) {
	t.Parallel()
	s, sb := newSQLTestSiphon(t)

	configDir := "/home/testuser/.config/siphon"
	sb.WriteFile(configDir+"/config.yaml", []byte(`
version: "1"
connections:
  mydb:
    name: mydb
    engine: mariadb
    host: localhost
    database: testdb
policies:
  mydb:
    cli: readonly
`), 0o644)
	s.ConfigService.ResetCache()

	// Read should succeed.
	result, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "mydb",
		Query:      "SELECT * FROM users",
		Surface:    SurfaceCLI,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Write should be denied.
	_, err = s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "mydb",
		Query:      "DELETE FROM users",
		Surface:    SurfaceCLI,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrOperationDenied)
}

func TestExecuteSQL_ConfirmPolicyRequiresConfirmation(t *testing.T) {
	t.Parallel()
	s, sb := newSQLTestSiphon(t)

	configDir := "/home/testuser/.config/siphon"
	sb.WriteFile(configDir+"/config.yaml", []byte(`
version: "1"
connections:
  mydb:
    name: mydb
    engine: mariadb
    host: localhost
    database: testdb
policies:
  mydb:
    mcp: confirm
`), 0o644)
	s.ConfigService.ResetCache()

	// Write without confirm should require confirmation.
	_, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "mydb",
		Query:      "INSERT INTO users VALUES (1)",
		Surface:    SurfaceMCP,
		Confirm:    false,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrConfirmationRequired)

	// With confirm should succeed.
	result, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "mydb",
		Query:      "INSERT INTO users VALUES (1)",
		Surface:    SurfaceMCP,
		Confirm:    true,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Read without confirm should succeed (confirm only affects writes).
	result, err = s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "mydb",
		Query:      "SELECT 1",
		Surface:    SurfaceMCP,
		Confirm:    false,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestExecuteSQL_FileInput(t *testing.T) {
	t.Parallel()
	s, sb := newSQLTestSiphon(t)

	// Write a SQL file.
	sb.WriteFile("/home/testuser/query.sql", []byte("SELECT * FROM users"), 0o644)

	result, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "mydb",
		File:       "/home/testuser/query.sql",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"id", "name"}, result.Columns)
}

func TestExecuteSQL_FormatSelection(t *testing.T) {
	t.Parallel()
	s, _ := newSQLTestSiphon(t)

	// Just verify it doesn't error for each format --
	// the formatter tests cover output correctness.
	for _, format := range []string{"table", "csv", "json"} {
		result, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
			Connection: "mydb",
			Query:      "SELECT * FROM users",
			Format:     format,
		})
		require.NoError(t, err, "format %s", format)
		require.NotNil(t, result)
	}
}

func TestExecuteSQL_DefaultConnection(t *testing.T) {
	t.Parallel()
	s, _ := newSQLTestSiphon(t)

	// Connection empty -- should use default.
	result, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Query: "SELECT 1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestExecuteSQL_ConnectionNotFound(t *testing.T) {
	t.Parallel()
	s, _ := newSQLTestSiphon(t)

	_, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "nonexistent",
		Query:      "SELECT 1",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrConnectionNotFound)
}

func TestExecuteSQL_EmptyQuery(t *testing.T) {
	t.Parallel()
	s, _ := newSQLTestSiphon(t)

	_, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "mydb",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "query is required")
}

func TestExecuteSQL_DenyPolicy(t *testing.T) {
	t.Parallel()
	s, sb := newSQLTestSiphon(t)

	configDir := "/home/testuser/.config/siphon"
	sb.WriteFile(configDir+"/config.yaml", []byte(`
version: "1"
connections:
  mydb:
    name: mydb
    engine: mariadb
    host: localhost
    database: testdb
policies:
  mydb:
    cli: deny
`), 0o644)
	s.ConfigService.ResetCache()

	_, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "mydb",
		Query:      "SELECT 1",
		Surface:    SurfaceCLI,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrOperationDenied)
}

func TestExecuteSQL_AllowSQLFalse(t *testing.T) {
	t.Parallel()
	s, sb := newSQLTestSiphon(t)

	configDir := "/home/testuser/.config/siphon"
	sb.WriteFile(configDir+"/config.yaml", []byte(`
version: "1"
connections:
  mydb:
    name: mydb
    engine: mariadb
    host: localhost
    database: testdb
policies:
  mydb:
    allow_sql: false
`), 0o644)
	s.ConfigService.ResetCache()

	_, err := s.ExecuteSQL(context.Background(), &ExecuteSQLOptions{
		Connection: "mydb",
		Query:      "SELECT 1",
		Surface:    SurfaceCLI,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrOperationDenied)
}
