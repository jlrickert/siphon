//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/jlrickert/siphon/pkg/engine"
)

// MariaDB test container configuration.
const (
	MariaDBHost     = "127.0.0.1"
	MariaDBPort     = 3307
	MariaDBUser     = "siphon"
	MariaDBPassword = "siphonpass"
	MariaDBDatabase = "testdb"
	MariaDBRootPass = "testpass"
)

// PostgreSQL test container configuration.
const (
	PostgreSQLHost     = "127.0.0.1"
	PostgreSQLPort     = 5433
	PostgreSQLUser     = "siphon"
	PostgreSQLPassword = "siphonpass"
	PostgreSQLDatabase = "testdb"
)

// MariaDBConfig returns a ConnectionConfig for the MariaDB test container.
func MariaDBConfig() *engine.ConnectionConfig {
	host := MariaDBHost
	port := MariaDBPort
	user := MariaDBUser
	password := MariaDBPassword
	database := MariaDBDatabase
	return &engine.ConnectionConfig{
		Name:     "test-mariadb",
		Engine:   engine.EngineMariaDB,
		Host:     &host,
		Port:     &port,
		User:     &user,
		Password: &password,
		Database: &database,
	}
}

// PostgreSQLConfig returns a ConnectionConfig for the PostgreSQL test container.
func PostgreSQLConfig() *engine.ConnectionConfig {
	host := PostgreSQLHost
	port := PostgreSQLPort
	user := PostgreSQLUser
	password := PostgreSQLPassword
	database := PostgreSQLDatabase
	return &engine.ConnectionConfig{
		Name:     "test-postgresql",
		Engine:   engine.EnginePostgreSQL,
		Host:     &host,
		Port:     &port,
		User:     &user,
		Password: &password,
		Database: &database,
	}
}

// RequireMariaDB skips the test if the MariaDB container is not reachable.
func RequireMariaDB(t *testing.T) {
	t.Helper()
	if !isPortReachable(MariaDBHost, MariaDBPort) {
		t.Skip("MariaDB container not available on port 3307; run 'docker compose up -d --wait'")
	}
}

// RequirePostgreSQL skips the test if the PostgreSQL container is not reachable.
func RequirePostgreSQL(t *testing.T) {
	t.Helper()
	if !isPortReachable(PostgreSQLHost, PostgreSQLPort) {
		t.Skip("PostgreSQL container not available on port 5433; run 'docker compose up -d --wait'")
	}
}

// ConnectMariaDB creates and connects a MariaDB adaptor, failing the test if
// unable to connect. The caller should defer adaptor.Close().
func ConnectMariaDB(t *testing.T) engine.Adaptor {
	t.Helper()
	RequireMariaDB(t)

	adaptor, err := engine.NewAdaptor(MariaDBConfig())
	if err != nil {
		t.Fatalf("failed to create MariaDB adaptor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := adaptor.Connect(ctx); err != nil {
		t.Fatalf("failed to connect to MariaDB: %v", err)
	}

	return adaptor
}

// ConnectPostgreSQL creates and connects a PostgreSQL adaptor, failing the test
// if unable to connect. The caller should defer adaptor.Close().
func ConnectPostgreSQL(t *testing.T) engine.Adaptor {
	t.Helper()
	RequirePostgreSQL(t)

	adaptor, err := engine.NewAdaptor(PostgreSQLConfig())
	if err != nil {
		t.Fatalf("failed to create PostgreSQL adaptor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := adaptor.Connect(ctx); err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}

	return adaptor
}

// isPortReachable attempts a TCP connection to host:port with a short timeout.
func isPortReachable(host string, port int) bool {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
