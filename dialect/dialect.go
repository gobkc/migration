package dialect

import (
	"context"
	"database/sql"
)

type Dialect interface {
	Name() string
	Placeholder(n int) string
	Now() string
	BoolType() string
	InsertMigrationSql() string
	UpdateMigrationSql() string
}

type UnlockFunc func(context.Context) error

type Locker interface {
	Lock(ctx context.Context, db *sql.DB) (UnlockFunc, error)
}

func closePinnedConn(conn *sql.Conn) {
	_ = conn.Raw(func(driverConn any) error {
		if closer, ok := driverConn.(interface{ Close() error }); ok {
			return closer.Close()
		}
		return nil
	})
	_ = conn.Close()
}
