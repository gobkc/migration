package migration

import "errors"

var (
	ErrLocked           = errors.New("migration locked")
	ErrLockUnsupported  = errors.New("dialect does not support migration locking")
	ErrDirtyDatabase    = errors.New("database is dirty, manual repair required")
	ErrChecksumMismatch = errors.New("migration checksum mismatch")
)
