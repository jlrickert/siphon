package siphon

import (
	"github.com/jlrickert/siphon/pkg/engine"
)

// Config is the top-level siphon configuration. Scalars use pointer types to
// distinguish "explicitly set to zero" from "absent" during cascade merging.
type Config struct {
	Version            *string                             `yaml:"version,omitempty" json:"version,omitempty"`
	DefaultConnection  *string                             `yaml:"default_connection,omitempty" json:"default_connection,omitempty"`
	FallbackConnection *string                             `yaml:"fallback_connection,omitempty" json:"fallback_connection,omitempty"`
	LogFile            *string                             `yaml:"log_file,omitempty" json:"log_file,omitempty"`
	LogLevel           *string                             `yaml:"log_level,omitempty" json:"log_level,omitempty"`
	Connections        map[string]*engine.ConnectionConfig `yaml:"connections,omitempty" json:"connections,omitempty"`
	Repos              map[string]*RepoConfig              `yaml:"repos,omitempty" json:"repos,omitempty"`
	DefaultRepo        *string                             `yaml:"default_repo,omitempty" json:"default_repo,omitempty"`
	Policies           map[string]*PolicyConfig            `yaml:"policies,omitempty" json:"policies,omitempty"`
	TableGroups        map[string]*TableGroup              `yaml:"table_groups,omitempty" json:"table_groups,omitempty"`
	Schedules          map[string]*ScheduleConfig          `yaml:"schedules,omitempty" json:"schedules,omitempty"`
	ConnectionMap      []ConnectionMapEntry                `yaml:"connection_map,omitempty" json:"connection_map,omitempty"`
	BackupNameFormat   *string                             `yaml:"backup_name_format,omitempty" json:"backup_name_format,omitempty"`
}

// RepoConfig defines a backup repository location.
type RepoConfig struct {
	Path     *string `yaml:"path,omitempty" json:"path,omitempty"`
	Type     *string `yaml:"type,omitempty" json:"type,omitempty"` // "local", "s3", etc.
	Compress *bool   `yaml:"compress,omitempty" json:"compress,omitempty"`
	Encrypt  *bool   `yaml:"encrypt,omitempty" json:"encrypt,omitempty"`
}

// PolicyConfig defines operational policies for connections. It includes both
// per-operation controls (allow_restore, etc.) and per-surface action overrides
// (cli, mcp, api).
type PolicyConfig struct {
	Connection    *string       `yaml:"connection,omitempty" json:"connection,omitempty"`
	CLI           *PolicyAction `yaml:"cli,omitempty" json:"cli,omitempty"`
	MCP           *PolicyAction `yaml:"mcp,omitempty" json:"mcp,omitempty"`
	API           *PolicyAction `yaml:"api,omitempty" json:"api,omitempty"`
	AllowRestore  *bool         `yaml:"allow_restore,omitempty" json:"allow_restore,omitempty"`
	AllowTransfer *bool         `yaml:"allow_transfer,omitempty" json:"allow_transfer,omitempty"`
	AllowSQL      *bool         `yaml:"allow_sql,omitempty" json:"allow_sql,omitempty"`
	DenyDatabases []string      `yaml:"deny_databases,omitempty" json:"deny_databases,omitempty"`
}

