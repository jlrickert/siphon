# Connection Management

## Overview

Manage database connections: list, add, remove, and test connectivity.
Connections are stored in the user config (`~/.config/siphon/config.yaml`)
and can be overridden per-project (`.siphon/config.yaml`).

## Supported Engines

| Engine     | Driver               | Default Port |
| ---------- | -------------------- | ------------ |
| mariadb    | go-sql-driver/mysql  | 3306         |
| postgresql | jackc/pgx            | 5432         |
| sqlite     | modernc.org/sqlite   | N/A (file)   |

## CLI

### List connections

```bash
siphon connections list
```

Displays a table of all configured connections with name, engine, host, port,
user, database, and whether a password is set.

### Add a connection

```bash
# MariaDB
siphon connections add mydb \
  --engine mariadb \
  --host localhost \
  --port 3306 \
  --user root \
  --password-env MYSQL_ROOT_PASSWORD \
  --database myapp

# PostgreSQL
siphon connections add pgdb \
  --engine postgresql \
  --host pg.example.com \
  --port 5432 \
  --user admin \
  --database analytics

# SQLite
siphon connections add localdb \
  --engine sqlite \
  --path /data/local.db
```

Flags:

| Flag             | Description                           |
| ---------------- | ------------------------------------- |
| `--engine`       | Database engine (mariadb, postgresql, sqlite) |
| `--host`         | Database host                         |
| `--port`         | Database port                         |
| `--user`         | Database user                         |
| `--password`     | Database password (prefer `--password-env`) |
| `--password-env` | Environment variable containing password |
| `--database`     | Default database name                 |
| `--path`         | SQLite file path                      |

### Remove a connection

```bash
siphon connections remove mydb
```

### Test a connection

```bash
siphon connections test mydb
siphon connections test mydb --timeout 10s
```

Tests connectivity by establishing a connection and sending a ping. The
`--timeout` flag sets the maximum wait time (default: 5s).

## MCP Tools

- `connections_list` — List all configured connections
- `connections_add` — Add a connection (params: name, engine, host, port, user, password, password_env, database, path)
- `connections_remove` — Remove a connection (params: name, force)
- `connections_test` — Test connectivity (params: name, timeout)

## API

- `Siphon.ListConnections(ctx, *ListConnectionsOptions) ([]ConnectionInfo, error)`
- `Siphon.AddConnection(ctx, *AddConnectionOptions) error`
- `Siphon.RemoveConnection(ctx, *RemoveConnectionOptions) error`
- `Siphon.TestConnection(ctx, *TestConnectionOptions) error`

## Connection Map

The `connection_map` config section maps working directories to connections.
When siphon runs from a directory matching an entry, that connection is used
as the default.

```yaml
connection_map:
  - prefix: /home/user/myapp
    connection: dev
  - match: ".*-staging$"
    mode: regex
    connection: staging
```

Resolution: entries are evaluated in order, first match wins. Mode defaults
to "prefix" when omitted.

## Policy Engine

Per-connection, per-surface policies control which operations are allowed:

```yaml
policies:
  prod:
    cli: allow
    mcp: confirm
    api: deny
  __default__:
    mcp: confirm
```

Actions: `allow`, `confirm`, `deny`, `readonly`.

Resolution cascade:
1. Per-connection, per-surface override
2. Global default (`__default__`) per-surface
3. Hardcoded default: `allow`

## Password Security

- Passwords are never logged or displayed in list output.
- The `has_password` field indicates whether credentials are configured.
- Use `--password-env` to reference an environment variable instead of
  storing passwords in config files.
