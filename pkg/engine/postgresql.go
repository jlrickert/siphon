package engine

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Compile-time interface check.
var _ Adaptor = (*PostgreSQLAdaptor)(nil)

// PostgreSQLAdaptor implements the Adaptor interface for PostgreSQL.
type PostgreSQLAdaptor struct {
	cfg *ConnectionConfig
	db  *sql.DB
}

// NewPostgreSQLAdaptor creates a PostgreSQL adaptor from configuration.
func NewPostgreSQLAdaptor(cfg *ConnectionConfig) *PostgreSQLAdaptor {
	return &PostgreSQLAdaptor{cfg: cfg}
}

func (a *PostgreSQLAdaptor) Engine() Engine {
	return EnginePostgreSQL
}

func (a *PostgreSQLAdaptor) Connect(ctx context.Context) error {
	dsn := a.buildDSN()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	a.db = db
	return nil
}

func (a *PostgreSQLAdaptor) Ping(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	return a.db.PingContext(ctx)
}

func (a *PostgreSQLAdaptor) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

func (a *PostgreSQLAdaptor) ListDatabases(ctx context.Context) ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	rows, err := a.db.QueryContext(ctx, "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
		}
		databases = append(databases, name)
	}
	return databases, rows.Err()
}

// buildDSN constructs a PostgreSQL connection string from the connection configuration.
func (a *PostgreSQLAdaptor) buildDSN() string {
	host := "localhost"
	if a.cfg.Host != nil && *a.cfg.Host != "" {
		host = *a.cfg.Host
	}
	port := 5432
	if a.cfg.Port != nil && *a.cfg.Port != 0 {
		port = *a.cfg.Port
	}
	user := ""
	if a.cfg.User != nil {
		user = *a.cfg.User
	}
	password := ""
	if a.cfg.Password != nil {
		password = *a.cfg.Password
	}
	database := ""
	if a.cfg.Database != nil {
		database = *a.cfg.Database
	}

	// Use key=value format for pgx.
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database)
	return dsn
}
