# Configuration

## Overview

View resolved configuration with optional per-field provenance.

## Config Hierarchy

| Rank | Source          | Discovery                          |
| ---- | --------------- | ---------------------------------- |
| 5    | CLI flags       | `cmd.Flags().Changed()`            |
| 4    | Project config  | `.siphon/config.yaml`              |
| 3    | User config     | `~/.config/siphon/config.yaml`     |
| 2    | Env vars        | `SIPHON_*`                         |
| 1    | Defaults        | Hardcoded                          |

## CLI

```bash
siphon config
siphon config --explain
```

## MCP Tools

- `config`

## API

- `Siphon.Config()`
