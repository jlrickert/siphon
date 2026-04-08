package siphon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jlrickert/cli-toolkit/cfgcascade"
	"github.com/jlrickert/cli-toolkit/toolkit"
	"gopkg.in/yaml.v3"
)

// ConfigLoadWarning represents a non-fatal issue encountered while loading config.
type ConfigLoadWarning struct {
	Source  string // "user config" or "project config"
	Path    string // file path that caused the issue
	Message string // human-readable description
	Err     error  // underlying error
}

// ConfigService loads, merges, and resolves siphon configuration state using
// a four-tier cfgcascade: user config (1), project config (2), local config (3),
// env vars (4). CLI flags are handled outside the cascade.
type ConfigService struct {
	Runtime     *toolkit.Runtime
	PathService *PathService
	ConfigPath  string // explicit config path override

	// LoadWarnings accumulates non-fatal issues from the last Config() call.
	LoadWarnings []ConfigLoadWarning

	// ResolvedSources lists provider names that contributed to the merged
	// config, most-specific first.
	ResolvedSources []string

	// Cached configs.
	userCache    *Config
	projectCache *Config
	localCache   *Config
	mergedCache  *Config
}

// NewConfigService builds a ConfigService rooted at root.
func NewConfigService(root string, rt *toolkit.Runtime) (*ConfigService, error) {
	pathService, err := NewPathService(rt, root)
	if err != nil {
		return nil, err
	}
	return &ConfigService{
		Runtime:     rt,
		PathService: pathService,
	}, nil
}

// ResetCache clears all cached configuration.
func (s *ConfigService) ResetCache() {
	s.mergedCache = nil
	s.userCache = nil
	s.projectCache = nil
	s.localCache = nil
	s.LoadWarnings = nil
	s.ResolvedSources = nil
}

// UserConfig returns the global user configuration.
func (s *ConfigService) UserConfig(cache bool) (*Config, error) {
	if cache && s.userCache != nil {
		return s.userCache, nil
	}
	path := filepath.Join(s.PathService.ConfigRoot, "config.yaml")
	cfg, err := ReadConfig(s.Runtime, path)
	if err != nil {
		return nil, err
	}
	s.userCache = cfg
	return cfg, nil
}

// ProjectConfig returns the project-level configuration.
func (s *ConfigService) ProjectConfig(cache bool) (*Config, error) {
	if cache && s.projectCache != nil {
		return s.projectCache, nil
	}
	cfg, err := ReadConfig(s.Runtime, filepath.Join(s.PathService.LocalConfigRoot, "config.yaml"))
	if err != nil {
		return nil, err
	}
	s.projectCache = cfg
	return cfg, nil
}

// LocalConfig returns the local (gitignored) configuration.
func (s *ConfigService) LocalConfig(cache bool) (*Config, error) {
	if cache && s.localCache != nil {
		return s.localCache, nil
	}
	cfg, err := ReadConfig(s.Runtime, s.PathService.LocalConfig())
	if err != nil {
		return nil, err
	}
	s.localCache = cfg
	return cfg, nil
}

// Config returns the merged configuration from all tiers.
func (s *ConfigService) Config(cache bool) (*Config, error) {
	if cache && s.mergedCache != nil {
		return s.mergedCache, nil
	}

	if s.ConfigPath != "" {
		cfg, err := ReadConfig(s.Runtime, s.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config at %s: %w", s.ConfigPath, err)
		}
		if cfg == nil {
			cfg = &Config{}
		}
		s.mergedCache = cfg
		return cfg, nil
	}

	s.LoadWarnings = nil

	userPath := filepath.Join(s.PathService.ConfigRoot, "config.yaml")
	projectPath := filepath.Join(s.PathService.LocalConfigRoot, "config.yaml")
	localPath := s.PathService.LocalConfig()

	cascade := &cfgcascade.Cascade[*Config]{
		Layers: []cfgcascade.Layer[*Config]{
			{
				Rank: 1,
				Provider: &cfgcascade.FuncProvider[*Config]{
					ProviderName: "user config",
					Fn: func(_ func(string) string) (*Config, error) {
						cfg, err := s.UserConfig(cache)
						if err != nil {
							if errors.Is(err, os.ErrNotExist) {
								return nil, os.ErrNotExist
							}
							return nil, err
						}
						return cfg, nil
					},
				},
			},
			{
				Rank: 2,
				Provider: &cfgcascade.FuncProvider[*Config]{
					ProviderName: "project config",
					Fn: func(_ func(string) string) (*Config, error) {
						cfg, err := s.ProjectConfig(cache)
						if err != nil {
							if errors.Is(err, os.ErrNotExist) {
								return nil, os.ErrNotExist
							}
							return nil, err
						}
						return cfg, nil
					},
				},
			},
			{
				Rank: 3,
				Provider: &cfgcascade.FuncProvider[*Config]{
					ProviderName: "local config",
					Fn: func(_ func(string) string) (*Config, error) {
						cfg, err := s.LocalConfig(cache)
						if err != nil {
							if errors.Is(err, os.ErrNotExist) {
								return nil, os.ErrNotExist
							}
							return nil, err
						}
						return cfg, nil
					},
				},
			},
			{
				Rank: 4,
				Provider: &cfgcascade.FuncProvider[*Config]{
					ProviderName: "env vars",
					Fn: func(getenv func(string) string) (*Config, error) {
						envProvider := &cfgcascade.EnvProvider{
							ProviderName: "env vars",
							Prefix:       siphonEnvPrefix,
							Keys:         siphonEnvVarKeys,
						}
						envMap, err := envProvider.Load(getenv)
						if err != nil {
							return nil, err
						}
						cfg := configFromEnvMap(envMap)
						if cfg == nil {
							return nil, os.ErrNotExist
						}
						return cfg, nil
					},
				},
			},
		},
		MergeFn: func(base, overlay *Config) *Config {
			return MergeConfig(base, overlay)
		},
	}

	rv := cascade.Resolve(s.Runtime.Env().Get)
	s.ResolvedSources = rv.Sources

	for _, pe := range rv.Errors {
		var path string
		switch pe.Name {
		case "user config":
			path = userPath
		case "project config":
			path = projectPath
		case "local config":
			path = localPath
		}
		s.LoadWarnings = append(s.LoadWarnings, ConfigLoadWarning{
			Source:  pe.Name,
			Path:    path,
			Message: fmt.Sprintf("failed to load %s at %s: %v", pe.Name, path, pe.Err),
			Err:     pe.Err,
		})
	}

	merged := rv.Value
	if merged == nil {
		merged = &Config{}
	}

	s.mergedCache = merged
	return s.mergedCache, nil
}

// ReadConfig reads a YAML config file and returns a Config. Returns
// os.ErrNotExist if the file does not exist.
func ReadConfig(rt *toolkit.Runtime, path string) (*Config, error) {
	data, err := rt.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return &cfg, nil
}

// configFromEnvMap converts environment variable overrides to a Config struct.
func configFromEnvMap(envMap map[string]string) *Config {
	if len(envMap) == 0 {
		return nil
	}
	cfg := &Config{}
	hasValue := false
	if v, ok := envMap["SIPHON_DEFAULT_CONNECTION"]; ok && v != "" {
		cfg.DefaultConnection = &v
		hasValue = true
	}
	if v, ok := envMap["SIPHON_LOG_FILE"]; ok && v != "" {
		cfg.LogFile = &v
		hasValue = true
	}
	if v, ok := envMap["SIPHON_LOG_LEVEL"]; ok && v != "" {
		cfg.LogLevel = &v
		hasValue = true
	}
	if !hasValue {
		return nil
	}
	return cfg
}
