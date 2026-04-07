# Transfer

## Overview

Transfer tables between databases using `CONN:DATABASE` colon syntax. Tables
are transferred in foreign key dependency order to maintain referential
integrity.

## Colon Syntax

Positional arguments use `CONN:DATABASE` format. If the database part is
omitted, the connection's default database is used.

```
source_conn:source_db  ->  target_conn:target_db
prod:myapp             ->  local:dev
staging                ->  local         (uses default databases)
```

## Table Selection

Tables are selected using a four-level precedence:

1. `--all-tables` overrides everything and transfers all tables
2. `--group` + `--exclude-group` + `--tables` union
3. `--tables` alone
4. Default: all tables from the source database

### Table Groups

Named table groups are defined in the config file. Groups support two modes:

**Simple list:**
```yaml
table_groups:
  core:
    tables:
      - users
      - orders
      - products
```

**Regex pattern:**
```yaml
table_groups:
  logs:
    pattern: "^log_"
    exclude:
      - log_internal
```

## Foreign Key Ordering

Tables are automatically sorted in topological order based on foreign key
dependencies. Dependencies are transferred first, so referential integrity
is maintained throughout the transfer. If a cycle is detected in the FK
graph, the transfer fails with an error.

## On-Conflict Strategies

The `--on-conflict` flag controls how existing rows in the target are handled:

| Strategy    | Behavior                                          |
|-------------|---------------------------------------------------|
| `skip`      | Skip rows that already exist (default)            |
| `overwrite` | Replace existing rows with source data            |
| `merge`     | Merge source data into existing rows              |

## CLI

```bash
siphon transfer SOURCE TARGET [flags]

# Transfer all tables
siphon transfer prod:myapp local:dev

# Transfer specific tables
siphon transfer prod:myapp local:dev --tables users,orders

# Transfer using table groups
siphon transfer prod:myapp local:dev --group core --exclude-group logs

# Transfer all tables with overwrite strategy
siphon transfer prod:myapp local:dev --all-tables --on-conflict overwrite
```

### Flags

| Flag               | Description                                       |
|--------------------|---------------------------------------------------|
| `--tables`         | Comma-separated table list                        |
| `--group`          | Named table group (repeatable)                    |
| `--exclude-group`  | Named table group to exclude (repeatable)         |
| `--all-tables`     | Transfer all tables                               |
| `--on-conflict`    | Conflict strategy: skip, overwrite, merge         |
| `--confirm`        | Confirm destructive operation                     |

## MCP Tools

- `transfer` - Transfer tables between databases

### Input Fields

| Field            | Type       | Description                               |
|------------------|------------|-------------------------------------------|
| `source`         | string     | Source CONN:DATABASE                       |
| `target`         | string     | Target CONN:DATABASE                       |
| `tables`         | []string   | Explicit table list                        |
| `groups`         | []string   | Named table groups to include              |
| `exclude_groups` | []string   | Named table groups to exclude              |
| `all_tables`     | bool       | Transfer all tables                        |
| `on_conflict`    | string     | Conflict strategy: skip, overwrite, merge  |
| `confirm`        | bool       | Confirm destructive operation              |

## API

- `Siphon.Transfer(ctx, &TransferOptions{})`

## Policy

Transfer operations respect connection policies:

- Source connection requires at least read access
- Target connection requires write access (readonly policy blocks transfer)
- MCP surface may require explicit `confirm: true` when policy is set to `confirm`
- Per-connection `allow_transfer: false` blocks the operation