// TableGroup defines a named group of tables. It supports two modes:
// a simple list of table names, or a regex-based pattern match.
type TableGroup struct {
	Tables  []string `yaml:"tables,omitempty" json:"tables,omitempty"`
	Match   *string  `yaml:"match,omitempty" json:"match,omitempty"` // regex mode
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// ScheduleConfig defines a scheduled backup job.
type ScheduleConfig struct {
	Connection *string          `yaml:"connection,omitempty" json:"connection,omitempty"`
	Database   *string          `yaml:"database,omitempty" json:"database,omitempty"`
	Repo       *string          `yaml:"repo,omitempty" json:"repo,omitempty"`
	BackupType *string          `yaml:"backup_type,omitempty" json:"backup_type,omitempty"` // "logical", "physical", "file"
	Compress   *string          `yaml:"compress,omitempty" json:"compress,omitempty"`
	Time       *string          `yaml:"time,omitempty" json:"time,omitempty"`         // HH:MM
	Interval   *string          `yaml:"interval,omitempty" json:"interval,omitempty"` // daily, hourly, weekly
	Backend    *string          `yaml:"backend,omitempty" json:"backend,omitempty"`   // "launchd" or "cron"
	Retention  *RetentionPolicy `yaml:"retention,omitempty" json:"retention,omitempty"`
	Enabled    *bool            `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

// ConnectionMapEntry maps a working directory pattern to a connection name.
// When a user runs siphon from a directory matching the pattern, the mapped
// connection is used as the default.
type ConnectionMapEntry struct {
	Prefix     *string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	Match      *string `yaml:"match,omitempty" json:"match,omitempty"`
	Mode       *string `yaml:"mode,omitempty" json:"mode,omitempty"` // "prefix" (default) or "regex"
	Connection string  `yaml:"connection" json:"connection"`
}

// MergeConfig merges overlay on top of base. Scalars use pointer-based
// override; maps use field-level deep merge for ConnectionConfig, RepoConfig,
// and PolicyConfig; ScheduleConfig and TableGroup use key-level replacement
// (last-wins-per-key); ConnectionMap uses append-with-dedup.
func MergeConfig(base, overlay *Config) *Config {
	if base == nil {
		return overlay
	}
	if overlay == nil {
		return base
	}

	merged := *base

	if overlay.Version != nil {
		merged.Version = overlay.Version
	}
	if overlay.DefaultConnection != nil {
		merged.DefaultConnection = overlay.DefaultConnection
	}
	if overlay.FallbackConnection != nil {
		merged.FallbackConnection = overlay.FallbackConnection
	}
	if overlay.LogFile != nil {
		merged.LogFile = overlay.LogFile
	}
	if overlay.LogLevel != nil {
		merged.LogLevel = overlay.LogLevel
	}
	if overlay.DefaultRepo != nil {
		merged.DefaultRepo = overlay.DefaultRepo
	}
	if overlay.BackupNameFormat != nil {
		merged.BackupNameFormat = overlay.BackupNameFormat
	}

	// Field-level deep merge for types with pointer-based override fields.
	merged.Connections = mergeConnectionConfigs(base.Connections, overlay.Connections)
	merged.Repos = mergeRepoConfigs(base.Repos, overlay.Repos)
	merged.Policies = mergePolicyConfigs(base.Policies, overlay.Policies)

	// Key-level replacement for self-contained entry types.
	merged.TableGroups = mergeMaps(base.TableGroups, overlay.TableGroups)
	merged.Schedules = mergeMaps(base.Schedules, overlay.Schedules)

	// Append-with-dedup for connection map. Dedup key is the connection name.
	if len(overlay.ConnectionMap) > 0 {
		seen := make(map[string]bool)
		for _, e := range base.ConnectionMap {
			seen[e.Connection] = true
		}
		combined := append([]ConnectionMapEntry{}, base.ConnectionMap...)
		for _, e := range overlay.ConnectionMap {
			if !seen[e.Connection] {
				combined = append(combined, e)
				seen[e.Connection] = true
			}
		}
		merged.ConnectionMap = combined
	}

	return &merged
}

// mergeConnectionConfigs deep-merges two connection maps. For matching keys,
// individual pointer fields from the overlay override the base; nil fields in
// the overlay preserve the base value.
func mergeConnectionConfigs(base, overlay map[string]*engine.ConnectionConfig) map[string]*engine.ConnectionConfig {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	result := make(map[string]*engine.ConnectionConfig)
	for k, v := range base {
		result[k] = v
	}
	for k, ov := range overlay {
		bv, exists := result[k]
		if !exists || bv == nil {
			result[k] = ov
			continue
		}
		if ov == nil {
			continue
		}
		merged := *bv
		if ov.Name != "" {
			merged.Name = ov.Name
		}
		if ov.Engine != "" {
			merged.Engine = ov.Engine
		}
		if ov.Host != nil {
			merged.Host = ov.Host
		}
		if ov.Port != nil {
			merged.Port = ov.Port
		}
		if ov.User != nil {
			merged.User = ov.User
		}
		if ov.Password != nil {
			merged.Password = ov.Password
		}
		if ov.PasswordEnv != nil {
			merged.PasswordEnv = ov.PasswordEnv
		}
		if ov.Database != nil {
			merged.Database = ov.Database
		}
		if ov.Path != nil {
			merged.Path = ov.Path
		}
		result[k] = &merged
	}
	return result
}

// mergeRepoConfigs deep-merges two repo maps. For matching keys, individual
// pointer fields from the overlay override the base.
func mergeRepoConfigs(base, overlay map[string]*RepoConfig) map[string]*RepoConfig {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	result := make(map[string]*RepoConfig)
	for k, v := range base {
		result[k] = v
	}
	for k, ov := range overlay {
		bv, exists := result[k]
		if !exists || bv == nil {
			result[k] = ov
			continue
		}
		if ov == nil {
			continue
		}
		merged := *bv
		if ov.Path != nil {
			merged.Path = ov.Path
		}
		if ov.Type != nil {
			merged.Type = ov.Type
		}
		if ov.Compress != nil {
			merged.Compress = ov.Compress
		}
		if ov.Encrypt != nil {
			merged.Encrypt = ov.Encrypt
		}
		result[k] = &merged
	}
	return result
}

// mergePolicyConfigs deep-merges two policy maps. For matching keys, individual
// pointer fields from the overlay override the base. DenyDatabases slices from
// the overlay replace the base (not appended).
func mergePolicyConfigs(base, overlay map[string]*PolicyConfig) map[string]*PolicyConfig {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	result := make(map[string]*PolicyConfig)
	for k, v := range base {
		result[k] = v
	}
	for k, ov := range overlay {
		bv, exists := result[k]
		if !exists || bv == nil {
			result[k] = ov
			continue
		}
		if ov == nil {
			continue
		}
		merged := *bv
		if ov.Connection != nil {
			merged.Connection = ov.Connection
		}
		if ov.CLI != nil {
			merged.CLI = ov.CLI
		}
		if ov.MCP != nil {
			merged.MCP = ov.MCP
		}
		if ov.API != nil {
			merged.API = ov.API
		}
		if ov.AllowRestore != nil {
			merged.AllowRestore = ov.AllowRestore
		}
		if ov.AllowTransfer != nil {
			merged.AllowTransfer = ov.AllowTransfer
		}
		if ov.AllowSQL != nil {
			merged.AllowSQL = ov.AllowSQL
		}
		if len(ov.DenyDatabases) > 0 {
			merged.DenyDatabases = ov.DenyDatabases
		}
		result[k] = &merged
	}
	return result
}

// mergeMaps merges two maps, with overlay values taking precedence (key-level
// replacement). Used for ScheduleConfig and TableGroup where entries are
// self-contained and partial override doesn't apply.
func mergeMaps[V any](base, overlay map[string]V) map[string]V {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	result := make(map[string]V)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overlay {
		result[k] = v
	}
	return result
}
