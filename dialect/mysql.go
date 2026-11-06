package dialect

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type MySQL struct{}

func (MySQL) Name() string { return "mysql" }

func (MySQL) Placeholder(n int) string { return "?" }

func (MySQL) Now() string { return "NOW()" }

func (MySQL) BoolType() string { return "TINYINT(1)" }

func (MySQL) InsertMigrationSql() string {
	return `
INSERT INTO migrations (version, change_log, checksum, applied_at, execution_time_s, dirty) VALUES (?, ?, ?, NOW(), 0, true)
`
}

func (MySQL) UpdateMigrationSql() string {
	return `
UPDATE migrations SET dirty=false, change_log=?, execution_time_s=? WHERE version=?
`
}

func (MySQL) Lock(ctx context.Context, db *sql.DB) (UnlockFunc, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire mysql migration lock connection: %w", err)
	}

	var database sql.NullString
	if err := conn.QueryRowContext(ctx, `SELECT DATABASE()`).Scan(&database); err != nil {
		closePinnedConn(conn)
		return nil, fmt.Errorf("resolve mysql migration lock database: %w", err)
	}
	name := mysqlLockName(database.String)

	for {
		var acquired sql.NullInt64
		if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK(?, 1)`, name).Scan(&acquired); err != nil {
			closePinnedConn(conn)
			return nil, fmt.Errorf("acquire mysql migration lock: %w", err)
		}
		if acquired.Valid && acquired.Int64 == 1 {
			break
		}
		select {
		case <-ctx.Done():
			closePinnedConn(conn)
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}

	return func(unlockCtx context.Context) error {
		var released sql.NullInt64
		if err := conn.QueryRowContext(unlockCtx, `SELECT RELEASE_LOCK(?)`, name).Scan(&released); err != nil {
			closePinnedConn(conn)
			return fmt.Errorf("release mysql migration lock: %w", err)
		}
		if !released.Valid || released.Int64 != 1 {
			closePinnedConn(conn)
			return fmt.Errorf("release mysql migration lock %q: not owned", name)
		}
		return conn.Close()
	}, nil
}

func mysqlLockName(database string) string {
	sum := sha1.Sum([]byte(database))
	return `gobkc_migration:` + hex.EncodeToString(sum[:])
}
