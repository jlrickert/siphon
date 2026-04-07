# Backup Schedules

## Overview

Create, list, and remove automated backup schedules.

## CLI

```bash
siphon schedule create NAME --connection NAME --cron "0 2 * * *" [--repo NAME] [--type logical]
siphon schedule list [--connection NAME]
siphon schedule remove NAME
```

## MCP Tools

- `schedule_create`
- `schedule_list`
- `schedule_remove`

## API

- `Siphon.CreateSchedule()`
- `Siphon.ListSchedules()`
- `Siphon.RemoveSchedule()`
