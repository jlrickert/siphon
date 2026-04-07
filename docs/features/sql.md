# SQL Execution

## Overview

Execute raw SQL queries against database connections. Supports multiple output
formats, file-based input, piped input, and interactive mode.

## CLI

```bash
# Direct query
siphon sql mydb "SELECT * FROM users"

# File input
siphon sql mydb -f schema.sql

# Piped input
echo "SELECT 1" | siphon sql mydb

# Interactive mode (when no query and TTY)
siphon sql mydb

# Override database
siphon sql mydb "SHOW TABLES" --database other_db

# Output format
siphon sql mydb "SELECT * FROM users" --format csv
siphon sql mydb "SELECT * FROM users" --format json
```

### Flags

| Flag         | Description                           | Default |
|--------------|---------------------------------------|---------|
| `-f, --file` | SQL file to execute                   |         |
| `--format`   | Output format: table, csv, json       | table   |
| `--database`  | Override database                     |         |
| `--confirm`   | Confirm destructive operation         | false   |

### Interactive Mode

When no query or file is provided and stdin is a TTY, siphon enters
interactive mode. Type SQL queries line by line. Use `\q`, `exit`, or
`quit` to exit. The prompt (`siphon> `) is printed to stderr.

## Output Formats

### Table (default)

Aligned columns using tabwriter:

```
id  name   email
--  ----   -----
1   Alice  alice@example.com
2   Bob    bob@example.com
```

### CSV

Standard CSV with headers:

```
id,name,email
1,Alice,alice@example.com
2,Bob,bob@example.com
```

### JSON

Array of objects, each row as `{column: value}`:

```json
[
  {"id": "1", "name": "Alice", "email": "alice@example.com"},
  {"id": "2", "name": "Bob", "email": "bob@example.com"}
]
```

## MCP Tools

### sql_execute

Execute a SQL query against a database connection. Non-interactive only.

| Parameter    | Type   | Required | Description                    |
|--------------|--------|----------|--------------------------------|
| connection   | string | no       | Target connection name         |
| database     | string | no       | Override database              |
| query        | string | yes      | SQL query to execute           |
| format       | string | no       | Output format (table/csv/json) |
| confirm      | bool   | no       | Confirm destructive operation  |

## API

- `Siphon.ExecuteSQL(ctx, opts)` -- core service method

## Policy Behavior

SQL execution respects the policy configuration:

- **allow**: All queries permitted
- **readonly**: Only read queries (SELECT, SHOW, DESCRIBE, EXPLAIN) are
  permitted. Write queries (INSERT, UPDATE, DELETE, DROP, ALTER, CREATE,
  TRUNCATE) are denied.
- **confirm**: Write queries on the MCP surface require `confirm: true`.
  Read queries are always allowed.
- **deny**: All queries are denied.

Per-connection `allow_sql: false` disables SQL execution entirely for that
connection.

## Write Detection

Queries are classified as read or write based on the first meaningful SQL
keyword after stripping comments and whitespace. This is a best-effort
keyword classifier, not a full SQL parser.
