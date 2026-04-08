package engine

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

// Compile-time interface checks.
var (
	_ Adaptor               = (*MariaDBAdaptor)(nil)
	_ PhysicalBackupAdaptor = (*MariaDBAdaptor)(nil)
	_ LogicalBackupAdaptor  = (*MariaDBAdaptor)(nil)
	_ QueryAdaptor          = (*MariaDBAdaptor)(nil)
	_ TransferAdaptor       = (*MariaDBAdaptor)(nil)
	_ RawDBAccessor         = (*MariaDBAdaptor)(nil)
)

// MariaDBAdaptor implements the Adaptor interface for MariaDB/MySQL.
type MariaDBAdaptor struct {
	baseDBAdaptor
	cfg *ConnectionConfig
}

// NewMariaDBAdaptor creates a MariaDB adaptor from configuration.
func NewMariaDBAdaptor(cfg *ConnectionConfig) *MariaDBAdaptor {
	return &MariaDBAdaptor{
		baseDBAdaptor: baseDBAdaptor{dialect: "mariadb"},
		cfg:           cfg,
	}
}

func (a *MariaDBAdaptor) Engine() Engine {
	return EngineMariaDB
}

func (a *MariaDBAdaptor) Connect(ctx context.Context) error {
	dsn := a.buildDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	a.db = db
	return nil
}

func (a *MariaDBAdaptor) Ping(ctx context.Context) error {
	if err := a.requireDB(); err != nil {
		return err
	}
	return a.db.PingContext(ctx)
}

func (a *MariaDBAdaptor) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

func (a *MariaDBAdaptor) ListDatabases(ctx context.Context) ([]string, error) {
	if err := a.requireDB(); err != nil {
		return nil, err
	}
	rows, err := a.db.QueryContext(ctx, "SHOW DATABASES")
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

// --- PhysicalBackupAdaptor ---

func (a *MariaDBAdaptor) PhysicalBackup(ctx context.Context, opts PhysicalBackupOptions) error {
	args := []string{"--backup", "--target-dir=" + opts.TargetDir}
	if a.cfg.User != nil {
		args = append(args, "--user="+*a.cfg.User)
	}
	if a.cfg.Password != nil {
		args = append(args, "--password="+*a.cfg.Password)
	}
	if a.cfg.Host != nil {
		args = append(args, "--host="+*a.cfg.Host)
	}
	if a.cfg.Port != nil {
		args = append(args, "--port="+strconv.Itoa(*a.cfg.Port))
	}
	args = append(args, opts.ExtraArgs...)

	cmd := exec.CommandContext(ctx, "mariabackup", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: mariabackup --backup: %s: %v", ErrPhysicalBackupFailed, string(output), err)
	}
	return nil
}

func (a *MariaDBAdaptor) Prepare(ctx context.Context, opts PrepareOptions) error {
	args := []string{"--prepare", "--target-dir=" + opts.TargetDir}
	args = append(args, opts.ExtraArgs...)

	cmd := exec.CommandContext(ctx, "mariabackup", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: mariabackup --prepare: %s: %v", ErrPrepareFailed, string(output), err)
	}
	return nil
}

func (a *MariaDBAdaptor) CopyBack(ctx context.Context, opts CopyBackOptions) error {
	args := []string{"--copy-back", "--target-dir=" + opts.SourceDir}
	if opts.DataDir != "" {
		args = append(args, "--datadir="+opts.DataDir)
	}
	args = append(args, opts.ExtraArgs...)

	cmd := exec.CommandContext(ctx, "mariabackup", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: mariabackup --copy-back: %s: %v", ErrCopyBackFailed, string(output), err)
	}
	return nil
}

// --- LogicalBackupAdaptor ---
// Note: Uses os.Create/os.Open directly because engine adaptors do not
// receive toolkit.Runtime. This is a documented exception — engine-level
// file I/O and external tool invocations bypass the Runtime abstraction.

func (a *MariaDBAdaptor) Dump(ctx context.Context, opts DumpOptions) error {
	args := []string{"--single-transaction", "--routines", "--triggers"}
	if a.cfg.User != nil {
		args = append(args, "-u", *a.cfg.User)
	}
	if a.cfg.Password != nil {
		args = append(args, "-p"+*a.cfg.Password)
	}
	if a.cfg.Host != nil {
		args = append(args, "-h", *a.cfg.Host)
	}
	if a.cfg.Port != nil {
		args = append(args, "-P", strconv.Itoa(*a.cfg.Port))
	}
	if opts.Database != "" {
		args = append(args, "--databases", opts.Database)
	}
	args = append(args, opts.ExtraArgs...)

	cmd := exec.CommandContext(ctx, "mariadb-dump", args...)
	if opts.Output != nil {
		cmd.Stdout = opts.Output
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%w: mariadb-dump: %v", ErrDumpFailed, err)
		}
		return nil
	}
	if opts.OutputPath != "" {
		f, err := os.Create(opts.OutputPath)
		if err != nil {
			return fmt.Errorf("%w: creating output file: %v", ErrDumpFailed, err)
		}
		defer f.Close()
		cmd.Stdout = f
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%w: mariadb-dump: %v", ErrDumpFailed, err)
		}
		return nil
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: mariadb-dump: %s: %v", ErrDumpFailed, string(output), err)
	}
	return nil
}

