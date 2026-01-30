package dialect

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
UPDATE migrations SET dirty=false, change_log=?,execution_time_s=? WHERE version=?
`
}
