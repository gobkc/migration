package dialect

import "fmt"

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
