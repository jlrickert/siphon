package siphon

import (
	"github.com/jlrickert/siphon/pkg/engine"
)

// Config is the top-level siphon configuration. Scalars use pointer types to
// distinguish "explicitly set to zero" from "absent" during cascade merging.
type Config struct {
	Version            *string                       `yaml:"version,omitempty" json:"version,omitempty"`
	DefaultConnection  *string                       `yaml:"default_connection,omitempty" json:"default_connection,omitempty"`
	FallbackConnection *string                       `yaml:"fallback_connection,omitempty" json:"fallback_connection,omitempty"`
	LogFile            *string                       `yaml:"log_file,omitempty" json:"log_file,omitempty"`
	LogLevel           *string                       `yaml:"log_level,omitempty" json:"log_level,omitempty"`
	Connections        map[string]*engine.ConnectionConfig `yaml:"connections,omitempty" json:"connections,omitempty"`
	Repos              map[string]*RepoConfig        `yaml:"repos,omitempty" json:"repos,omitempty"`
	DefaultRepo        *string                       `yaml:"default_repo,omitempty" json:"default_repo,omitempty"`
	Policies           map[string]*PolicyConfig      `yaml:"policies,omitempty" json:"policies,omitempty"`
	TableGroups        map[string]*TableGroup        `yaml:"table_groups,omitempty" json:"table_groups,omitempty"`
	Schedules          map[string]*ScheduleConfig    `yaml:"schedules,omitempty" json:"schedules,omitempty"`
	ConnectionMap      []ConnectionMapEntry          `yaml:"connection_map,omitempty" json:"connection_map,omitempty"`
	BackupNameFormat   *string                       `yaml:"backup_name_format,omitempty" json:"backup_name_format,omitempty"`
}

// RepoConfig defines a backup repository location.
type RepoConfig struct {
	Path     *string `yaml:"path,omitempty" json:"path,omitempty"`
	Type     *string `yaml:"type,omitempty" json:"type,omitempty"` // "local", "s3", etc.
	Compress *bool   `yaml:"compress,omitempty" json:"compress,omitempty"`
	Encrypt  *bool   `yaml:"encrypt,omitempty" json:"encrypt,omitempty"`
}

// PolicyConfig defines operational policies for connections.
type PolicyConfig struct {
	Connection    *string  `yaml:"connection,omitempty" json:"connection,omitempty"`
	AllowRestore  *bool    `yaml:"allow_restore,omitempty" json:"allow_restore,omitempty"`
	AllowTransfer *bool    `yaml:"allow_transfer,omitempty" json:"allow_transfer,omitempty"`
	AllowSQL      *bool    `yaml:"allow_sql,omitempty" json:"allow_sql,omitempty"`
	DenyDatabases []string `yaml:"deny_databases,omitempty" json:"deny_databases,omitempty"`
}

// TableGroup defines a named group of tables. It supports two modes:
// a simple list of table names, or a regex-based pattern match.
type TableGroup struct {
	Tables  []string `yaml:"tables,omitempty" json:"tables,omitempty"`
	Pattern *string  `yaml:"pattern,omitempty" json:"pattern,omitempty"` // regex mode
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// ScheduleConfig defines a scheduled backup job.
type ScheduleConfig struct {
	Connection *string `yaml:"connection,omitempty" json:"connection,omitempty"`
	Database   *string `yaml:"database,omitempty" json:"database,omitempty"`
	Cron       *string `yaml:"cron,omitempty" json:"cron,omitempty"`
	Repo       *string `yaml:"repo,omitempty" json:"repo,omitempty"`
	BackupType *string `yaml:"backup_type,omitempty" json:"backup_type,omitempty"` // "logical", "physical", "file"
	Retain     *int    `yaml:"retain,omitempty" json:"retain,omitempty"`
	Enabled    *bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

// ConnectionMapEntry maps a connection alias to its environment context.
type ConnectionMapEntry struct {
	Alias       string `yaml:"alias" json:"alias"`
	Environment string `yaml:"environment,omitempty" json:"environment,omitempty"` // "dev", "staging", "prod"
}

// MergeConfig merges overlay on top of base. Scalars use pointer-based
// override; maps use recursive deep merge; lists append with dedup.
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

	// Deep-merge maps.
	merged.Connections = mergeMaps(base.Connections, overlay.Connections)
	merged.Repos = mergeMaps(base.Repos, overlay.Repos)
	merged.Policies = mergeMaps(base.Policies, overlay.Policies)
	merged.TableGroups = mergeMaps(base.TableGroups, overlay.TableGroups)
	merged.Schedules = mergeMaps(base.Schedules, overlay.Schedules)

	// Append-with-dedup for connection map.
	if len(overlay.ConnectionMap) > 0 {
		seen := make(map[string]bool)
		for _, e := range base.ConnectionMap {
			seen[e.Alias] = true
		}
		combined := append([]ConnectionMapEntry{}, base.ConnectionMap...)
		for _, e := range overlay.ConnectionMap {
			if !seen[e.Alias] {
				combined = append(combined, e)
				seen[e.Alias] = true
			}
		}
		merged.ConnectionMap = combined
	}

	return &merged
}

// mergeMaps merges two maps, with overlay values taking precedence.
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
