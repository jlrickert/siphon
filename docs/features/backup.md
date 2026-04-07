# Backup

## Overview

Create, list, inspect, verify, and label database backups.

## CLI

```bash
siphon backup create CONN:DB [--type logical|physical|file] [--repo NAME] [--label LABEL]
siphon backup info BACKUP_ID
siphon backup list [--connection NAME] [--database NAME] [--repo NAME]
siphon backup verify BACKUP_ID
siphon backup label BACKUP_ID LABEL
```

## MCP Tools

- `backup_create`
- `backup_info`
- `backup_list`
- `backup_verify`
- `backup_label`

## API

- `Siphon.Backup()`
- `Siphon.BackupInfo()`
- `Siphon.ListBackups()`
- `Siphon.VerifyBackup()`
- `Siphon.LabelBackup()`
