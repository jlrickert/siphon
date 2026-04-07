# Backup Schedules

## Overview

Create, list, and remove automated backup schedules. Supports launchd (macOS)
and cron (Linux) backends with optional retention policies.

See [scheduling.md](scheduling.md) for the full feature reference.

## CLI

```bash
# Create a daily backup schedule at 2 AM
siphon schedule create nightly --connection dev --time 02:00 --launchd

# Create with retention policy
siphon schedule create nightly --connection dev --time 02:00 --launchd --keep-count 7 --keep-age 30d

# List all schedules
siphon schedule list
siphon schedule list --format json

# Remove a schedule
siphon schedule remove nightly
```

## MCP Tools

- `schedule_create` -- Create a backup schedule
- `schedule_list` -- List backup schedules
- `schedule_remove` -- Remove a backup schedule

## API

- `Siphon.CreateSchedule(ctx, *CreateScheduleOptions) error`
- `Siphon.ListSchedules(ctx, *ListSchedulesOptions) ([]ScheduleInfo, error)`
- `Siphon.RemoveSchedule(ctx, *RemoveScheduleOptions) error`

## Retention

Schedules support a retention policy with two dimensions:

- `keep_count` -- Keep the N most recent backups
- `keep_age` -- Keep backups newer than a duration (e.g., `720h`, `30d`)

Both can be combined; a backup is pruned if it fails either check.
