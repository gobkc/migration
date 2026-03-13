package migration

import (
	"database/sql"
	"maps"
	"sync"

	"github.com/gobkc/migration/dialect"
	"github.com/gobkc/migration/source"
)

var once sync.Once
var migrator = &Migrator{}

type Options struct {
	Db        *sql.DB
	Dialect   dialect.Dialect
	Source    source.Source
	Variables map[string]any
}

func Settings(callbacks ...func(options *Options) *Migrator) {
	once.Do(func() {
		var opts = &Options{Variables: make(map[string]any)}
		for _, callback := range callbacks {
			migrator = callback(opts)
			migrator.db = opts.Db
			migrator.dialect = opts.Dialect
			migrator.source = opts.Source
			maps.Copy(migrator.variables, opts.Variables)
		}
	})
}
