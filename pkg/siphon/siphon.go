package siphon

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jlrickert/cli-toolkit/toolkit"
	"github.com/jlrickert/siphon/pkg/engine"
)

// AdaptorFactory creates an engine adaptor from connection config. The default
// factory is engine.NewAdaptor. Tests can override this to inject mock adaptors.
type AdaptorFactory func(cfg *engine.ConnectionConfig) (engine.Adaptor, error)

// Siphon is the central service struct. All operations flow through Siphon
// via two parallel entry points: CLI commands and MCP tool calls.
type Siphon struct {
	Runtime        *toolkit.Runtime
	PathService    *PathService
	ConfigService  *ConfigService
	AdaptorFactory AdaptorFactory
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
		Runtime:        rt,
		PathService:    pathService,
		ConfigService:  configService,
		AdaptorFactory: engine.NewAdaptor,
	}, nil
}

// home returns the user's home directory via Runtime.
func (s *Siphon) home() string {
	home, err := s.Runtime.GetHome()
	if err != nil {
		return ""
	}
	return home
}

// createAdaptor resolves PasswordEnv and expands ~ in Path via Runtime, then
// creates an engine adaptor.
func (s *Siphon) createAdaptor(cfg *engine.ConnectionConfig) (engine.Adaptor, error) {
	resolved := s.resolvePasswordEnv(cfg)
	resolved = expandConnectionPaths(resolved, s.home())
	factory := s.AdaptorFactory
	if factory == nil {
		factory = engine.NewAdaptor
	}
	return factory(resolved)
}

// expandConnectionPaths returns a copy of cfg with ~ expanded in path fields.
func expandConnectionPaths(cfg *engine.ConnectionConfig, home string) *engine.ConnectionConfig {
	if cfg.Path == nil {
		return cfg
	}
	expanded := ExpandPath(*cfg.Path, home)
	if expanded == *cfg.Path {
		return cfg
	}
	cp := *cfg
	cp.Path = &expanded
	return &cp
}

// resolvePasswordEnv returns a copy of cfg with Password populated from the
// environment variable named by PasswordEnv, using Runtime for env access.
func (s *Siphon) resolvePasswordEnv(cfg *engine.ConnectionConfig) *engine.ConnectionConfig {
	if cfg.PasswordEnv == nil || *cfg.PasswordEnv == "" {
		return cfg
	}
	if cfg.Password != nil && *cfg.Password != "" {
		return cfg
	}
	envVal := s.Runtime.Env().Get(*cfg.PasswordEnv)
	if envVal == "" {
		return cfg
	}
	cp := *cfg
	cp.Password = &envVal
	return &cp
}

// --- Connection management ---

// ListConnectionsOptions configures listing connections.
type ListConnectionsOptions struct {
	Format string // output format
}

// ConnectionInfo describes a configured connection for display purposes.
// Sensitive fields (password) are never included.
type ConnectionInfo struct {
	Name        string        `json:"name"`
	Engine      engine.Engine `json:"engine"`
	Host        string        `json:"host,omitempty"`
	Port        int           `json:"port,omitempty"`
	User        string        `json:"user,omitempty"`
	Database    string        `json:"database,omitempty"`
	Path        string        `json:"path,omitempty"`
	HasPassword bool          `json:"has_password"`
}

