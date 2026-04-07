package siphon

import (
	"context"
	"fmt"

	"github.com/jlrickert/siphon/pkg/engine"
)

// executeSQLImpl implements the ExecuteSQL service method.
func (s *Siphon) executeSQLImpl(ctx context.Context, opts *ExecuteSQLOptions) (*QueryResult, error) {
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// 1. Resolve connection name.
	connName := opts.Connection
	if connName == "" {
		if cfg.DefaultConnection != nil {
			connName = *cfg.DefaultConnection
		}
	}
	if connName == "" {
		return nil, fmt.Errorf("connection name is required")
	}

	cc, ok := cfg.Connections[connName]
	if !ok {
		return nil, ErrConnectionNotFound
	}

	// 2. Check policy.
	surface := opts.Surface
	if surface == "" {
		surface = SurfaceCLI
	}

	action := ResolvePolicy(cfg.Policies, connName, surface)
	isWrite := IsWriteQuery(opts.Query)

	switch action {
	case PolicyDeny:
		return nil, ErrOperationDenied
	case PolicyReadonly:
		if isWrite {
			return nil, ErrOperationDenied
		}
	case PolicyConfirm:
		if isWrite && surface == SurfaceMCP && !opts.Confirm {
			return nil, ErrConfirmationRequired
		}
	case PolicyAllow:
		// proceed
	}

	// Check per-connection allow_sql policy.
	if connPolicy, ok := cfg.Policies[connName]; ok && connPolicy != nil {
		if connPolicy.AllowSQL != nil && !*connPolicy.AllowSQL {
			return nil, ErrOperationDenied
		}
	}

	// 3. Override database if specified.
	if opts.Database != "" {
		db := opts.Database
		cc = &engine.ConnectionConfig{
			Name:        cc.Name,
			Engine:      cc.Engine,
			Host:        cc.Host,
			Port:        cc.Port,
			User:        cc.User,
			Password:    cc.Password,
			PasswordEnv: cc.PasswordEnv,
			Database:    &db,
			Path:        cc.Path,
		}
	}

	// 4. Create adaptor, connect.
	factory := s.AdaptorFactory
	if factory == nil {
		factory = engine.NewAdaptor
	}

	adaptor, err := factory(cc)
	if err != nil {
		return nil, fmt.Errorf("creating adaptor: %w", err)
	}
	defer adaptor.Close()

	if err := adaptor.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connect failed: %w", err)
	}

	// 5. Check QueryAdaptor capability.
	qa, ok := engine.HasQuery(adaptor)
	if !ok {
		return nil, fmt.Errorf("%w: engine %s does not support SQL execution",
			ErrUnsupportedCapability, adaptor.Engine())
	}

	// 6. Execute.
	if isWrite {
		sqlResult, err := qa.Execute(ctx, opts.Query)
		if err != nil {
			return nil, fmt.Errorf("execute failed: %w", err)
		}
		affected, _ := sqlResult.RowsAffected()
		return &QueryResult{
			Columns:      []string{"rows_affected"},
			Rows:         [][]string{{fmt.Sprintf("%d", affected)}},
			RowsAffected: affected,
		}, nil
	}

	engineResult, err := qa.Query(ctx, opts.Query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return &QueryResult{
		Columns: engineResult.Columns,
		Rows:    engineResult.Rows,
	}, nil
}
