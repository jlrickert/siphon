package engine

import "fmt"

// NewAdaptor creates an engine adaptor for the given connection configuration.
// It dispatches on Engine type. Password resolution from environment variables
// (PasswordEnv) is handled by the service layer via toolkit.Runtime, not here.
func NewAdaptor(cfg *ConnectionConfig) (Adaptor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("connection config is nil")
	}

	switch cfg.Engine {
	case EngineMariaDB:
		return NewMariaDBAdaptor(cfg), nil
	case EnginePostgreSQL:
		return NewPostgreSQLAdaptor(cfg), nil
	case EngineSQLite:
		return NewSQLiteAdaptor(cfg), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrConnectionFailed, cfg.Engine)
	}
}