// ListConnections returns all configured connections.
func (s *Siphon) ListConnections(ctx context.Context, opts *ListConnectionsOptions) ([]ConnectionInfo, error) {
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	var connections []ConnectionInfo
	for name, cc := range cfg.Connections {
		info := ConnectionInfo{
			Name:   name,
			Engine: cc.Engine,
		}
		if cc.Host != nil {
			info.Host = *cc.Host
		}
		if cc.Port != nil {
			info.Port = *cc.Port
		}
		if cc.User != nil {
			info.User = *cc.User
		}
		if cc.Database != nil {
			info.Database = *cc.Database
		}
		if cc.Path != nil {
			info.Path = *cc.Path
		}
		info.HasPassword = (cc.Password != nil && *cc.Password != "") ||
			(cc.PasswordEnv != nil && *cc.PasswordEnv != "")
		connections = append(connections, info)
	}

	// Sort by name for deterministic output.
	sortConnectionInfos(connections)
	return connections, nil
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

// AddConnection adds a new connection to the user configuration.
func (s *Siphon) AddConnection(ctx context.Context, opts *AddConnectionOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("connection name is required")
	}

	// Load the user config (not merged) so we write back to the right file.
	userCfg, err := s.ConfigService.UserConfig(false)
	if err != nil {
		// If user config doesn't exist, start with an empty one.
		userCfg = &Config{}
	}

	if userCfg.Connections == nil {
		userCfg.Connections = make(map[string]*engine.ConnectionConfig)
	}

	if _, exists := userCfg.Connections[opts.Name]; exists {
		return ErrConnectionExists
	}

	cc := &engine.ConnectionConfig{
		Name:   opts.Name,
		Engine: opts.Engine,
	}
	if opts.Host != "" {
		cc.Host = &opts.Host
	}
	if opts.Port != 0 {
		cc.Port = &opts.Port
	}
	if opts.User != "" {
		cc.User = &opts.User
	}
	if opts.Password != "" {
		cc.Password = &opts.Password
	}
	if opts.PasswordEnv != "" {
		cc.PasswordEnv = &opts.PasswordEnv
	}
	if opts.Database != "" {
		cc.Database = &opts.Database
	}
	if opts.Path != "" {
		cc.Path = &opts.Path
	}

	userCfg.Connections[opts.Name] = cc

	path := s.PathService.UserConfig()
	if err := WriteConfig(s.Runtime, path, userCfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Invalidate config cache since we changed the underlying file.
	s.ConfigService.ResetCache()
	return nil
}

// RemoveConnectionOptions configures removing a connection.
type RemoveConnectionOptions struct {
	Name  string
	Force bool
}

// RemoveConnection removes a connection from the user configuration.
func (s *Siphon) RemoveConnection(ctx context.Context, opts *RemoveConnectionOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("connection name is required")
	}

	userCfg, err := s.ConfigService.UserConfig(false)
	if err != nil {
		return ErrConnectionNotFound
	}

	if userCfg.Connections == nil {
		return ErrConnectionNotFound
	}

	if _, exists := userCfg.Connections[opts.Name]; !exists {
		return ErrConnectionNotFound
	}

	delete(userCfg.Connections, opts.Name)

	path := s.PathService.UserConfig()
	if err := WriteConfig(s.Runtime, path, userCfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	s.ConfigService.ResetCache()
	return nil
}

// TestConnectionOptions configures testing a connection.
type TestConnectionOptions struct {
	Name    string
	Timeout time.Duration
}

// TestConnectionResult holds the result of a connection test.
type TestConnectionResult struct {
	Name    string        `json:"name"`
	Success bool          `json:"success"`
	Latency time.Duration `json:"latency"`
	Error   string        `json:"error,omitempty"`
}

// TestConnection tests connectivity to a named connection.
func (s *Siphon) TestConnection(ctx context.Context, opts *TestConnectionOptions) error {
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cc, ok := cfg.Connections[opts.Name]
	if !ok {
		return ErrConnectionNotFound
	}

	adaptor, err := s.createAdaptor(cc)
	if err != nil {
		return fmt.Errorf("creating adaptor: %w", err)
	}
	defer adaptor.Close()

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := adaptor.Connect(testCtx); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	if err := adaptor.Ping(testCtx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

// --- Backup operations ---

// BackupOptions configures a backup operation.
type BackupOptions struct {
	Connection string
	Database   string
	BackupType string // "logical", "physical", "file"
	Compress   string // "zstd", "gzip", "none"
	Name       string // explicit backup name (overrides template)
	Repo       string
	Label      string
	Message    string
	Tables     []string
	Force      bool
}

// Backup creates a database backup.
func (s *Siphon) Backup(ctx context.Context, opts *BackupOptions) (*BackupDescriptor, error) {
	return s.backupImpl(ctx, opts)
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
	return s.backupInfoImpl(ctx, opts)
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
	return s.listBackupsImpl(ctx, opts)
}

// VerifyBackupOptions configures backup verification.
type VerifyBackupOptions struct {
	BackupID string
	Path     string
}

// VerifyBackup verifies the integrity of a backup.
func (s *Siphon) VerifyBackup(ctx context.Context, opts *VerifyBackupOptions) error {
	return s.verifyBackupImpl(ctx, opts)
}

// LabelBackupOptions configures labeling a backup.
type LabelBackupOptions struct {
	BackupID string
	Label    string
}

// LabelBackup assigns or updates a label on a backup.
func (s *Siphon) LabelBackup(ctx context.Context, opts *LabelBackupOptions) error {
	return s.labelBackupImpl(ctx, opts)
}

// --- Restore operations ---

// RestoreOptions configures a restore operation.
type RestoreOptions struct {
	Connection string   // target connection name
	Path       string   // backup path or @repo/name
	BackupID   string   // alias for Path (legacy)
	Force      bool     // required for destructive restore
	Database   string   // override target database name
	Tables     []string // partial restore -- specific tables only
	Type       string   // explicit backup type override (physical/logical/file)
	Surface    Surface  // which surface is calling (for policy)
	Confirm    bool     // MCP confirmation for two-step destructive ops
}

// Restore restores a database from a backup.
func (s *Siphon) Restore(ctx context.Context, opts *RestoreOptions) error {
	return s.restoreImpl(ctx, opts)
}

// --- Transfer operations ---

// TransferOptions configures a cross-database transfer.
type TransferOptions struct {
	Source        string   // source connection:database (colon syntax)
	Target        string   // target connection:database (colon syntax)
	Tables        []string // explicit table list
	Groups        []string // named table groups from config
	ExcludeGroups []string // named table groups to exclude
	AllTables     bool     // override: transfer all tables
	OnConflict    string   // "skip", "overwrite", "merge" (default: skip)
	Surface       Surface  // which surface is calling (for policy)
	Confirm       bool     // MCP confirmation for two-step destructive ops
}

// Transfer transfers tables between databases.
func (s *Siphon) Transfer(ctx context.Context, opts *TransferOptions) error {
	return s.transferImpl(ctx, opts)
}

// --- SQL execution ---

// ExecuteSQLOptions configures raw SQL execution.
type ExecuteSQLOptions struct {
	Connection string
	Query      string  // SQL query text
	File       string  // path to SQL file (alternative to Query)
	Format     string  // output format: table, csv, json
	Database   string  // override database
	Surface    Surface // which surface is calling (for policy)
	Confirm    bool    // MCP confirmation for two-step destructive ops
}

// QueryResult holds the result of a SQL query.
type QueryResult struct {
	Columns      []string   `json:"columns"`
	Rows         [][]string `json:"rows"`
	RowsAffected int64      `json:"rows_affected"`
}

// ExecuteSQL executes a SQL query against a connection.
func (s *Siphon) ExecuteSQL(ctx context.Context, opts *ExecuteSQLOptions) (*QueryResult, error) {
	// If a file is specified and no inline query, read the file.
	if opts.File != "" && opts.Query == "" {
		data, err := s.Runtime.ReadFile(opts.File)
		if err != nil {
			return nil, fmt.Errorf("reading SQL file: %w", err)
		}
		opts.Query = string(data)
	}

	if opts.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	return s.executeSQLImpl(ctx, opts)
}

// --- Schedule operations ---

// CreateScheduleOptions configures creating a backup schedule.
type CreateScheduleOptions struct {
	Name             string
	Connection       string
	Repo             string           // backup repository
	Database         string           // override database
	BackupType       string           // physical/logical/file
	Compress         string           // zstd/gzip/none
	Message          string           // default message for backups
	BackupNameFormat string           // template override
	Time             string           // HH:MM format
	Interval         string           // daily, hourly, weekly
	Backend          string           // "launchd" or "cron"
	Retention        *RetentionPolicy // optional retention policy
	Surface          Surface
	Confirm          bool
}

// ScheduleInfo describes a configured backup schedule.
type ScheduleInfo struct {
	Name       string           `json:"name"`
	Connection string           `json:"connection"`
	Repo       string           `json:"repo,omitempty"`
	Database   string           `json:"database,omitempty"`
	BackupType string           `json:"backup_type,omitempty"`
	Compress   string           `json:"compress,omitempty"`
	Time       string           `json:"time,omitempty"`
	Interval   string           `json:"interval,omitempty"`
	Backend    string           `json:"backend,omitempty"`
	Active     bool             `json:"active"`
	Retention  *RetentionPolicy `json:"retention,omitempty"`
}

// CreateSchedule creates a new backup schedule.
func (s *Siphon) CreateSchedule(ctx context.Context, opts *CreateScheduleOptions) error {
	return s.createScheduleImpl(ctx, opts)
}

// ListSchedulesOptions configures listing schedules.
type ListSchedulesOptions struct {
	Surface Surface
}

// ListSchedules returns all configured backup schedules.
func (s *Siphon) ListSchedules(ctx context.Context, opts *ListSchedulesOptions) ([]ScheduleInfo, error) {
	return s.listSchedulesImpl(ctx, opts)
}

// RemoveScheduleOptions configures removing a schedule.
type RemoveScheduleOptions struct {
	Name    string
	Surface Surface
	Confirm bool
}

// RemoveSchedule removes a backup schedule.
func (s *Siphon) RemoveSchedule(ctx context.Context, opts *RemoveScheduleOptions) error {
	return s.removeScheduleImpl(ctx, opts)
}

// --- Config operations ---

// ConfigOptions configures reading resolved configuration.
type ConfigOptions struct {
	Explain bool
}

// ResolvedConfig holds the fully resolved configuration with optional
// per-field provenance.
type ResolvedConfig struct {
	Config     *Config           `json:"config"`
	Provenance map[string]string `json:"provenance,omitempty"` // field -> source tier
}

// Config returns the resolved configuration.
func (s *Siphon) Config(ctx context.Context, opts *ConfigOptions) (*ResolvedConfig, error) {
	cfg, err := s.ConfigService.Config(false)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	result := &ResolvedConfig{
		Config: cfg,
	}

	if opts.Explain {
		result.Provenance = buildProvenance(s.ConfigService)
	}

	return result, nil
}

// buildProvenance creates a field-to-source mapping from the ConfigService's
// resolved sources.
func buildProvenance(cs *ConfigService) map[string]string {
	prov := make(map[string]string)
	if len(cs.ResolvedSources) == 0 {
		return prov
	}

	// The resolved sources are listed most-specific first. We record
	// which sources contributed so the user can see provenance.
	for _, source := range cs.ResolvedSources {
		prov["source"] = source
		break // Most specific source wins for the summary.
	}

	// Build detailed provenance by comparing each tier. Most-specific first:
	// local config overrides project config overrides user config.
	userCfg, _ := cs.UserConfig(true)
	projectCfg, _ := cs.ProjectConfig(true)
	localCfg, _ := cs.LocalConfig(true)

	// Helper: set provenance for a field if not already set by a higher tier.
	setIfAbsent := func(key, source string) {
		if _, set := prov[key]; !set {
			prov[key] = source
		}
	}

	// Apply tiers most-specific first so higher tiers take precedence.
	tiers := []struct {
		cfg  *Config
		name string
	}{
		{localCfg, "local config"},
		{projectCfg, "project config"},
		{userCfg, "user config"},
	}

	for _, tier := range tiers {
		if tier.cfg == nil {
			continue
		}
		if tier.cfg.DefaultConnection != nil {
			setIfAbsent("default_connection", tier.name)
		}
		if tier.cfg.LogFile != nil {
			setIfAbsent("log_file", tier.name)
		}
		if tier.cfg.LogLevel != nil {
			setIfAbsent("log_level", tier.name)
		}
		if len(tier.cfg.Connections) > 0 {
			for name := range tier.cfg.Connections {
				setIfAbsent("connections."+name, tier.name)
			}
		}
	}

	return prov
}

// sortConnectionInfos sorts connection infos by name for deterministic output.
func sortConnectionInfos(infos []ConnectionInfo) {
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
}
