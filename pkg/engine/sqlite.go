package engine

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	_ "modernc.org/sqlite"
)

// Compile-time interface checks.
var (
	_ Adaptor           = (*SQLiteAdaptor)(nil)
	_ FileBackupAdaptor = (*SQLiteAdaptor)(nil)
	_ QueryAdaptor      = (*SQLiteAdaptor)(nil)
	_ TransferAdaptor   = (*SQLiteAdaptor)(nil)
	_ RawDBAccessor     = (*SQLiteAdaptor)(nil)
)

// SQLiteAdaptor implements the Adaptor interface for SQLite.
// SQLite is an embedded engine -- the "connection" is a file path,
// and the database concept is implicit (the file IS the database).
type SQLiteAdaptor struct {
	cfg *ConnectionConfig
	db  *sql.DB
}

// NewSQLiteAdaptor creates a SQLite adaptor from configuration.
func NewSQLiteAdaptor(cfg *ConnectionConfig) *SQLiteAdaptor {
	return &SQLiteAdaptor{cfg: cfg}
}

func (a *SQLiteAdaptor) Engine() Engine {
	return EngineSQLite
}

func (a *SQLiteAdaptor) Connect(ctx context.Context) error {
	path := a.filePath()
	if path == "" {
		return fmt.Errorf("%w: no path configured for SQLite connection", ErrConnectionFailed)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	a.db = db
	return nil
}

func (a *SQLiteAdaptor) Ping(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	return a.db.PingContext(ctx)
}

func (a *SQLiteAdaptor) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

func (a *SQLiteAdaptor) ListDatabases(ctx context.Context) ([]string, error) {
	// SQLite has a single database per file. Return the filename as the
	// database name.
	path := a.filePath()
	if path == "" {
		return nil, nil
	}
	return []string{filepath.Base(path)}, nil
}

// --- FileBackupAdaptor ---

func (a *SQLiteAdaptor) CopyBackup(ctx context.Context, opts CopyBackupOptions) error {
	// Try VACUUM INTO first (creates a consistent copy without locking).
	if a.db != nil {
		_, err := a.db.ExecContext(ctx, "VACUUM INTO ?", opts.DestPath)
		if err == nil {
			return nil
		}
		// VACUUM INTO not available or failed; fall back to file copy.
	}
	return copyFile(opts.SourcePath, opts.DestPath)
}

func (a *SQLiteAdaptor) CopyRestore(ctx context.Context, opts CopyRestoreOptions) error {
	if !opts.Force {
		if _, err := os.Stat(opts.DestPath); err == nil {
			return fmt.Errorf("destination %s already exists (use --force to overwrite)", opts.DestPath)
		}
	}
	return copyFile(opts.SourcePath, opts.DestPath)
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}
	return out.Close()
}

func (a *SQLiteAdaptor) RawDB() *sql.DB {
	return a.db
}

// --- TransferAdaptor ---

func (a *SQLiteAdaptor) ListTables(ctx context.Context, database string) ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	rows, err := a.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func (a *SQLiteAdaptor) GetForeignKeys(ctx context.Context, database string) ([]ForeignKey, error) {
	if a.db == nil {
		return nil, fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	tables, err := a.ListTables(ctx, database)
	if err != nil {
		return nil, err
	}

	var fks []ForeignKey
	for _, table := range tables {
		rows, err := a.db.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list(%s)", table))
		if err != nil {
			continue
		}
		for rows.Next() {
			var id, seq int
			var refTable, from, to, onUpdate, onDelete, match string
			if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				rows.Close()
				continue
			}
			fks = append(fks, ForeignKey{
				Table:           table,
				ReferencedTable: refTable,
			})
		}
		rows.Close()
	}
	return fks, nil
}

func (a *SQLiteAdaptor) TransferTables(ctx context.Context, opts TransferOptions) error {
	if a.db == nil {
		return fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	if opts.SourceDB == nil {
		return fmt.Errorf("%w: source database connection required", ErrTransferFailed)
	}
	return transferTablesSQL(ctx, opts.SourceDB, a.db, opts, "sqlite")
}

// --- QueryAdaptor ---

func (a *SQLiteAdaptor) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if a.db == nil {
		return nil, fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	return a.db.ExecContext(ctx, query, args...)
}

func (a *SQLiteAdaptor) Query(ctx context.Context, query string, args ...any) (*QueryResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
	}
	defer rows.Close()
	return scanQueryResult(rows)
}

// filePath returns the configured SQLite file path.
func (a *SQLiteAdaptor) filePath() string {
	if a.cfg.Path != nil && *a.cfg.Path != "" {
		return *a.cfg.Path
	}
	if a.cfg.Database != nil && *a.cfg.Database != "" {
		return *a.cfg.Database
	}
	return ""
}
