# SQL Execution

## Overview

Execute raw SQL queries against database connections.

## CLI

```bash
siphon sql "SELECT * FROM users" [--connection NAME] [--database NAME] [--format table|json|csv]
```

## MCP Tools

- `sql_execute`

## API

- `Siphon.ExecuteSQL()`
