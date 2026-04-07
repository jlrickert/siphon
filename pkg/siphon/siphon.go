package siphon

import (
	"context"
	"fmt"
	"time"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/siphon/pkg/engine"
)

// Siphon is the central service struct. All operations flow through Siphon
// via two parallel entry points: CLI commands and MCP tool calls.
type Siphon struct {
	Runtime       *toolkit.Runtime
	PathService   *PathService
	ConfigService *ConfigService
}

// SiphonOptions configures a new Siphon instance.
type SiphonOptions struct {
	Root       string
	ConfigPath string
	Runtime    *toolkit.Runtime
}

// New creates a new Siphon service instance.
func New(opts SiphonOptions) (*Siphon, error) {
	rt := opts.Runtime
	if rt == nil {
		var err error
		rt, err = toolkit.NewRuntime()
		if err != nil {
			return nil, fmt.Errorf("unable to create runtime: %w", err)
		}
	}
	if err := rt.Validate(); err != nil {
		return nil, fmt.Errorf("invalid runtime: %w", err)
	}

	root := opts.Root
	if root == "" {
		wd, err := rt.Getwd()
		if err != nil {
			return nil, fmt.Errorf("unable to determine working directory: %w", err)
		}
		root = wd
	}

	pathService, err := NewPathService(rt, root)
	if err != nil {
		return nil, fmt.Errorf("unable to create path service: %w", err)
	}
	configService := &ConfigService{
		Runtime:     rt,
		PathService: pathService,
		ConfigPath:  opts.ConfigPath,
	}

	return &Siphon{
		Runtime:       rt,
		PathService:   pathService,
		ConfigService: configService,
	}, nil
}

// --- Connection management ---

// ListConnectionsOptions configures listing connections.
type ListConnectionsOptions struct {
	Format string // output format
}

// ConnectionInfo describes a configured connection.
type ConnectionInfo struct {
	Name   string        `json:"name"`
	Engine engine.Engine `json:"engine"`
	Host   string        `json:"host,omitempty"`
	Port   int           `json:"port,omitempty"`
}

// ListConnections returns all configured connections.
func (s *Siphon) ListConnections(ctx context.Context, opts *ListConnectionsOptions) ([]ConnectionInfo, error) {
	return nil, ErrNotImplemented
}

// AddConnectionOptions configures adding a connection.
type AddConnectionOptions struct {
	Name        string
	Engine      engine.Engine
	Host        string
	Port        int
	User        string
	Password    string
	PasswordEnv string
	Database    string
	Path        string // SQLite file path
}

// AddConnection adds a new connection to the configuration.
func (s *Siphon) AddConnection(ctx context.Context, opts *AddConnectionOptions) error {
	return ErrNotImplemented
}

// RemoveConnectionOptions configures removing a connection.
type RemoveConnectionOptions struct {
	Name  string
	Force bool
}

// RemoveConnection removes a connection from the configuration.
func (s *Siphon) RemoveConnection(ctx context.Context, opts *RemoveConnectionOptions) error {
	return ErrNotImplemented
}

// TestConnectionOptions configures testing a connection.
type TestConnectionOptions struct {
	Name string
}

// TestConnection tests connectivity to a named connection.
func (s *Siphon) TestConnection(ctx context.Context, opts *TestConnectionOptions) error {
	return ErrNotImplemented
}

// --- Backup operations ---

// BackupOptions configures a backup operation.
type BackupOptions struct {
	Connection string
	Database   string
	BackupType string // "logical", "physical", "file"
	Repo       string
	Label      string
	Tables     []string
	Force      bool
}

// Backup creates a database backup.
func (s *Siphon) Backup(ctx context.Context, opts *BackupOptions) error {
	return ErrNotImplemented
}

// BackupInfoOptions configures retrieving backup metadata.
type BackupInfoOptions struct {
	BackupID string
	Path     string
}

// BackupDescriptor holds metadata for a completed backup.
type BackupDescriptor struct {
	ID         string        `json:"id"`
	Engine     engine.Engine `json:"engine"`
	BackupType string        `json:"backup_type"` // "logical", "physical", "file"
	Connection string        `json:"connection"`
	Database   string        `json:"database"`
	Timestamp  time.Time     `json:"timestamp"`
	Size       int64         `json:"size"`
	Checksum   string        `json:"checksum,omitempty"`
	Label      string        `json:"label,omitempty"`
	FilePaths  []string      `json:"file_paths,omitempty"`
}

// BackupInfo retrieves metadata for a specific backup.
func (s *Siphon) BackupInfo(ctx context.Context, opts *BackupInfoOptions) (*BackupDescriptor, error) {
	return nil, ErrNotImplemented
}

