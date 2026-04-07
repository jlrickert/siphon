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
# Show resolved config as YAML
siphon config

# Show config with per-field provenance annotations
siphon config --explain
```

The `--explain` flag shows which configuration tier provided each value,
helping debug config cascade resolution.

## MCP Tools

- `config` — Show resolved configuration (params: explain)

## API

- `Siphon.Config(ctx, *ConfigOptions) (*ResolvedConfig, error)`

## Merge Semantics

- **Scalars**: Pointer-based override (`*string`, `*bool`). Higher-rank
  tiers override lower-rank values.
- **Maps** (connections, repos, policies): Recursive deep merge. Keys from
  higher-rank tiers override same-name keys from lower-rank tiers.
- **Lists** (`connectionMap`): Append with deduplication by connection name.

## Environment Variables

| Variable                    | Config field         |
| --------------------------- | -------------------- |
| `SIPHON_DEFAULT_CONNECTION` | `default_connection` |
| `SIPHON_LOG_FILE`           | `log_file`           |
| `SIPHON_LOG_LEVEL`          | `log_level`          |
