package migration

import (
	"database/sql"
)

func tryLock(db *sql.DB) (bool, error) {

	_, err := db.Exec(`
        INSERT INTO migrations
        (version, change_log, checksum, applied_at, execution_time_s, dirty)
        VALUES (-1, 'locked', 'lock', NOW(), 0, false)
    `)

	if err != nil {
		return false, nil // duplicate = locked
	}

	return true, nil
}

func unlock(db *sql.DB) {
	db.Exec(`DELETE FROM migrations WHERE version = -1`)
}
