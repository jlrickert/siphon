package engine

import "errors"

var (
	// ErrConnectionFailed indicates that a connection attempt failed.
	ErrConnectionFailed = errors.New("connection failed")

	// ErrQueryFailed indicates that a query execution failed.
	ErrQueryFailed = errors.New("query failed")

	// ErrDumpFailed indicates that a logical dump operation failed.
	ErrDumpFailed = errors.New("dump failed")

	// ErrLoadFailed indicates that a dump load operation failed.
	ErrLoadFailed = errors.New("load failed")

	// ErrPhysicalBackupFailed indicates that a physical backup operation failed.
	ErrPhysicalBackupFailed = errors.New("physical backup failed")

	// ErrPrepareFailed indicates that a backup prepare operation failed.
	ErrPrepareFailed = errors.New("prepare failed")

	// ErrCopyBackFailed indicates that a copy-back operation failed.
	ErrCopyBackFailed = errors.New("copy-back failed")

	// ErrTransferFailed indicates that a table transfer operation failed.
	ErrTransferFailed = errors.New("transfer failed")
)
