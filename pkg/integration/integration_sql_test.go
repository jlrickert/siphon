//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/stretchr/testify/require"
)

func TestSQLExecuteMariaDB(t *testing.T) {
	adaptor := ConnectMariaDB(t)
	defer adaptor.Close()

	qa, ok := engine.HasQuery(adaptor)
	require.True(t, ok, "MariaDB adaptor must support query")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Simple SELECT.
	result, err := qa.Query(ctx, "SELECT 1 AS num, 'hello' AS greeting")
	require.NoError(t, err)
	require.Equal(t, []string{"num", "greeting"}, result.Columns)
	require.Len(t, result.Rows, 1)
	require.Equal(t, "1", result.Rows[0][0])
	require.Equal(t, "hello", result.Rows[0][1])
}

func TestSQLExecutePostgreSQL(t *testing.T) {
	adaptor := ConnectPostgreSQL(t)
	defer adaptor.Close()

	qa, ok := engine.HasQuery(adaptor)
	require.True(t, ok, "PostgreSQL adaptor must support query")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Simple SELECT.
	result, err := qa.Query(ctx, "SELECT 1 AS num, 'hello' AS greeting")
	require.NoError(t, err)
	require.Equal(t, []string{"num", "greeting"}, result.Columns)
	require.Len(t, result.Rows, 1)
	require.Equal(t, "1", result.Rows[0][0])
	require.Equal(t, "hello", result.Rows[0][1])
}

func TestSQLCreateAndQueryMariaDB(t *testing.T) {
	adaptor := ConnectMariaDB(t)
	defer adaptor.Close()

	qa, ok := engine.HasQuery(adaptor)
	require.True(t, ok)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// DDL + DML cycle.
	_, err := qa.Execute(ctx, "CREATE TABLE IF NOT EXISTS sql_test (id INT PRIMARY KEY, name VARCHAR(50))")
	require.NoError(t, err)

	_, err = qa.Execute(ctx, "INSERT IGNORE INTO sql_test VALUES (1, 'alice'), (2, 'bob'), (3, 'carol')")
	require.NoError(t, err)

	result, err := qa.Query(ctx, "SELECT id, name FROM sql_test WHERE id <= 2 ORDER BY id")
	require.NoError(t, err)
	require.Equal(t, []string{"id", "name"}, result.Columns)
	require.Len(t, result.Rows, 2)
	require.Equal(t, "alice", result.Rows[0][1])
	require.Equal(t, "bob", result.Rows[1][1])

	// Cleanup.
	_, _ = qa.Execute(ctx, "DROP TABLE IF EXISTS sql_test")
}

func TestSQLCreateAndQueryPostgreSQL(t *testing.T) {
	adaptor := ConnectPostgreSQL(t)
	defer adaptor.Close()

	qa, ok := engine.HasQuery(adaptor)
	require.True(t, ok)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// DDL + DML cycle.
	_, err := qa.Execute(ctx, "CREATE TABLE IF NOT EXISTS sql_test (id INT PRIMARY KEY, name VARCHAR(50))")
	require.NoError(t, err)

	_, err = qa.Execute(ctx, "INSERT INTO sql_test VALUES (1, 'alice'), (2, 'bob'), (3, 'carol') ON CONFLICT DO NOTHING")
	require.NoError(t, err)

	result, err := qa.Query(ctx, "SELECT id, name FROM sql_test WHERE id <= 2 ORDER BY id")
	require.NoError(t, err)
	require.Equal(t, []string{"id", "name"}, result.Columns)
	require.Len(t, result.Rows, 2)
	require.Equal(t, "alice", result.Rows[0][1])
	require.Equal(t, "bob", result.Rows[1][1])

	// Cleanup.
	_, _ = qa.Execute(ctx, "DROP TABLE IF EXISTS sql_test")
}

func TestSQLOutputFormatsMariaDB(t *testing.T) {
	adaptor := ConnectMariaDB(t)
	defer adaptor.Close()

	qa, ok := engine.HasQuery(adaptor)
	require.True(t, ok)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Multi-row query for formatter verification.
	result, err := qa.Query(ctx, "SELECT 1 AS a, 'x' AS b UNION ALL SELECT 2, 'y' UNION ALL SELECT 3, 'z'")
	require.NoError(t, err)
	require.Len(t, result.Columns, 2)
	require.Len(t, result.Rows, 3)
}

func TestSQLOutputFormatsPostgreSQL(t *testing.T) {
	adaptor := ConnectPostgreSQL(t)
	defer adaptor.Close()

	qa, ok := engine.HasQuery(adaptor)
	require.True(t, ok)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Multi-row query for formatter verification.
	result, err := qa.Query(ctx, "SELECT a, b FROM (VALUES (1, 'x'), (2, 'y'), (3, 'z')) AS t(a, b)")
	require.NoError(t, err)
	require.Len(t, result.Columns, 2)
	require.Len(t, result.Rows, 3)
}
