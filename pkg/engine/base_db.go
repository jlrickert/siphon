package engine

import (
	"context"
	"database/sql"
	"fmt"
)

// baseDBAdaptor provides shared Execute, Query, RawDB, and TransferTables
// implementations for all adaptors backed by *sql.DB. Embed this in concrete
// adaptor types to avoid duplicating these methods.
type baseDBAdaptor struct {
	db      *sql.DB
	dialect string // "sqlite", "mariadb", or "postgresql"
}

func (b *baseDBAdaptor) requireDB() error {
	if b.db == nil {
		return fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	return nil
}

func (b *baseDBAdaptor) RawDB() *sql.DB {
	return b.db
}

// --- QueryAdaptor ---

func (b *baseDBAdaptor) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if err := b.requireDB(); err != nil {
		return nil, err
	}
	return b.db.ExecContext(ctx, query, args...)
}

func (b *baseDBAdaptor) Query(ctx context.Context, query string, args ...any) (*QueryResult, error) {
	if err := b.requireDB(); err != nil {
		return nil, err
	}
	rows, err := b.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
	}
	defer rows.Close()
	return scanQueryResult(rows)
}

// --- TransferAdaptor (TransferTables only) ---

func (b *baseDBAdaptor) TransferTables(ctx context.Context, opts TransferOptions) error {
	if err := b.requireDB(); err != nil {
		return err
	}
	if opts.SourceDB == nil {
		return fmt.Errorf("%w: source database connection required", ErrTransferFailed)
	}
	return transferTablesSQL(ctx, opts.SourceDB, b.db, opts, b.dialect)
}
