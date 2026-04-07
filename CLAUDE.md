# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## Overview

**siphon** is a Go database management tool exposing backup, restore, transfer,
SQL execution, and connection management through three isomorphic interfaces:
CLI, MCP server, and Go API. All three share the same operation surface. Targets
MariaDB (primary), PostgreSQL (secondary), SQLite (tertiary/embedded).

Architecture flow: `cli/mcp <-> api <-> adaptor`

- **CLI** — Cobra commands mapping 1:1 to API methods
- **MCP** — JSON-RPC tools mapping 1:1 to API methods (via `siphon mcp`)
- **API** — Service layer containing all business logic
- **Adaptor** — Interface-based backend abstraction per database engine

## Build & Development Commands

```bash
# Build
go build ./cmd/siphon

# Test
go test ./...                              # all tests
go test ./pkg/siphon/... -v                # single package, verbose
go test -race ./pkg/siphon/...             # with race detector

# Install (requires go-task)
task install                               # install siphon + completions
task test                                  # cached test run

# Lint
go vet ./...
```

## Architecture

### Package Map

- **`cmd/siphon/`** — Binary entrypoint.
- **`pkg/siphon/`** — Service layer: API methods, `*Options` structs, config
  resolution, path service.
- **`pkg/cli/`** — Cobra commands bridging CLI flags to service methods.
- **`pkg/mcp/`** — MCP server exposing service methods over JSON-RPC.
- **`pkg/engine/`** — Engine adaptor interfaces and implementations (MariaDB,
  PostgreSQL, SQLite, Mock).
- **`pkg/parity/`** — CLI-MCP-API parity tests.

### Key Types and Flow

**Siphon** (`pkg/siphon/siphon.go`) is the central service struct. All
operations flow through Siphon via two parallel entry points:

```
CLI command   → pkg/cli (Cobra)    → pkg/siphon.Siphon → pkg/engine.Adaptor
MCP tool call → pkg/mcp (JSON-RPC) → pkg/siphon.Siphon → pkg/engine.Adaptor
```

Both paths converge at `pkg/siphon.Siphon`, sharing the same method and
`*Options` struct for each feature. The CLI path uses Cobra flags resolved into
options and writes results to stdout. The MCP path uses input structs annotated
via `jsonschema` tags and returns `CallToolResult` values.

### Engine Adaptor Model

Engine adaptors use the capability interface coproduct pattern: a core `Adaptor`
interface with optional capability interfaces checked at runtime via type
assertion.

**Core `Adaptor` interface:** Connect, Ping, Close, ListDatabases, Engine.

**Optional capabilities:**

- `LogicalBackupAdaptor` — Dump, LoadDump
- `PhysicalBackupAdaptor` — PhysicalBackup, Prepare, CopyBack
- `FileBackupAdaptor` — CopyBackup, CopyRestore
- `TransferAdaptor` — TransferTables
- `QueryAdaptor` — Execute, Query

**SQLite degenerate case:** SQLite is an embedded engine — the "connection" is a
file path, the database concept is implicit (the file IS the database). The
SQLite adaptor implements core Adaptor + FileBackupAdaptor + TransferAdaptor
only. No PhysicalBackupAdaptor or LogicalBackupAdaptor.

### Key Types

- **`DatabaseTarget`** — (connection, database) product type. The fundamental
  data unit for most operations.
- **`ConnectionConfig`** — Host, port, credentials, engine type.
- **`Engine`** — Enum: MariaDB, PostgreSQL, SQLite.
- **`BackupDescriptor`** — Backup metadata (engine, type, timestamp, checksum,
  label, source connection, file paths).
- **`Config`** — Resolved configuration with connections, repos, policies, table
  groups.

### Config Hierarchy

Configuration is resolved via a five-tier cascade using `cfgcascade` from
cli-toolkit (most specific wins):

| Rank | Source | Discovery |
| ---- | ------ | --------- |
| 5 | CLI flags | Cobra `cmd.Flags().Changed()` |
| 4 | Project config | `.siphon/config.yaml` |
| 3 | User config | `~/.config/siphon/config.yaml` |
| 2 | Env vars (`SIPHON_*`) | `rt.Env().Get()` prefix scan |
| 1 | Defaults | Hardcoded in code |

Supported env vars: `SIPHON_DEFAULT_CONNECTION`, `SIPHON_LOG_FILE`,
`SIPHON_LOG_LEVEL`.

Use `siphon config --explain` for per-field provenance showing which tier
provided each value.

**Merge semantics:** Scalars use pointer-based override (`*string`, `*bool`);
maps (connections, repos, policies) use recursive deep merge; schedules use
last-wins per key; lists (`connectionMap`) use append with deduplication.

### Dependency: cli-toolkit

The `github.com/jlrickert/cli-toolkit` module provides `toolkit.Runtime` — the
explicit dependency container carrying filesystem, env, clock, logger, hasher,
stream, and process identity. All I/O in siphon flows through Runtime, enabling
sandboxed test environments.

### Runtime Abstraction Rule

