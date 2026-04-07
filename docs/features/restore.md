# Restore

## Overview

Restore a database from a backup. Requires `--force` for destructive operations.

## CLI

```bash
siphon restore BACKUP_ID [--connection NAME] [--database NAME] [--force]
```

## MCP Tools

- `restore` (requires `confirm: true` for destructive operations)

## API

- `Siphon.Restore()`
