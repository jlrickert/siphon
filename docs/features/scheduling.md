# Scheduling and Automation

## Overview

Create, list, and remove automated backup schedules. Siphon supports two
scheduling backends:

- **launchd** (macOS) -- generates a plist in `~/Library/LaunchAgents/`
- **cron** (Linux) -- generates a cron entry for manual installation

Each schedule is persisted in the user config under the `schedules` map, so
it survives across sessions and can be managed through both CLI and MCP.

## CLI

### Create a schedule

```bash
# Daily backup at 2 AM using launchd (default)
siphon schedule create nightly --connection dev --time 02:00 --launchd

# Weekly backup on Sunday at 3 AM
siphon schedule create weekly-dev --connection dev --time 03:00 --interval weekly --launchd

# Hourly backup using cron backend
siphon schedule create hourly-dev --connection dev --time 00:15 --interval hourly --cron

# With all options
siphon schedule create full \
  --connection dev \
  --repo nightly \
  --database testdb \
  --type logical \
  --compress zstd \
  --time 02:00 \
  --interval daily \
  --launchd \
  --keep-count 7 \
  --keep-age 30d \
  --message "Scheduled nightly backup"
```

### List schedules

```bash
siphon schedule list
siphon schedule list --format json
```

### Remove a schedule

```bash
siphon schedule remove nightly
siphon schedule rm nightly
```

## MCP Tools

- `schedule_create` -- Create a backup schedule
- `schedule_list` -- List backup schedules
- `schedule_remove` -- Remove a backup schedule

## API

- `Siphon.CreateSchedule(ctx, *CreateScheduleOptions) error`
- `Siphon.ListSchedules(ctx, *ListSchedulesOptions) ([]ScheduleInfo, error)`
- `Siphon.RemoveSchedule(ctx, *RemoveScheduleOptions) error`

## Configuration

Schedules are stored in the user config (`~/.config/siphon/config.yaml`):

```yaml
schedules:
  nightly:
    connection: dev
    repo: nightly
    backup_type: logical
    compress: zstd
    time: "02:00"
    interval: daily
    backend: launchd
    retention:
      keep_count: 7
      keep_age: 30d
    enabled: true
```

### Fields

| Field        | Description                              | Default    |
| ------------ | ---------------------------------------- | ---------- |
| `connection` | Connection to backup (required)          | --         |
| `repo`       | Backup repository                        | --         |
| `database`   | Override target database                 | --         |
| `backup_type`| Backup type (logical/physical/file)      | `logical`  |
| `compress`   | Compression (zstd/gzip/none)             | --         |
| `time`       | Schedule time in HH:MM format            | `02:00`    |
| `interval`   | Frequency: daily, hourly, weekly         | `daily`    |
| `backend`    | Scheduling backend: launchd or cron      | `launchd`  |
| `retention`  | Optional retention policy                | --         |
| `enabled`    | Whether the schedule is active           | `true`     |

## Backends

### launchd (macOS)

Creates a plist at `~/Library/LaunchAgents/com.siphon.backup.<name>.plist`.
The plist uses `StartCalendarInterval` for time-based scheduling.

Logs are written to `~/Library/Logs/siphon/<name>.{out,err}.log`.

Remove deletes the plist file.

### cron (Linux)

Generates a cron entry that can be installed in the user's crontab. The
generated entry follows the standard five-field cron format:

```
M H * * * /path/to/siphon backup create <connection> [flags]
```

The entry is stored in the schedule config for reference. Users install it
into their crontab manually or via `crontab -e`.

## Retention Policy

Schedules can include a retention policy that defines how many backups to
keep:

| Field        | Description                                    | Example |
| ------------ | ---------------------------------------------- | ------- |
| `keep_count` | Keep the N most recent backups                 | `7`     |
| `keep_age`   | Keep backups newer than this duration           | `30d`   |

Both fields can be combined. A backup is pruned if it fails either check
(union of both rules).

Duration strings support Go syntax (`720h`) and a convenience day suffix
(`30d`).

The `ApplyRetention` function returns a list of backups that would be pruned
without actually deleting them (dry-run capability).

## Intervals

| Interval | launchd behavior                    | cron behavior    |
| -------- | ----------------------------------- | ---------------- |
| `daily`  | Runs at HH:MM every day             | `M H * * *`      |
| `hourly` | Runs at :MM every hour              | `M * * * *`      |
| `weekly` | Runs at HH:MM every Sunday          | `M H * * 0`      |
