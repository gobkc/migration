package dialect

type Dialect interface {
	Name() string
	Placeholder(n int) string
	Now() string
	BoolType() string
	InsertMigrationSql() string
	UpdateMigrationSql() string
}
