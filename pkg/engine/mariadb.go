package engine

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// Compile-time interface check.
var _ Adaptor = (*MariaDBAdaptor)(nil)

// MariaDBAdaptor implements the Adaptor interface for MariaDB/MySQL.
type MariaDBAdaptor struct {
	cfg *ConnectionConfig
	db  *sql.DB
}

// NewMariaDBAdaptor creates a MariaDB adaptor from configuration.
func NewMariaDBAdaptor(cfg *ConnectionConfig) *MariaDBAdaptor {
	return &MariaDBAdaptor{cfg: cfg}
}

func (a *MariaDBAdaptor) Engine() Engine {
	return EngineMariaDB
}

func (a *MariaDBAdaptor) Connect(ctx context.Context) error {
	dsn := a.buildDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	a.db = db
	return nil
}

func (a *MariaDBAdaptor) Ping(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	return a.db.PingContext(ctx)
}

func (a *MariaDBAdaptor) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

func (a *MariaDBAdaptor) ListDatabases(ctx context.Context) ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("%w: not connected", ErrConnectionFailed)
	}
	rows, err := a.db.QueryContext(ctx, "SHOW DATABASES")
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

// buildDSN constructs a MySQL DSN from the connection configuration.
func (a *MariaDBAdaptor) buildDSN() string {
	user := ""
	if a.cfg.User != nil {
		user = *a.cfg.User
	}
	password := ""
	if a.cfg.Password != nil {
		password = *a.cfg.Password
	}
	host := "localhost"
	if a.cfg.Host != nil && *a.cfg.Host != "" {
		host = *a.cfg.Host
	}
	port := 3306
	if a.cfg.Port != nil && *a.cfg.Port != 0 {
		port = *a.cfg.Port
	}
	database := ""
	if a.cfg.Database != nil {
		database = *a.cfg.Database
	}

	// Format: user:password@tcp(host:port)/database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, database)
	return dsn
}
