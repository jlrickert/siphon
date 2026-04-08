package engine_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/stretchr/testify/require"
)

func newSQLiteAdaptor(t *testing.T) *engine.SQLiteAdaptor {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	a := engine.NewSQLiteAdaptor(&engine.ConnectionConfig{
		Path: &path,
	})
	ctx := context.Background()
	require.NoError(t, a.Connect(ctx))
	t.Cleanup(func() { a.Close() })
	return a
}

func TestSQLiteQueryAdaptor_Execute(t *testing.T) {
	a := newSQLiteAdaptor(t)
	ctx := context.Background()

	// Create a table.
	result, err := a.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Insert rows.
	result, err = a.Execute(ctx, "INSERT INTO users (name, age) VALUES (?, ?)", "alice", 30)
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	result, err = a.Execute(ctx, "INSERT INTO users (name, age) VALUES (?, ?)", "bob", 25)
	require.NoError(t, err)
	affected, err = result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
}

func TestSQLiteQueryAdaptor_Query(t *testing.T) {
	a := newSQLiteAdaptor(t)
	ctx := context.Background()

	// Setup.
	_, err := a.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")
	require.NoError(t, err)
	_, err = a.Execute(ctx, "INSERT INTO users (name, age) VALUES ('alice', 30), ('bob', 25)")
	require.NoError(t, err)

	// Query.
	qr, err := a.Query(ctx, "SELECT name, age FROM users ORDER BY name")
	require.NoError(t, err)
	require.Equal(t, []string{"name", "age"}, qr.Columns)
	require.Len(t, qr.Rows, 2)
	require.Equal(t, []string{"alice", "30"}, qr.Rows[0])
	require.Equal(t, []string{"bob", "25"}, qr.Rows[1])
}

func TestSQLiteQueryAdaptor_QueryNullValues(t *testing.T) {
	a := newSQLiteAdaptor(t)
	ctx := context.Background()

	_, err := a.Execute(ctx, "CREATE TABLE items (id INTEGER PRIMARY KEY, label TEXT)")
	require.NoError(t, err)
	_, err = a.Execute(ctx, "INSERT INTO items (label) VALUES (NULL)")
	require.NoError(t, err)

	qr, err := a.Query(ctx, "SELECT label FROM items")
	require.NoError(t, err)
	require.Len(t, qr.Rows, 1)
	require.Equal(t, "NULL", qr.Rows[0][0])
}

func TestSQLiteQueryAdaptor_NotConnected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	a := engine.NewSQLiteAdaptor(&engine.ConnectionConfig{
		Path: &path,
	})
	ctx := context.Background()

	_, err := a.Execute(ctx, "SELECT 1")
	require.Error(t, err)

	_, err = a.Query(ctx, "SELECT 1")
	require.Error(t, err)
}

func TestSQLiteTransferAdaptor_ListTables(t *testing.T) {
	a := newSQLiteAdaptor(t)
	ctx := context.Background()

	_, err := a.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)
	_, err = a.Execute(ctx, "CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER)")
	require.NoError(t, err)

	tables, err := a.ListTables(ctx, "")
	require.NoError(t, err)
	require.Equal(t, []string{"orders", "users"}, tables)
}

func TestSQLiteTransferAdaptor_GetForeignKeys(t *testing.T) {
	a := newSQLiteAdaptor(t)
	ctx := context.Background()

	_, err := a.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)
	_, err = a.Execute(ctx, "CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER REFERENCES users(id))")
	require.NoError(t, err)

	fks, err := a.GetForeignKeys(ctx, "")
	require.NoError(t, err)
	require.Len(t, fks, 1)
	require.Equal(t, "orders", fks[0].Table)
	require.Equal(t, "users", fks[0].ReferencedTable)
}

func TestSQLiteTransferAdaptor_TransferTables(t *testing.T) {
	// Create source database with data.
	src := newSQLiteAdaptor(t)
	ctx := context.Background()

	_, err := src.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")
	require.NoError(t, err)
	_, err = src.Execute(ctx, "INSERT INTO users (name, age) VALUES ('alice', 30), ('bob', 25)")
	require.NoError(t, err)

	// Create destination database with same schema.
	dst := newSQLiteAdaptor(t)
	_, err = dst.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")
	require.NoError(t, err)

	// Transfer.
	err = dst.TransferTables(ctx, engine.TransferOptions{
		Tables:   []string{"users"},
		SourceDB: src.RawDB(),
	})
	require.NoError(t, err)

	// Verify data arrived.
	qr, err := dst.Query(ctx, "SELECT name, age FROM users ORDER BY name")
	require.NoError(t, err)
	require.Len(t, qr.Rows, 2)
	require.Equal(t, []string{"alice", "30"}, qr.Rows[0])
	require.Equal(t, []string{"bob", "25"}, qr.Rows[1])
}

func TestSQLiteTransferAdaptor_TransferSkipConflict(t *testing.T) {
	src := newSQLiteAdaptor(t)
	dst := newSQLiteAdaptor(t)
	ctx := context.Background()

	// Source has alice and bob.
	_, err := src.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)
	_, err = src.Execute(ctx, "INSERT INTO users (id, name) VALUES (1, 'alice'), (2, 'bob')")
	require.NoError(t, err)

	// Destination already has alice with different data.
	_, err = dst.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)
	_, err = dst.Execute(ctx, "INSERT INTO users (id, name) VALUES (1, 'alice-original')")
	require.NoError(t, err)

	// Transfer with skip — alice should NOT be overwritten.
	err = dst.TransferTables(ctx, engine.TransferOptions{
		Tables:     []string{"users"},
		OnConflict: "skip",
		SourceDB:   src.RawDB(),
	})
	require.NoError(t, err)

	qr, err := dst.Query(ctx, "SELECT name FROM users ORDER BY id")
	require.NoError(t, err)
	require.Len(t, qr.Rows, 2)
	require.Equal(t, "alice-original", qr.Rows[0][0]) // kept original
	require.Equal(t, "bob", qr.Rows[1][0])             // inserted new
}

func TestSQLiteTransferAdaptor_TransferOverwriteConflict(t *testing.T) {
	src := newSQLiteAdaptor(t)
	dst := newSQLiteAdaptor(t)
	ctx := context.Background()

	_, err := src.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)
	_, err = src.Execute(ctx, "INSERT INTO users (id, name) VALUES (1, 'alice-new')")
	require.NoError(t, err)

	_, err = dst.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)
	_, err = dst.Execute(ctx, "INSERT INTO users (id, name) VALUES (1, 'alice-old')")
	require.NoError(t, err)

	// Transfer with overwrite — alice should be replaced.
	err = dst.TransferTables(ctx, engine.TransferOptions{
		Tables:     []string{"users"},
		OnConflict: "overwrite",
		SourceDB:   src.RawDB(),
	})
	require.NoError(t, err)

	qr, err := dst.Query(ctx, "SELECT name FROM users WHERE id = 1")
	require.NoError(t, err)
	require.Len(t, qr.Rows, 1)
	require.Equal(t, "alice-new", qr.Rows[0][0])
}
