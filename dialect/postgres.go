package dialect

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	postgresLockNamespace = int32(0x67626b63)
	postgresLockSQL       = `SELECT pg_advisory_lock($1, hashtext(current_database()))`
	postgresUnlockSQL     = `SELECT pg_advisory_unlock($1, hashtext(current_database()))`
)

type Postgres struct{}

func (Postgres) Name() string { return "postgres" }

func (Postgres) Placeholder(n int) string {
	return fmt.Sprintf("$%d", n)
}

func (Postgres) Now() string { return "NOW()" }

func (Postgres) BoolType() string { return "BOOLEAN" }

func (Postgres) InsertMigrationSql() string {
	return `
INSERT INTO migrations (version, change_log, checksum, applied_at, execution_time_s, dirty) VALUES ($1, $2, $3, NOW(), 0, true)
`
}

func (Postgres) UpdateMigrationSql() string {
	return `
UPDATE migrations SET dirty=false, change_log=$1, execution_time_s=$2 WHERE version=$3
`
}

func (Postgres) Lock(ctx context.Context, db *sql.DB) (UnlockFunc, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire postgres migration lock connection: %w", err)
	}
	if _, err := conn.ExecContext(ctx, postgresLockSQL, postgresLockNamespace); err != nil {
		closePinnedConn(conn)
		return nil, fmt.Errorf("acquire postgres migration lock: %w", err)
	}
	return func(unlockCtx context.Context) error {
		if _, err := conn.ExecContext(unlockCtx, postgresUnlockSQL, postgresLockNamespace); err != nil {
			closePinnedConn(conn)
			return fmt.Errorf("release postgres migration lock: %w", err)
		}
		return conn.Close()
	}, nil
}
