package migration

import "errors"

var (
	ErrLocked           = errors.New("migration locked")
	ErrDirtyDatabase    = errors.New("database is dirty, manual repair required")
	ErrChecksumMismatch = errors.New("migration checksum mismatch")
)
