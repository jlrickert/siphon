//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/stretchr/testify/require"
)

func TestBackupMariaDBLogical(t *testing.T) {
	adaptor := ConnectMariaDB(t)
	defer adaptor.Close()

	lb, ok := engine.HasLogicalBackup(adaptor)
	require.True(t, ok, "MariaDB adaptor must support logical backup")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dir := t.TempDir()
	err := lb.Dump(ctx, engine.DumpOptions{
		Database:   MariaDBDatabase,
		OutputPath: dir + "/dump.sql",
	})
	require.NoError(t, err, "logical dump should succeed")
}

func TestBackupPostgreSQLLogical(t *testing.T) {
	adaptor := ConnectPostgreSQL(t)
	defer adaptor.Close()

	lb, ok := engine.HasLogicalBackup(adaptor)
	require.True(t, ok, "PostgreSQL adaptor must support logical backup")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dir := t.TempDir()
	err := lb.Dump(ctx, engine.DumpOptions{
		Database:   PostgreSQLDatabase,
		OutputPath: dir + "/dump.sql",
	})
	require.NoError(t, err, "logical dump should succeed")
}

func TestBackupRestoreCycleMariaDB(t *testing.T) {
	adaptor := ConnectMariaDB(t)
	defer adaptor.Close()

	qa, ok := engine.HasQuery(adaptor)
	require.True(t, ok, "MariaDB adaptor must support query")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a test table and insert data.
	_, err := qa.Execute(ctx, "CREATE TABLE IF NOT EXISTS backup_cycle_test (id INT PRIMARY KEY, val VARCHAR(100))")
	require.NoError(t, err)

	_, err = qa.Execute(ctx, "INSERT IGNORE INTO backup_cycle_test VALUES (1, 'hello'), (2, 'world')")
	require.NoError(t, err)

	// Dump.
	lb, ok := engine.HasLogicalBackup(adaptor)
	require.True(t, ok)

	dir := t.TempDir()
	dumpPath := dir + "/cycle-dump.sql"
	err = lb.Dump(ctx, engine.DumpOptions{
		Database:   MariaDBDatabase,
		OutputPath: dumpPath,
	})
	require.NoError(t, err)

	// Drop the table.
	_, err = qa.Execute(ctx, "DROP TABLE backup_cycle_test")
	require.NoError(t, err)

	// Restore from dump.
	err = lb.LoadDump(ctx, engine.LoadDumpOptions{
		Database:  MariaDBDatabase,
		InputPath: dumpPath,
	})
	require.NoError(t, err)

	// Verify data survived the cycle.
	result, err := qa.Query(ctx, "SELECT id, val FROM backup_cycle_test ORDER BY id")
	require.NoError(t, err)
	require.Len(t, result.Rows, 2)
	require.Equal(t, "1", result.Rows[0][0])
	require.Equal(t, "hello", result.Rows[0][1])
	require.Equal(t, "2", result.Rows[1][0])
	require.Equal(t, "world", result.Rows[1][1])

	// Cleanup.
	_, _ = qa.Execute(ctx, "DROP TABLE IF EXISTS backup_cycle_test")
}

func TestBackupRestoreCyclePostgreSQL(t *testing.T) {
	adaptor := ConnectPostgreSQL(t)
	defer adaptor.Close()

	qa, ok := engine.HasQuery(adaptor)
	require.True(t, ok, "PostgreSQL adaptor must support query")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a test table and insert data.
	_, err := qa.Execute(ctx, "CREATE TABLE IF NOT EXISTS backup_cycle_test (id INT PRIMARY KEY, val VARCHAR(100))")
	require.NoError(t, err)

	_, err = qa.Execute(ctx, "INSERT INTO backup_cycle_test VALUES (1, 'hello'), (2, 'world') ON CONFLICT DO NOTHING")
	require.NoError(t, err)

	// Dump.
	lb, ok := engine.HasLogicalBackup(adaptor)
	require.True(t, ok)

	dir := t.TempDir()
	dumpPath := dir + "/cycle-dump.sql"
	err = lb.Dump(ctx, engine.DumpOptions{
		Database:   PostgreSQLDatabase,
		OutputPath: dumpPath,
	})
	require.NoError(t, err)

	// Drop the table.
	_, err = qa.Execute(ctx, "DROP TABLE backup_cycle_test")
	require.NoError(t, err)

	// Restore from dump.
	err = lb.LoadDump(ctx, engine.LoadDumpOptions{
		Database:  PostgreSQLDatabase,
		InputPath: dumpPath,
	})
	require.NoError(t, err)

	// Verify data survived the cycle.
	result, err := qa.Query(ctx, "SELECT id, val FROM backup_cycle_test ORDER BY id")
	require.NoError(t, err)
	require.Len(t, result.Rows, 2)
	require.Equal(t, "1", result.Rows[0][0])
	require.Equal(t, "hello", result.Rows[0][1])
	require.Equal(t, "2", result.Rows[1][0])
	require.Equal(t, "world", result.Rows[1][1])

	// Cleanup.
	_, _ = qa.Execute(ctx, "DROP TABLE IF EXISTS backup_cycle_test")
}
