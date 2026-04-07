package engine

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Compile-time interface checks.
var (
	_ Adaptor              = (*PostgreSQLAdaptor)(nil)
	_ LogicalBackupAdaptor = (*PostgreSQLAdaptor)(nil)
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
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: pg_restore: %s: %v", ErrLoadFailed, string(output), err)
	}
	return nil
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
