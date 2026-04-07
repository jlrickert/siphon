# siphon

Database management tool exposing backup, restore, transfer, SQL execution, and
connection management through three isomorphic interfaces.

## Features

- **Backup** — Physical (mariabackup), logical (mysqldump, pg_dump), and
  file-based (SQLite) backups with compression, incremental support, and
  manifest generation
- **Restore** — Engine-specific restore paths with `--force` safety, automatic
  decompression, cross-connection restore, and manifest-based type discovery
- **Transfer** — Cross-database table copying with foreign key ordering,
  configurable table groups, and conflict resolution
- **SQL execution** — Interactive, piped, and file-based SQL with table, CSV,
  and JSON output formatters
- **Connection management** — Named connections with test, add, remove, and list
  operations; directory-based connection resolution via `connectionMap`
- **Scheduling** — Automated backups via launchd (macOS) and cron (Linux) with
  retention policies

## Supported Engines

| Engine     | Role                | Backup Types                                |
| ---------- | ------------------- | ------------------------------------------- |
| MariaDB    | Primary             | Physical (mariabackup), Logical (mysqldump) |
| PostgreSQL | Secondary           | Logical (pg_dump)                           |
| SQLite     | Tertiary (embedded) | File copy                                   |

## Three Interfaces

All interfaces share the same operation surface. A missing surface is a bug.

- **CLI** — `siphon <command>` via Cobra
- **MCP server** — `siphon mcp` exposes all operations over JSON-RPC for AI
  coding agents
- **Go API** — `pkg/siphon.Siphon` service struct for programmatic use

## Quick Start

### Install

```bash
go install github.com/jlrickert/siphon/cmd/siphon@latest
```

### Configure a Connection

Create `~/.config/siphon/config.yaml`:

```yaml
connections:
  mydb:
    engine: mariadb
    host: localhost
    port: 3306
    user: root
    password_env: MYDB_PASSWORD
```

### Test Connectivity

```bash
export MYDB_PASSWORD="your-password"
siphon connections test mydb
```

### Basic Operations

```bash
# Backup
siphon backup create mydb

# Restore (requires --force)
siphon restore mydb /path/to/backup --force

# Transfer tables between connections
siphon transfer prod:myapp local:dev

# Execute SQL
siphon sql mydb "SELECT * FROM users LIMIT 10"

# List connections
siphon connections list
```

## Architecture

```
CLI command   → pkg/cli (Cobra)    → pkg/siphon.Siphon → pkg/engine.Adaptor
MCP tool call → pkg/mcp (JSON-RPC) → pkg/siphon.Siphon → pkg/engine.Adaptor
```

Both paths converge at the service layer (`pkg/siphon.Siphon`), which contains
all business logic. Engine adaptors implement database-specific operations
behind capability interfaces, supporting the full range of engines without an
adapter matrix.

See [CLAUDE.md](CLAUDE.md) for detailed architecture documentation.

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
