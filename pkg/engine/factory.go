package engine

import (
	"fmt"
	"os"
)

// NewAdaptor creates an engine adaptor for the given connection configuration.
// It dispatches on Engine type and resolves password_env if set.
func NewAdaptor(cfg *ConnectionConfig) (Adaptor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("connection config is nil")
	}

	// Resolve password from environment variable if password_env is set.
	resolved := resolvePasswordEnv(cfg)

	switch resolved.Engine {
	case EngineMariaDB:
		return NewMariaDBAdaptor(resolved), nil
	case EnginePostgreSQL:
		return NewPostgreSQLAdaptor(resolved), nil
	case EngineSQLite:
		return NewSQLiteAdaptor(resolved), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrConnectionFailed, resolved.Engine)
	}
}

// resolvePasswordEnv returns a copy of cfg with Password populated from the
// environment variable named by PasswordEnv, if set and Password is not
// already provided.
func resolvePasswordEnv(cfg *ConnectionConfig) *ConnectionConfig {
	if cfg.PasswordEnv == nil || *cfg.PasswordEnv == "" {
		return cfg
	}
	if cfg.Password != nil && *cfg.Password != "" {
		return cfg
	}

	envVal := os.Getenv(*cfg.PasswordEnv)
	if envVal == "" {
		return cfg
	}

	// Return a shallow copy with the password resolved.
	copy := *cfg
	copy.Password = &envVal
	return &copy
}