func (a *MariaDBAdaptor) LoadDump(ctx context.Context, opts LoadDumpOptions) error {
	args := []string{}
	if a.cfg.User != nil {
		args = append(args, "-u", *a.cfg.User)
	}
	if a.cfg.Password != nil {
		args = append(args, "-p"+*a.cfg.Password)
	}
	if a.cfg.Host != nil {
		args = append(args, "-h", *a.cfg.Host)
	}
	if a.cfg.Port != nil {
		args = append(args, "-P", strconv.Itoa(*a.cfg.Port))
	}
	if opts.Database != "" {
		args = append(args, opts.Database)
	}
	args = append(args, opts.ExtraArgs...)

	cmd := exec.CommandContext(ctx, "mariadb", args...)
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
		return fmt.Errorf("%w: mariadb load: %s: %v", ErrLoadFailed, string(output), err)
	}
	return nil
}

// --- TransferAdaptor (ListTables, GetForeignKeys) ---
// TransferTables is provided by the embedded baseDBAdaptor.

func (a *MariaDBAdaptor) ListTables(ctx context.Context, database string) ([]string, error) {
	if err := a.requireDB(); err != nil {
		return nil, err
	}
	query := "SHOW TABLES"
	if database != "" {
		query = fmt.Sprintf("SHOW TABLES FROM %s", database)
	}
	rows, err := a.db.QueryContext(ctx, query)
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

func (a *MariaDBAdaptor) GetForeignKeys(ctx context.Context, database string) ([]ForeignKey, error) {
	if err := a.requireDB(); err != nil {
		return nil, err
	}
	query := `SELECT TABLE_NAME, REFERENCED_TABLE_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE REFERENCED_TABLE_NAME IS NOT NULL
		AND TABLE_SCHEMA = ?`
	db := database
	if db == "" && a.cfg.Database != nil {
		db = *a.cfg.Database
	}
	rows, err := a.db.QueryContext(ctx, query, db)
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

// buildDSN constructs a MySQL DSN from the connection configuration.
func (a *MariaDBAdaptor) buildDSN() string {
	user := ""
	if a.cfg.User != nil {
		user = *a.cfg.User
	}
	password := ""
	if a.cfg.Password != nil {
		password = *a.cfg.Password
	}
	host := "localhost"
	if a.cfg.Host != nil && *a.cfg.Host != "" {
		host = *a.cfg.Host
	}
	port := 3306
	if a.cfg.Port != nil && *a.cfg.Port != 0 {
		port = *a.cfg.Port
	}
	database := ""
	if a.cfg.Database != nil {
		database = *a.cfg.Database
	}

	// Format: user:password@tcp(host:port)/database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, database)
	return dsn
}
