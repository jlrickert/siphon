# Restore

## Overview

Restore a database from a backup. This is a destructive operation that
requires the `--force` flag. For MCP tool calls, `confirm: true` is also
required.

Restore supports three backup types, mirroring the backup capabilities:

- **logical** -- Load a SQL dump via `mariadb` or `pg_restore` (MariaDB, PostgreSQL)
- **physical** -- Prepare + copy-back via `mariabackup` (MariaDB only)
- **file** -- File-level copy restore (SQLite)

The backup type is auto-detected from the `backup.meta.yaml` manifest. If
no manifest exists, use `--type` to specify the backup type explicitly.

## CLI

### Basic restore

```bash
# Restore from a local path
siphon restore dev /path/to/backup --force

# Restore using @repo syntax
siphon restore dev @nightly/my-backup --force
```

### Database redirect

Restore to a different database than the one recorded in the backup:

```bash
siphon restore dev /path/to/backup --force --database staging_db
```

### Partial restore (tables)

Restore specific tables only (logical backups only):

```bash
siphon restore dev /path/to/backup --force --tables users,orders
```

Note: `--tables` is not supported for physical backups and will return
an error.

### Explicit type override

When restoring from a directory without a manifest:

```bash
siphon restore dev /path/to/backup --force --type logical
```

### Cross-connection restore

Restore a backup from one connection to a different target:

```bash
# Backup was created from "prod" connection, restore to "staging"
siphon restore staging /path/to/prod-backup --force
```

### Flags

| Flag         | Short | Description                                      |
| ------------ | ----- | ------------------------------------------------ |
| `--force`    | `-f`  | Required for destructive restore                 |
| `--database` |       | Override target database name                    |
| `--tables`   |       | Comma-separated list of tables (logical only)    |
| `--type`     |       | Explicit backup type (physical, logical, file)   |

## MCP Tools

- `restore` -- Restore a database from a backup (requires `confirm: true`)

### Parameters

| Parameter    | Type     | Required | Description                              |
| ------------ | -------- | -------- | ---------------------------------------- |
| `connection` | string   | yes      | Target connection name                   |
| `path`       | string   | yes      | Backup path or @repo/name               |
| `force`      | boolean  | yes      | Must be true for destructive restore     |
| `confirm`    | boolean  | yes      | Must be true for MCP destructive ops     |
| `database`   | string   | no       | Override target database name            |
| `tables`     | string[] | no       | Specific tables for partial restore      |
| `type`       | string   | no       | Explicit backup type override            |

## API

- `Siphon.Restore(ctx, *RestoreOptions) error`

### RestoreOptions

```go
type RestoreOptions struct {
    Connection string   // target connection name
    Path       string   // backup path or @repo/name
    Force      bool     // required for destructive restore
    Database   string   // override target database name
    Tables     []string // partial restore -- specific tables only
    Type       string   // explicit backup type override
    Surface    Surface  // which surface is calling (for policy)
    Confirm    bool     // MCP confirmation
}
```

## Engine Capability Matrix

| Engine     | Logical Restore | Physical Restore | File Restore |
| ---------- | --------------- | ---------------- | ------------ |
| MariaDB    | Yes             | Yes              | No           |
| PostgreSQL | Yes             | No               | No           |
| SQLite     | No              | No               | Yes          |

## Restore Flow

1. Resolve backup path (supports `@repo/name` syntax)
2. Resolve target connection from config
3. Check policy -- deny/readonly blocks, confirm required for MCP
4. Enforce `--force` flag
5. Read manifest from `backup.meta.yaml` (or use `--type` override)
6. Validate backup type against engine capabilities
7. Create adaptor, connect to target
8. Execute restore:
   - **Physical**: `mariabackup --prepare` then `--copy-back`
   - **Logical**: Decompress if needed, then pipe through database client
   - **File**: Copy backup file to destination path
9. `--database` overrides the target database for logical restores
10. `--tables` filters specific tables (logical only; rejected for physical)

## Policy

Restore operations respect the policy system:

- `deny` / `readonly` -- Blocks the restore entirely
- `confirm` -- Requires `confirm: true` on MCP surface
- `allow` -- Permits the restore (still requires `--force`)
- Per-connection `allow_restore: false` blocks restore for that connection

## Decompression

Compressed backups are automatically decompressed based on the manifest's
`compress` field. Supported algorithms:

| Algorithm | Extension | Notes                    |
| --------- | --------- | ------------------------ |
| `zstd`    | `.zst`    | Best ratio/speed balance |
| `gzip`    | `.gz`     | Widely available         |
| `none`    | (none)    | No compression           |