// ListBackupsOptions configures listing backups.
type ListBackupsOptions struct {
	Connection string
	Database   string
	Repo       string
	Limit      int
}

// ListBackups returns available backups matching the filter criteria.
func (s *Siphon) ListBackups(ctx context.Context, opts *ListBackupsOptions) ([]BackupDescriptor, error) {
	return nil, ErrNotImplemented
}

// VerifyBackupOptions configures backup verification.
type VerifyBackupOptions struct {
	BackupID string
	Path     string
}

// VerifyBackup verifies the integrity of a backup.
func (s *Siphon) VerifyBackup(ctx context.Context, opts *VerifyBackupOptions) error {
	return ErrNotImplemented
}

// LabelBackupOptions configures labeling a backup.
type LabelBackupOptions struct {
	BackupID string
	Label    string
}

// LabelBackup assigns or updates a label on a backup.
func (s *Siphon) LabelBackup(ctx context.Context, opts *LabelBackupOptions) error {
	return ErrNotImplemented
}

// --- Restore operations ---

// RestoreOptions configures a restore operation.
type RestoreOptions struct {
	BackupID   string
	Path       string
	Connection string
	Database   string
	Force      bool
	Confirm    bool
}

// Restore restores a database from a backup.
func (s *Siphon) Restore(ctx context.Context, opts *RestoreOptions) error {
	return ErrNotImplemented
}

// --- Transfer operations ---

// TransferOptions configures a cross-database transfer.
type TransferOptions struct {
	Source      string // CONN:DATABASE format
	Destination string // CONN:DATABASE format
	Tables      []string
	Truncate    bool
	Force       bool
	Confirm     bool
}

// Transfer transfers tables between databases.
func (s *Siphon) Transfer(ctx context.Context, opts *TransferOptions) error {
	return ErrNotImplemented
}

// --- SQL execution ---

// ExecuteSQLOptions configures raw SQL execution.
type ExecuteSQLOptions struct {
	Connection string
	Database   string
	Query      string
	Format     string // "table", "json", "csv"
}

// QueryResult holds the result of a SQL query.
type QueryResult struct {
	Columns      []string   `json:"columns"`
	Rows         [][]string `json:"rows"`
	RowsAffected int64      `json:"rows_affected"`
}

// ExecuteSQL executes a SQL query against a connection.
func (s *Siphon) ExecuteSQL(ctx context.Context, opts *ExecuteSQLOptions) (*QueryResult, error) {
	return nil, ErrNotImplemented
}

// --- Schedule operations ---

// CreateScheduleOptions configures creating a backup schedule.
type CreateScheduleOptions struct {
	Name       string
	Connection string
	Database   string
	Cron       string
	Repo       string
	BackupType string
	Retain     int
}

// ScheduleInfo describes a configured backup schedule.
type ScheduleInfo struct {
	Name       string `json:"name"`
	Connection string `json:"connection"`
	Database   string `json:"database"`
	Cron       string `json:"cron"`
	Repo       string `json:"repo"`
	BackupType string `json:"backup_type"`
	Retain     int    `json:"retain"`
	Enabled    bool   `json:"enabled"`
}

// CreateSchedule creates a new backup schedule.
func (s *Siphon) CreateSchedule(ctx context.Context, opts *CreateScheduleOptions) error {
	return ErrNotImplemented
}

// ListSchedulesOptions configures listing schedules.
type ListSchedulesOptions struct {
	Connection string
}

// ListSchedules returns all configured backup schedules.
func (s *Siphon) ListSchedules(ctx context.Context, opts *ListSchedulesOptions) ([]ScheduleInfo, error) {
	return nil, ErrNotImplemented
}

// RemoveScheduleOptions configures removing a schedule.
type RemoveScheduleOptions struct {
	Name string
}

// RemoveSchedule removes a backup schedule.
func (s *Siphon) RemoveSchedule(ctx context.Context, opts *RemoveScheduleOptions) error {
	return ErrNotImplemented
}

// --- Config operations ---

// ConfigOptions configures reading resolved configuration.
type ConfigOptions struct {
	Explain bool
}

// ResolvedConfig holds the fully resolved configuration with optional
// per-field provenance.
type ResolvedConfig struct {
	Config    *Config            `json:"config"`
	Provenance map[string]string `json:"provenance,omitempty"` // field -> source tier
}

// Config returns the resolved configuration.
func (s *Siphon) Config(ctx context.Context, opts *ConfigOptions) (*ResolvedConfig, error) {
	return nil, ErrNotImplemented
}
