package siphon

import "errors"

var (
	// ErrNotImplemented indicates the operation is stubbed and not yet implemented.
	ErrNotImplemented = errors.New("not implemented")

	// ErrNotConnected indicates no active connection exists.
	ErrNotConnected = errors.New("not connected")

	// ErrConnectionNotFound indicates the named connection was not found.
	ErrConnectionNotFound = errors.New("connection not found")

	// ErrConnectionExists indicates the named connection already exists.
	ErrConnectionExists = errors.New("connection already exists")

	// ErrBackupFailed indicates that a backup operation failed.
	ErrBackupFailed = errors.New("backup failed")

	// ErrRestoreFailed indicates that a restore operation failed.
	ErrRestoreFailed = errors.New("restore failed")

	// ErrForceRequired indicates --force is required for a destructive operation.
	ErrForceRequired = errors.New("--force required for destructive operation")

	// ErrUnsupportedCapability indicates the engine does not support the
	// requested capability.
	ErrUnsupportedCapability = errors.New("unsupported capability for this engine")

	// ErrInvalidEngine indicates an unrecognized engine type.
	ErrInvalidEngine = errors.New("invalid engine type")

	// ErrManifestNotFound indicates the backup manifest was not found.
	ErrManifestNotFound = errors.New("backup manifest not found")

	// ErrBackupTypeMismatch indicates a backup type mismatch during restore.
	ErrBackupTypeMismatch = errors.New("backup type mismatch")

	// ErrUnsupportedRestoreTarget indicates the restore target is not supported.
	ErrUnsupportedRestoreTarget = errors.New("unsupported restore target")

	// ErrOperationDenied indicates the operation was denied by policy.
	ErrOperationDenied = errors.New("operation denied by policy")

	// ErrConfirmationRequired indicates a destructive MCP operation needs
	// confirm: true.
	ErrConfirmationRequired = errors.New("confirmation required")

	// ErrTransferFailed indicates that a transfer operation failed.
	ErrTransferFailed = errors.New("transfer failed")

	// ErrCycleDetected indicates a dependency cycle was found during
	// topological sort.
	ErrCycleDetected = errors.New("cycle detected")

	// ErrScheduleExists indicates the named schedule already exists.
	ErrScheduleExists = errors.New("schedule already exists")

	// ErrScheduleNotFound indicates the named schedule was not found.
	ErrScheduleNotFound = errors.New("schedule not found")
)
