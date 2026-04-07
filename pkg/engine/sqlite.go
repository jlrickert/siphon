package engine

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Compile-time interface check.
var _ Adaptor = (*SQLiteAdaptor)(nil)

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
