package engine

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Compile-time interface checks.
var (
	_ Adaptor              = (*PostgreSQLAdaptor)(nil)
	_ LogicalBackupAdaptor = (*PostgreSQLAdaptor)(nil)
	_ QueryAdaptor         = (*PostgreSQLAdaptor)(nil)
	_ TransferAdaptor      = (*PostgreSQLAdaptor)(nil)
	_ RawDBAccessor        = (*PostgreSQLAdaptor)(nil)
)

// PostgreSQLAdaptor implements the Adaptor interface for PostgreSQL.
type PostgreSQLAdaptor struct {
	cfg *ConnectionConfig
	db  *sql.DB
}

// NewPostgreSQLAdaptor creates a PostgreSQL adaptor from configuration.
func NewPostgreSQLAdaptor(cfg *ConnectionConfig) *PostgreSQLAdaptor {
	return &PostgreSQLAdaptor{cfg: cfg}
}

func (a *PostgreSQLAdaptor) Engine() Engine {
	return EnginePostgreSQL
}

func (a *PostgreSQLAdaptor) Connect(ctx context.Context) error {
	dsn := a.buildDSN()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	a.db = db
	return nil
}

func (a *PostgreSQLAdaptor) Ping(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	return a.db.PingContext(ctx)
}

func (a *PostgreSQLAdaptor) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

func (a *PostgreSQLAdaptor) ListDatabases(ctx context.Context) ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	rows, err := a.db.QueryContext(ctx, "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
		}
		databases = append(databases, name)
	}
	return databases, rows.Err()
}

// --- LogicalBackupAdaptor ---

func (a *PostgreSQLAdaptor) Dump(ctx context.Context, opts DumpOptions) error {
	args := []string{"-Fc"} // custom format
	if a.cfg.Host != nil {
		args = append(args, "-h", *a.cfg.Host)
	}
	if a.cfg.Port != nil {
		args = append(args, "-p", strconv.Itoa(*a.cfg.Port))
	}
	if a.cfg.User != nil {
		args = append(args, "-U", *a.cfg.User)
	}
	db := opts.Database
	if db == "" && a.cfg.Database != nil {
		db = *a.cfg.Database
	}
	if db != "" {
		args = append(args, db)
	}
	args = append(args, opts.ExtraArgs...)

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	if a.cfg.Password != nil {
		cmd.Env = append(cmd.Environ(), "PGPASSWORD="+*a.cfg.Password)
	}
	if opts.Output != nil {
		cmd.Stdout = opts.Output
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%w: pg_dump: %v", ErrDumpFailed, err)
		}
		return nil
	}
	if opts.OutputPath != "" {
		f, err := os.Create(opts.OutputPath)
		if err != nil {
			return fmt.Errorf("%w: creating output file: %v", ErrDumpFailed, err)
		}
		defer f.Close()
		var stderr bytes.Buffer
		cmd.Stdout = f
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%w: pg_dump: %s: %v", ErrDumpFailed, stderr.String(), err)
		}
		return nil
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: pg_dump: %s: %v", ErrDumpFailed, string(output), err)
	}
	return nil
}

func (a *PostgreSQLAdaptor) LoadDump(ctx context.Context, opts LoadDumpOptions) error {
	args := []string{}
	if a.cfg.Host != nil {
		args = append(args, "-h", *a.cfg.Host)
	}
	if a.cfg.Port != nil {
		args = append(args, "-p", strconv.Itoa(*a.cfg.Port))
	}
	if a.cfg.User != nil {
		args = append(args, "-U", *a.cfg.User)
	}
	db := opts.Database
	if db == "" && a.cfg.Database != nil {
		db = *a.cfg.Database
	}
	if db != "" {
		args = append(args, "-d", db)
	}
	args = append(args, opts.ExtraArgs...)

	cmd := exec.CommandContext(ctx, "pg_restore", args...)
	if a.cfg.Password != nil {
		cmd.Env = append(cmd.Environ(), "PGPASSWORD="+*a.cfg.Password)
	}
	if opts.Input != nil {
		cmd.Stdin = opts.Input
	} else if opts.InputPath != "" {
		f, err := os.Open(opts.InputPath)
		if err != nil {
			return fmt.Errorf("%w: opening input file: %v", ErrLoadFailed, err)
		}
		defer f.Close()
		cmd.Stdin = f
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: pg_restore: %s: %v", ErrLoadFailed, string(output), err)
	}
	return nil
}

func (a *PostgreSQLAdaptor) RawDB() *sql.DB {
	return a.db
}

// --- TransferAdaptor ---

func (a *PostgreSQLAdaptor) ListTables(ctx context.Context, database string) ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	rows, err := a.db.QueryContext(ctx,
		"SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename")
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

func (a *PostgreSQLAdaptor) GetForeignKeys(ctx context.Context, database string) ([]ForeignKey, error) {
	if a.db == nil {
		return nil, fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	query := `SELECT tc.table_name, ccu.table_name AS referenced_table_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
			AND tc.table_schema = ccu.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_schema = 'public'`
	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		if err := rows.Scan(&fk.Table, &fk.ReferencedTable); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

func (a *PostgreSQLAdaptor) TransferTables(ctx context.Context, opts TransferOptions) error {
	if a.db == nil {
		return fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	if opts.SourceDB == nil {
		return fmt.Errorf("%w: source database connection required", ErrTransferFailed)
	}
	return transferTablesSQL(ctx, opts.SourceDB, a.db, opts, "postgresql")
}

// --- QueryAdaptor ---

func (a *PostgreSQLAdaptor) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if a.db == nil {
		return nil, fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	return a.db.ExecContext(ctx, query, args...)
}

func (a *PostgreSQLAdaptor) Query(ctx context.Context, query string, args ...any) (*QueryResult, error) {
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

// buildDSN constructs a PostgreSQL connection string from the connection configuration.
func (a *PostgreSQLAdaptor) buildDSN() string {
	host := "localhost"
	if a.cfg.Host != nil && *a.cfg.Host != "" {
		host = *a.cfg.Host
	}
	port := 5432
	if a.cfg.Port != nil && *a.cfg.Port != 0 {
		port = *a.cfg.Port
	}
	user := ""
	if a.cfg.User != nil {
		user = *a.cfg.User
	}
	password := ""
	if a.cfg.Password != nil {
		password = *a.cfg.Password
	}
	database := ""
	if a.cfg.Database != nil {
		database = *a.cfg.Database
	}

	// Use key=value format for pgx.
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database)
	return dsn
}
