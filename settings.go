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
	db        *sql.DB
	dialect   dialect.Dialect
	source    source.Source
	variables map[string]any
}

func Settings(callbacks ...func(options *Options) *Migrator) {
	once.Do(func() {
		var opts = &Options{variables: make(map[string]any)}
		for _, callback := range callbacks {
			migrator = callback(opts)
			migrator.db = opts.db
			migrator.dialect = opts.dialect
			migrator.source = opts.source
			maps.Copy(migrator.variables, opts.variables)
		}
	})
}
