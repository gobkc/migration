package migration

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gobkc/migration/dialect"
)

func ensureTable(db *sql.DB, d dialect.Dialect) error {

	q := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS migrations (
    version BIGINT PRIMARY KEY,
    change_log text,
    checksum VARCHAR(64) NOT NULL,
    applied_at TIMESTAMP NOT NULL,
    execution_time_s double precision NOT NULL,
    dirty %s NOT NULL
);
`, d.BoolType())

	_, err := db.Exec(q)
	return err
}

func isDirty(db *sql.DB) (bool, error) {
	var dirty bool

	err := db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM migrations WHERE dirty = true
        )
    `).Scan(&dirty)

	return dirty, err
}

func getApplied(db *sql.DB) (map[int64]string, error) {

	rows, err := db.Query(`
        SELECT version, checksum
        FROM migrations
        WHERE version > -1
    `)
	if err != nil {
		if strings.Contains(err.Error(), "42703") {
			return make(map[int64]string), nil
		}
		return nil, err
	}
	defer rows.Close()

	m := make(map[int64]string)

	for rows.Next() {
		var v int64
		var c string
		rows.Scan(&v, &c)
		m[v] = c
	}

	return m, nil
}