All I/O in `pkg/siphon`, `pkg/cli`, `pkg/mcp`, and `pkg/engine` must go through
`toolkit.Runtime`. Direct stdlib calls bypass the sandboxed test environment and
break test isolation. Specifically:

- **File I/O**: Use `rt.ReadFile` / `rt.WriteFile` — never `os.ReadFile` /
  `os.WriteFile`.
- **Streams**: Use `rt.Stream().Out` / `rt.Stream().Err` — never `os.Stdout` /
  `os.Stderr` directly.
- **Clock**: Use `rt.Clock().Now()` — never `time.Now()`.
- **Commands**: Use `exec.CommandContext(ctx, ...)` — never bare
  `exec.Command(...)`.

## Testing

- **Sandbox pattern**: Tests use `sandbox.NewSandbox(t, ...)` from cli-toolkit,
  which creates a jailed temp directory with a test runtime (mock clock, MD5
  hasher, test logger).
- **Mock adaptor**: `pkg/engine/mock.go` implements all capability interfaces
  for unit tests without real database connections.
- **Parity tests**: `pkg/parity/` contains reflection-based coverage tests that
  verify every exported Siphon API method has both a CLI command and MCP tool
  registered, plus table-driven comparison tests that verify CLI and MCP produce
  equivalent results for the same input.
- **Race detection**: Run `go test -race ./...` to verify concurrent safety.
- **Testify**: Uses `github.com/stretchr/testify/require` for assertions.

## Error Handling

- Sentinel errors in `pkg/siphon/errors.go`: `ErrNotImplemented`,
  `ErrNotConnected`, `ErrConnectionNotFound`, `ErrConnectionExists`,
  `ErrBackupFailed`, `ErrRestoreFailed`, `ErrForceRequired`,
  `ErrUnsupportedCapability`, `ErrInvalidEngine`, `ErrManifestNotFound`,
  `ErrBackupTypeMismatch`, `ErrUnsupportedRestoreTarget`,
  `ErrOperationDenied`, `ErrConfirmationRequired`.
- Engine-level errors in `pkg/engine/errors.go`.
- Check with `errors.Is()` for sentinels.

## Feature Organization

Features are vertical slices: each capability cuts through every layer from the
API down to tests. CLI, MCP, and API are peer surfaces — all three must expose
the same features at parity.

### Four-surface parity rule

Every feature must exist across four surfaces: CLI command, MCP tool, API
method, and documentation. **A missing surface is a bug.** Parity tests enforce
CLI-MCP-API alignment; doc coverage is enforced by convention.

### Feature anatomy

| Layer | Location pattern | Purpose |
| ----- | ---------------- | ------- |
| **API** | `pkg/siphon/siphon_*.go` | Business logic method + `*Options` struct |
| **CLI command** | `pkg/cli/cmd_*.go` | Cobra command wiring flags to API method |
| **Completions** | `pkg/cli/cmd_*.go` | `ValidArgsFunction` and custom completers for flags/args |
| **MCP tool** | `pkg/mcp/tools_*.go` | JSON-RPC tool with input struct + `jsonschema` tags |
| **Tests** | `*_test.go` in each pkg | Unit, parity, completion tests |
| **Documentation** | `docs/features/` | User-facing docs for the capability |

### Checklist

When adding or modifying a feature, update each of these:

1. **API method** (`pkg/siphon/`) — business logic + Options struct + tests
2. **CLI command** (`pkg/cli/cmd_*.go`) — Cobra command wiring flags to API
3. **Shell completions** — `ValidArgsFunction` + `RegisterFlagCompletionFunc`
   for all fixed-value flags
4. **MCP tool** (`pkg/mcp/tools_*.go`) — JSON-RPC tool with `jsonschema` tags
5. **Documentation** (`docs/features/`) — user-facing docs
6. **Tests** — unit, CLI integration, MCP tool, parity test case

**Configuration changes:** Any change to configuration structure must also
update `schemas/siphon-config.json`. This schema is referenced by editors for
validation and completion hints. A config field added without a schema update
will lack editor support and validation.

## Gotchas

- **Commit conventions**: Conventional commits (`feat:`, `fix:`, `refactor:`),
  summaries ≤72 chars.
- **Cobra skips PersistentPostRunE when RunE returns an error.** Any cleanup
  or logging that must run on both success and failure paths cannot rely on
  PersistentPostRunE. Handle cleanup in the top-level run function.
- **SQLite has no client-server model.** Connection is a file path, database
  concept is implicit.
- **Physical backups (mariabackup) are server-scoped**, not database-scoped.
  This is the documented exception to the one-backup-one-database rule.
- **`--force` flag required for destructive operations** (restore, overwrite).
- **Colon syntax `CONN:DATABASE` for transfer.** Omitting the colon uses the
  connection's default database.
- **`@repo` syntax for backup repositories.** `@nightly/mydb-2026-04-02`
  resolves to a configured repo path.
- **Connection policies are enforced in the service layer**, not CLI/MCP
  translation layers. This ensures consistent behavior across all surfaces.
- **MCP destructive operations use two-step confirmation** (`confirm: true`
  parameter).
- **Use `*string` / `*bool` for config fields** to distinguish "explicitly set
  to zero" from "absent."
