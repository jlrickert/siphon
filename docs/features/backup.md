# Backup

## Overview

Create, list, inspect, verify, and label database backups. Supports three
backup types depending on the database engine:

- **logical** — SQL dump via `mariadb-dump` or `pg_dump` (MariaDB, PostgreSQL)
- **physical** — Binary backup via `mariabackup` (MariaDB only)
- **file** — File-level copy via `VACUUM INTO` or file copy (SQLite)

Each backup produces a `backup.meta.yaml` manifest containing metadata,
checksums, and file inventory for verification.

## CLI

### Create a backup

```bash
# Logical backup (default for MariaDB/PostgreSQL)
siphon backup create dev:testdb

# Physical backup (MariaDB only)
siphon backup create dev --type physical

# File backup (SQLite)
siphon backup create local --type file

# With compression
siphon backup create dev:testdb --compress gzip

# With custom name
siphon backup create dev:testdb --name my-backup-2026-04-07

# Into a named repository
siphon backup create dev:testdb --repo nightly

# With label and message
siphon backup create dev:testdb --label "pre-migration" -m "Before schema change"
```

### Inspect backup metadata

```bash
siphon backup info /path/to/backup
siphon backup info @nightly/my-backup-2026-04-07
siphon backup info /path/to/backup --format json
siphon backup info /path/to/backup --format yaml
```

### List backups

```bash
# List all backups in a repository
siphon backup list --repo nightly

# Filter by connection
siphon backup list --repo nightly --connection dev

# Filter by database
siphon backup list --repo nightly --database testdb

# Sort by size or engine
siphon backup list --repo nightly --sort size

# Limit results
siphon backup list --repo nightly --limit 5

# Output as JSON or YAML
siphon backup list --repo nightly --format json
```

### Verify backup integrity

```bash
siphon backup verify /path/to/backup
siphon backup verify @nightly/my-backup-2026-04-07
```

Verification checks that all files listed in the manifest exist and have
matching SHA-256 checksums.

### Label a backup

```bash
siphon backup label /path/to/backup "production"
siphon backup label @nightly/my-backup-2026-04-07 "pre-migration"
```

## MCP Tools

- `backup_create` — Create a database backup
- `backup_info` — Show backup metadata
- `backup_list` — List available backups
- `backup_verify` — Verify backup integrity
- `backup_label` — Set or update a backup label

## API

- `Siphon.Backup(ctx, *BackupOptions) (*BackupDescriptor, error)`
- `Siphon.BackupInfo(ctx, *BackupInfoOptions) (*BackupDescriptor, error)`
- `Siphon.ListBackups(ctx, *ListBackupsOptions) ([]BackupDescriptor, error)`
- `Siphon.VerifyBackup(ctx, *VerifyBackupOptions) error`
- `Siphon.LabelBackup(ctx, *LabelBackupOptions) error`

## Backup Name Templates

Backup directories are named using Go templates. The default template is:

```
{{.Connection}}-{{.Date}}-{{.Time}}
```

Available variables:

| Variable       | Example            |
| -------------- | ------------------ |
| `.Connection`  | `dev`              |
| `.Database`    | `testdb`           |
| `.Date`        | `2026-04-07`       |
| `.Time`        | `14-30-15`         |
| `.Engine`      | `mariadb`          |
| `.Type`        | `logical`          |
| `.Timestamp`   | (Go time.Time)     |

Configure globally via `backup_name_format` in config or per-backup via
`--name`.

## Compression

Backup data can be compressed with `--compress`:

| Algorithm | Extension | Notes                    |
| --------- | --------- | ------------------------ |
| `zstd`    | `.zst`    | Best ratio/speed balance |
| `gzip`    | `.gz`     | Widely available         |
| `none`    | (none)    | No compression (default) |

## Manifest Format

Each backup directory contains a `backup.meta.yaml` manifest:

```yaml
version: "1"
engine: mariadb
type: logical
connection: dev
database: testdb
timestamp: 2026-04-07T14:30:15Z
checksum: "abc123..."
compress: gzip
label: nightly
message: "Scheduled backup"
files:
  - path: dump.sql.gz
    size: 10240
    checksum: "def456..."
size: 10240
```

## Engine Capabilities

| Engine     | Logical | Physical | File |
| ---------- | ------- | -------- | ---- |
| MariaDB    | Yes     | Yes      | No   |
| PostgreSQL | Yes     | No       | No   |
| SQLite     | No      | No       | Yes  |

## @repo Syntax

Backup paths support `@repo` syntax for referencing configured repositories:

```bash
# References the "nightly" repo path from config
siphon backup info @nightly/my-backup

# Equivalent to the repo's configured path
siphon backup info /var/backups/nightly/my-backup
```

Configure repos in `~/.config/siphon/config.yaml`:

```yaml
repos:
  nightly:
    path: /var/backups/nightly
```
