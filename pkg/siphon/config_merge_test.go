package siphon

import (
	"testing"

	"github.com/jlrickert/siphon/pkg/engine"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func TestMergeConnectionConfigs(t *testing.T) {
	tests := []struct {
		name    string
		base    map[string]*engine.ConnectionConfig
		overlay map[string]*engine.ConnectionConfig
		check   func(t *testing.T, result map[string]*engine.ConnectionConfig)
	}{
		{
			name:    "both nil",
			base:    nil,
			overlay: nil,
			check: func(t *testing.T, result map[string]*engine.ConnectionConfig) {
				require.Nil(t, result)
			},
		},
		{
			name: "overlay adds new key",
			base: map[string]*engine.ConnectionConfig{
				"existing": {Name: "existing", Engine: engine.EngineMariaDB},
			},
			overlay: map[string]*engine.ConnectionConfig{
				"new": {Name: "new", Engine: engine.EnginePostgreSQL, Host: ptr("newhost")},
			},
			check: func(t *testing.T, result map[string]*engine.ConnectionConfig) {
				require.Len(t, result, 2)
				require.Equal(t, "existing", result["existing"].Name)
				require.Equal(t, "new", result["new"].Name)
				require.Equal(t, "newhost", *result["new"].Host)
			},
		},
		{
			name: "overlay overrides single field preserving others",
			base: map[string]*engine.ConnectionConfig{
				"mydb": {
					Name:     "mydb",
					Engine:   engine.EngineMariaDB,
					Host:     ptr("localhost"),
					Port:     ptr(3306),
					User:     ptr("admin"),
					Password: ptr("secret"),
					Database: ptr("production"),
				},
			},
			overlay: map[string]*engine.ConnectionConfig{
				"mydb": {
					Database: ptr("staging"),
				},
			},
			check: func(t *testing.T, result map[string]*engine.ConnectionConfig) {
				require.Len(t, result, 1)
				conn := result["mydb"]
				require.Equal(t, "mydb", conn.Name)
				require.Equal(t, engine.EngineMariaDB, conn.Engine)
				require.Equal(t, "localhost", *conn.Host)
				require.Equal(t, 3306, *conn.Port)
				require.Equal(t, "admin", *conn.User)
				require.Equal(t, "secret", *conn.Password)
				require.Equal(t, "staging", *conn.Database)
			},
		},
		{
			name: "overlay with nil fields preserves base",
			base: map[string]*engine.ConnectionConfig{
				"mydb": {
					Name:   "mydb",
					Engine: engine.EngineMariaDB,
					Host:   ptr("localhost"),
					Port:   ptr(3306),
				},
			},
			overlay: map[string]*engine.ConnectionConfig{
				"mydb": {},
			},
			check: func(t *testing.T, result map[string]*engine.ConnectionConfig) {
				conn := result["mydb"]
				require.Equal(t, "mydb", conn.Name)
				require.Equal(t, engine.EngineMariaDB, conn.Engine)
				require.Equal(t, "localhost", *conn.Host)
				require.Equal(t, 3306, *conn.Port)
			},
		},
		{
			name: "overlay replaces engine and name when set",
			base: map[string]*engine.ConnectionConfig{
				"mydb": {
					Name:   "mydb",
					Engine: engine.EngineMariaDB,
					Host:   ptr("localhost"),
				},
			},
			overlay: map[string]*engine.ConnectionConfig{
				"mydb": {
					Name:   "mydb-renamed",
					Engine: engine.EnginePostgreSQL,
				},
			},
			check: func(t *testing.T, result map[string]*engine.ConnectionConfig) {
				conn := result["mydb"]
				require.Equal(t, "mydb-renamed", conn.Name)
				require.Equal(t, engine.EnginePostgreSQL, conn.Engine)
				require.Equal(t, "localhost", *conn.Host)
			},
		},
		{
			name: "nil overlay value preserves base entry",
			base: map[string]*engine.ConnectionConfig{
				"mydb": {Name: "mydb", Engine: engine.EngineMariaDB, Host: ptr("localhost")},
			},
			overlay: map[string]*engine.ConnectionConfig{
				"mydb": nil,
			},
			check: func(t *testing.T, result map[string]*engine.ConnectionConfig) {
				require.Equal(t, "mydb", result["mydb"].Name)
				require.Equal(t, "localhost", *result["mydb"].Host)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeConnectionConfigs(tt.base, tt.overlay)
			tt.check(t, result)
		})
	}
}

func TestMergeRepoConfigs(t *testing.T) {
	tests := []struct {
		name    string
		base    map[string]*RepoConfig
		overlay map[string]*RepoConfig
		check   func(t *testing.T, result map[string]*RepoConfig)
	}{
		{
			name:    "both nil",
			base:    nil,
			overlay: nil,
			check: func(t *testing.T, result map[string]*RepoConfig) {
				require.Nil(t, result)
			},
		},
		{
			name: "overlay overrides single field",
			base: map[string]*RepoConfig{
				"nightly": {
					Path:     ptr("/backups/nightly"),
					Type:     ptr("local"),
					Compress: ptr(true),
					Encrypt:  ptr(false),
				},
			},
			overlay: map[string]*RepoConfig{
				"nightly": {
					Compress: ptr(false),
				},
			},
			check: func(t *testing.T, result map[string]*RepoConfig) {
				repo := result["nightly"]
				require.Equal(t, "/backups/nightly", *repo.Path)
				require.Equal(t, "local", *repo.Type)
				require.Equal(t, false, *repo.Compress)
				require.Equal(t, false, *repo.Encrypt)
			},
		},
		{
			name: "overlay adds new key",
			base: map[string]*RepoConfig{
				"nightly": {Path: ptr("/backups/nightly")},
			},
			overlay: map[string]*RepoConfig{
				"hourly": {Path: ptr("/backups/hourly"), Type: ptr("s3")},
			},
			check: func(t *testing.T, result map[string]*RepoConfig) {
				require.Len(t, result, 2)
				require.Equal(t, "/backups/nightly", *result["nightly"].Path)
				require.Equal(t, "s3", *result["hourly"].Type)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeRepoConfigs(tt.base, tt.overlay)
			tt.check(t, result)
		})
	}
}

func TestMergePolicyConfigs(t *testing.T) {
	allow := PolicyAllow
	deny := PolicyDeny
	confirm := PolicyConfirm

	tests := []struct {
		name    string
		base    map[string]*PolicyConfig
		overlay map[string]*PolicyConfig
		check   func(t *testing.T, result map[string]*PolicyConfig)
	}{
		{
			name:    "both nil",
			base:    nil,
			overlay: nil,
			check: func(t *testing.T, result map[string]*PolicyConfig) {
				require.Nil(t, result)
			},
		},
		{
			name: "overlay overrides single policy field",
			base: map[string]*PolicyConfig{
				"prod": {
					CLI:          &allow,
					MCP:          &deny,
					AllowRestore: ptr(false),
					AllowSQL:     ptr(true),
				},
			},
			overlay: map[string]*PolicyConfig{
				"prod": {
					MCP: &confirm,
				},
			},
			check: func(t *testing.T, result map[string]*PolicyConfig) {
				p := result["prod"]
				require.Equal(t, PolicyAllow, *p.CLI)
				require.Equal(t, PolicyConfirm, *p.MCP)
				require.Equal(t, false, *p.AllowRestore)
				require.Equal(t, true, *p.AllowSQL)
			},
		},
		{
			name: "overlay DenyDatabases replaces base",
			base: map[string]*PolicyConfig{
				"prod": {
					DenyDatabases: []string{"mysql", "information_schema"},
					AllowSQL:      ptr(true),
				},
			},
			overlay: map[string]*PolicyConfig{
				"prod": {
					DenyDatabases: []string{"mysql", "information_schema", "sys"},
				},
			},
			check: func(t *testing.T, result map[string]*PolicyConfig) {
				p := result["prod"]
				require.Equal(t, []string{"mysql", "information_schema", "sys"}, p.DenyDatabases)
				require.Equal(t, true, *p.AllowSQL)
			},
		},
		{
			name: "empty overlay DenyDatabases preserves base",
			base: map[string]*PolicyConfig{
				"prod": {
					DenyDatabases: []string{"mysql"},
				},
			},
			overlay: map[string]*PolicyConfig{
				"prod": {
					AllowSQL: ptr(false),
				},
			},
			check: func(t *testing.T, result map[string]*PolicyConfig) {
				p := result["prod"]
				require.Equal(t, []string{"mysql"}, p.DenyDatabases)
				require.Equal(t, false, *p.AllowSQL)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergePolicyConfigs(tt.base, tt.overlay)
			tt.check(t, result)
		})
	}
}

func TestMergeConfig_DeepMerge(t *testing.T) {
	t.Run("full config merge preserves connection fields across tiers", func(t *testing.T) {
		user := &Config{
			DefaultConnection: ptr("prod"),
			LogLevel:          ptr("info"),
			Connections: map[string]*engine.ConnectionConfig{
				"prod": {
					Name:     "prod",
					Engine:   engine.EngineMariaDB,
					Host:     ptr("db.example.com"),
					Port:     ptr(3306),
					User:     ptr("admin"),
					Password: ptr("userpass"),
					Database: ptr("myapp"),
				},
			},
			Repos: map[string]*RepoConfig{
				"nightly": {
					Path:     ptr("/backups/nightly"),
					Type:     ptr("local"),
					Compress: ptr(true),
				},
			},
		}

		project := &Config{
			LogLevel: ptr("warn"),
			Connections: map[string]*engine.ConnectionConfig{
				"prod": {
					Database: ptr("myapp_staging"),
				},
			},
			Repos: map[string]*RepoConfig{
				"nightly": {
					Compress: ptr(false),
				},
			},
		}

		merged := MergeConfig(user, project)

		// Scalars: project overrides user.
		require.Equal(t, "prod", *merged.DefaultConnection)
		require.Equal(t, "warn", *merged.LogLevel)

		// Connection: project overrides only database, rest preserved.
		conn := merged.Connections["prod"]
		require.Equal(t, "prod", conn.Name)
		require.Equal(t, engine.EngineMariaDB, conn.Engine)
		require.Equal(t, "db.example.com", *conn.Host)
		require.Equal(t, 3306, *conn.Port)
		require.Equal(t, "admin", *conn.User)
		require.Equal(t, "userpass", *conn.Password)
		require.Equal(t, "myapp_staging", *conn.Database)

		// Repo: project overrides only compress, rest preserved.
		repo := merged.Repos["nightly"]
		require.Equal(t, "/backups/nightly", *repo.Path)
		require.Equal(t, "local", *repo.Type)
		require.Equal(t, false, *repo.Compress)
	})

	t.Run("schedule and table group use key-level replacement", func(t *testing.T) {
		base := &Config{
			Schedules: map[string]*ScheduleConfig{
				"nightly": {
					Connection: ptr("prod"),
					Database:   ptr("myapp"),
					Time:       ptr("02:00"),
					Interval:   ptr("daily"),
				},
			},
			TableGroups: map[string]*TableGroup{
				"core": {
					Tables: []string{"users", "orders"},
				},
			},
		}

		overlay := &Config{
			Schedules: map[string]*ScheduleConfig{
				"nightly": {
					Time: ptr("03:00"),
				},
			},
			TableGroups: map[string]*TableGroup{
				"core": {
					Tables: []string{"users", "orders", "products"},
				},
			},
		}

		merged := MergeConfig(base, overlay)

		// Schedule: key-level replacement -- only Time is set.
		sched := merged.Schedules["nightly"]
		require.Equal(t, "03:00", *sched.Time)
		require.Nil(t, sched.Connection, "key-level replacement should not preserve base fields")
		require.Nil(t, sched.Database)

		// TableGroup: key-level replacement.
		tg := merged.TableGroups["core"]
		require.Equal(t, []string{"users", "orders", "products"}, tg.Tables)
	})
}
