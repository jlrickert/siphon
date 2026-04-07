# siphon documentation

User-facing documentation for siphon features and operations.

## Features

| Feature | Description | Doc |
|---------|-------------|-----|
| [Connection Management](features/connections.md) | List, add, remove, and test database connections | connections.md |
| [Backup](features/backup.md) | Physical, logical, and file-based database backups | backup.md |
| [Restore](features/restore.md) | Restore databases from backups with safety controls | restore.md |
| [Transfer](features/transfer.md) | Cross-database table transfer with FK ordering | transfer.md |
| [SQL Execution](features/sql.md) | Execute raw SQL with multiple output formats | sql.md |
| [Configuration](features/config.md) | Five-tier config cascade with per-field provenance | config.md |
| [Scheduling](features/scheduling.md) | Automated backups via launchd and cron | scheduling.md |

## Supported Engines

| Engine     | Role                | Backup Types                                |
|------------|---------------------|---------------------------------------------|
| MariaDB    | Primary             | Physical (mariabackup), Logical (mysqldump) |
| PostgreSQL | Secondary           | Logical (pg_dump)                           |
| SQLite     | Tertiary (embedded) | File copy                                   |

## Three Interfaces

All interfaces share the same operation surface:

- **CLI** -- `siphon <command>` via Cobra
- **MCP server** -- `siphon mcp` exposes all operations over JSON-RPC
- **Go API** -- `pkg/siphon.Siphon` service struct for programmatic use

## Architecture

```
CLI command   -> pkg/cli (Cobra)    -> pkg/siphon.Siphon -> pkg/engine.Adaptor
MCP tool call -> pkg/mcp (JSON-RPC) -> pkg/siphon.Siphon -> pkg/engine.Adaptor
```

See [CLAUDE.md](../CLAUDE.md) for detailed architecture and developer guidance.
