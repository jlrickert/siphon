package siphon

import (
	"context"
	"fmt"
	"os"
)

// ConfigScope identifies which config tier to target.
type ConfigScope string

const (
	ConfigScopeUser    ConfigScope = "user"
	ConfigScopeProject ConfigScope = "project"
	ConfigScopeLocal   ConfigScope = "local"
)

// configScopePath resolves a ConfigScope to a file path.
func (s *Siphon) configScopePath(scope ConfigScope) (string, error) {
	switch scope {
	case ConfigScopeUser:
		return s.PathService.UserConfig(), nil
	case ConfigScopeProject:
		return s.PathService.ProjectConfig(), nil
	case ConfigScopeLocal:
		return s.PathService.LocalConfig(), nil
	default:
		return "", fmt.Errorf("unknown config scope %q", scope)
	}
}

// ConfigInitOptions configures config file initialization.
type ConfigInitOptions struct {
	Scope ConfigScope
	Force bool
}

// ConfigInitResult holds the result of a config init operation.
type ConfigInitResult struct {
	Path    string `json:"path"`
	Created bool   `json:"created"`
}

// ConfigInit creates a config file at the specified scope with template content.
// Returns ErrOperationDenied if the file already exists and Force is false.
func (s *Siphon) ConfigInit(ctx context.Context, opts *ConfigInitOptions) (*ConfigInitResult, error) {
	path, err := s.configScopePath(opts.Scope)
	if err != nil {
		return nil, err
	}

	// Check if file exists.
	if _, err := s.Runtime.ReadFile(path); err == nil {
		if !opts.Force {
			return nil, fmt.Errorf("config file already exists at %s (use --force to overwrite): %w", path, ErrOperationDenied)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("checking config at %s: %w", path, err)
	}

	template := DefaultConfigTemplate()
	if err := s.Runtime.WriteFile(path, []byte(template), 0o644); err != nil {
		return nil, fmt.Errorf("writing config to %s: %w", path, err)
	}

	return &ConfigInitResult{Path: path, Created: true}, nil
}

// ConfigTemplateOptions configures template output.
type ConfigTemplateOptions struct{}

// ConfigTemplate returns an annotated YAML config template.
func (s *Siphon) ConfigTemplate(ctx context.Context, opts *ConfigTemplateOptions) (string, error) {
	return DefaultConfigTemplate(), nil
}

// ConfigEditOptions configures the edit command.
type ConfigEditOptions struct {
	Scope ConfigScope
}

// ConfigEditResult holds the resolved path for editing.
type ConfigEditResult struct {
	Path string `json:"path"`
}

// ConfigEdit resolves the config file path for the specified scope. If the file
// does not exist, it is created with template content. The CLI layer handles
// opening the file in $EDITOR.
func (s *Siphon) ConfigEdit(ctx context.Context, opts *ConfigEditOptions) (*ConfigEditResult, error) {
	path, err := s.configScopePath(opts.Scope)
	if err != nil {
		return nil, err
	}

	// Create with template if file does not exist.
	if _, err := s.Runtime.ReadFile(path); err != nil {
		if os.IsNotExist(err) {
			template := DefaultConfigTemplate()
			if err := s.Runtime.WriteFile(path, []byte(template), 0o644); err != nil {
				return nil, fmt.Errorf("creating config at %s: %w", path, err)
			}
		} else {
			return nil, fmt.Errorf("reading config at %s: %w", path, err)
		}
	}

	return &ConfigEditResult{Path: path}, nil
}

// DefaultConfigTemplate returns an annotated YAML template for a siphon config.
func DefaultConfigTemplate() string {
	return `# yaml-language-server: $schema=https://raw.githubusercontent.com/jlrickert/siphon/main/schemas/siphon-config.json
# Siphon configuration
# See: https://github.com/jlrickert/siphon
#
# version: "1"
#
# # Default connection used when none is specified.
# default_connection: dev
#
# # Fallback connection when default is unavailable.
# fallback_connection: local
#
# # Logging configuration.
# log_file: /var/log/siphon.log
# log_level: info  # debug, info, warn, error
#
# # Database connections.
# connections:
#   dev:
#     name: dev
#     engine: mariadb  # mariadb, postgresql, sqlite
#     host: localhost
#     port: 3306
#     user: root
#     password: secret
#     # password_env: MYSQL_PASSWORD  # alternative: read from env var
#     database: myapp_dev
#   local-sqlite:
#     name: local-sqlite
#     engine: sqlite
#     path: ./data/local.db
#
# # Backup repositories.
# repos:
#   nightly:
#     path: /backups/nightly
#     type: local  # local, s3
#     compress: true
#     encrypt: false
#
# # Default backup repository.
# default_repo: nightly
#
# # Operational policies.
# policies:
#   __default__:
#     cli: allow      # allow, confirm, deny, readonly
#     mcp: confirm
#     api: allow
#   production:
#     connection: prod
#     cli: confirm
#     mcp: deny
#     allow_restore: false
#     allow_transfer: false
#     allow_sql: false
#     deny_databases:
#       - mysql
#       - information_schema
#
# # Named table groups for selective backup/transfer.
# table_groups:
#   core:
#     tables:
#       - users
#       - orders
#       - products
#   logs:
#     match: "^log_.*"
#     exclude:
#       - log_debug
#
# # Scheduled backup jobs.
# schedules:
#   nightly-prod:
#     connection: prod
#     database: myapp
#     repo: nightly
#     backup_type: logical
#     time: "02:00"
#     interval: daily
#     backend: launchd  # launchd, cron
#     retention:
#       keep_count: 7
#       keep_age: 720h
#     enabled: true
#
# # Working directory to connection mapping.
# connection_map:
#   - prefix: ~/projects/myapp
#     connection: dev
#   - match: ".*staging.*"
#     mode: regex
#     connection: staging
`
}
