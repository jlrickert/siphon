package engine

import (
	"context"
	"database/sql"
	"io"
)

// Adaptor is the core interface that every database engine must implement.
type Adaptor interface {
	Connect(ctx context.Context) error
	Ping(ctx context.Context) error
	Close() error
	ListDatabases(ctx context.Context) ([]string, error)
	Engine() Engine
}

// LogicalBackupAdaptor provides mysqldump/pg_dump style backup capabilities.
type LogicalBackupAdaptor interface {
	Dump(ctx context.Context, opts DumpOptions) error
	LoadDump(ctx context.Context, opts LoadDumpOptions) error
}

// PhysicalBackupAdaptor provides mariabackup/pg_basebackup style backup
// capabilities. Physical backups are server-scoped, not database-scoped.
type PhysicalBackupAdaptor interface {
	PhysicalBackup(ctx context.Context, opts PhysicalBackupOptions) error
	Prepare(ctx context.Context, opts PrepareOptions) error
	CopyBack(ctx context.Context, opts CopyBackOptions) error
}

// FileBackupAdaptor provides file-level copy backup (used by SQLite).
type FileBackupAdaptor interface {
	CopyBackup(ctx context.Context, opts CopyBackupOptions) error
	CopyRestore(ctx context.Context, opts CopyRestoreOptions) error
}

// TransferAdaptor provides cross-database table transfer.
type TransferAdaptor interface {
	ListTables(ctx context.Context, database string) ([]string, error)
	GetForeignKeys(ctx context.Context, database string) ([]ForeignKey, error)
	TransferTables(ctx context.Context, opts TransferOptions) error
}

// QueryAdaptor provides raw SQL execution capabilities.
type QueryAdaptor interface {
	Execute(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(ctx context.Context, query string, args ...any) (*QueryResult, error)
}

// DumpOptions configures a logical dump operation.
type DumpOptions struct {
	Database   string
	Tables     []string
	Output     io.Writer
	OutputPath string
	ExtraArgs  []string
}

// LoadDumpOptions configures loading a logical dump.
type LoadDumpOptions struct {
	Database  string
	Input     io.Reader
	InputPath string
	ExtraArgs []string
}

// PhysicalBackupOptions configures a physical backup operation.
type PhysicalBackupOptions struct {
	TargetDir string
	ExtraArgs []string
}

// PrepareOptions configures a physical backup prepare step.
type PrepareOptions struct {
	TargetDir string
	ExtraArgs []string
}

// CopyBackOptions configures restoring a physical backup.
type CopyBackOptions struct {
	SourceDir string
	DataDir   string
	ExtraArgs []string
}

// CopyBackupOptions configures a file-level copy backup (SQLite).
type CopyBackupOptions struct {
	SourcePath string
	DestPath   string
}

// CopyRestoreOptions configures a file-level copy restore (SQLite).
type CopyRestoreOptions struct {
	SourcePath string
	DestPath   string
	Force      bool
}

// TransferOptions configures a cross-database table transfer.
type TransferOptions struct {
	Source      DatabaseTarget
	Destination DatabaseTarget
	Tables      []string
	OnConflict  string // "skip", "overwrite", "merge"
}

// QueryResult holds the result of a query execution.
type QueryResult struct {
	Columns []string
	Rows    [][]string
}
