package siphon_test

import (
	"context"
	"embed"
	"testing"

	"github.com/jlrickert/cli-toolkit/sandbox"
	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/jlrickert/siphon/pkg/siphon"
	"github.com/stretchr/testify/require"
)

//go:embed all:data/**
var testdata embed.FS

func newTestSiphon(t *testing.T) (*siphon.Siphon, *sandbox.Sandbox) {
	t.Helper()

	sb := sandbox.NewSandbox(t, &sandbox.Options{
		Data: testdata,
		Home: "/home/testuser",
		User: "testuser",
	}, sandbox.WithFixture("testuser", "~"))

	s, err := siphon.New(siphon.SiphonOptions{
		Runtime: sb.Runtime(),
	})
	require.NoError(t, err)
	return s, sb
}

func TestListConnections(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphon(t)
	ctx := context.Background()

	connections, err := s.ListConnections(ctx, &siphon.ListConnectionsOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, connections)

	// The test fixture has a "dev" connection.
	found := false
	for _, c := range connections {
		if c.Name == "dev" {
			found = true
			require.Equal(t, engine.EngineMariaDB, c.Engine)
			require.Equal(t, "localhost", c.Host)
			require.Equal(t, 3306, c.Port)
			break
		}
	}
	require.True(t, found, "expected 'dev' connection in list")
}

func TestAddConnection(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphon(t)
	ctx := context.Background()

	err := s.AddConnection(ctx, &siphon.AddConnectionOptions{
		Name:     "staging",
		Engine:   engine.EnginePostgreSQL,
		Host:     "staging.example.com",
		Port:     5432,
		User:     "admin",
		Database: "staging_db",
	})
	require.NoError(t, err)

	// Verify it appears in the list.
	connections, err := s.ListConnections(ctx, &siphon.ListConnectionsOptions{})
	require.NoError(t, err)

	found := false
	for _, c := range connections {
		if c.Name == "staging" {
			found = true
			require.Equal(t, engine.EnginePostgreSQL, c.Engine)
			require.Equal(t, "staging.example.com", c.Host)
			require.Equal(t, 5432, c.Port)
			break
		}
	}
	require.True(t, found, "expected 'staging' connection after add")
}

func TestAddConnection_DuplicateReturnsError(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphon(t)
	ctx := context.Background()

	err := s.AddConnection(ctx, &siphon.AddConnectionOptions{
		Name:   "dev",
		Engine: engine.EngineMariaDB,
	})
	require.ErrorIs(t, err, siphon.ErrConnectionExists)
}

func TestRemoveConnection(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphon(t)
	ctx := context.Background()

	// First add, then remove.
	err := s.AddConnection(ctx, &siphon.AddConnectionOptions{
		Name:   "temp",
		Engine: engine.EngineSQLite,
		Path:   "/tmp/test.db",
	})
	require.NoError(t, err)

	err = s.RemoveConnection(ctx, &siphon.RemoveConnectionOptions{
		Name: "temp",
	})
	require.NoError(t, err)

	// Verify it's gone.
	connections, err := s.ListConnections(ctx, &siphon.ListConnectionsOptions{})
	require.NoError(t, err)
	for _, c := range connections {
		require.NotEqual(t, "temp", c.Name)
	}
}

func TestRemoveConnection_NotFound(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphon(t)
	ctx := context.Background()

	err := s.RemoveConnection(ctx, &siphon.RemoveConnectionOptions{
		Name: "nonexistent",
	})
	require.ErrorIs(t, err, siphon.ErrConnectionNotFound)
}

func TestAddConnection_SQLiteWithPath(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphon(t)
	ctx := context.Background()

	err := s.AddConnection(ctx, &siphon.AddConnectionOptions{
		Name:   "local-db",
		Engine: engine.EngineSQLite,
		Path:   "/data/local.db",
	})
	require.NoError(t, err)

	connections, err := s.ListConnections(ctx, &siphon.ListConnectionsOptions{})
	require.NoError(t, err)

	found := false
	for _, c := range connections {
		if c.Name == "local-db" {
			found = true
			require.Equal(t, engine.EngineSQLite, c.Engine)
			require.Equal(t, "/data/local.db", c.Path)
			break
		}
	}
	require.True(t, found)
}

func TestAddConnection_PasswordEnv(t *testing.T) {
	t.Parallel()
	s, _ := newTestSiphon(t)
	ctx := context.Background()

	err := s.AddConnection(ctx, &siphon.AddConnectionOptions{
		Name:        "secure",
		Engine:      engine.EngineMariaDB,
		Host:        "db.example.com",
		PasswordEnv: "DB_PASSWORD",
	})
	require.NoError(t, err)

	connections, err := s.ListConnections(ctx, &siphon.ListConnectionsOptions{})
	require.NoError(t, err)

	found := false
	for _, c := range connections {
		if c.Name == "secure" {
			found = true
			require.True(t, c.HasPassword, "password_env should count as having password")
			break
		}
	}
	require.True(t, found)
}
