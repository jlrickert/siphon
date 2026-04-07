# Connection Management

## Overview

Manage database connections: list, add, remove, and test connectivity.

## CLI

```bash
siphon connections list
siphon connections add NAME --engine mariadb --host localhost --port 3306
siphon connections remove NAME
siphon connections test NAME
```

## MCP Tools

- `connections_list`
- `connections_add`
- `connections_remove`
- `connections_test`

## API

- `Siphon.ListConnections()`
- `Siphon.AddConnection()`
- `Siphon.RemoveConnection()`
- `Siphon.TestConnection()`
