package migration

import (
	"database/sql"
	"sync"

	"github.com/gobkc/migration/dialect"
	"github.com/gobkc/migration/source"
)

var once sync.Once
var migrator = &Migrator{}

type Options struct {
	Db      *sql.DB
	Dialect dialect.Dialect
	Source  source.Source
}

func Settings(callbacks ...func(options *Options)) *Migrator {
	once.Do(func() {
		var opts = &Options{}
		for _, callback := range callbacks {
			callback(opts)
			migrator.db = opts.Db
			migrator.dialect = opts.Dialect
			migrator.source = opts.Source
		}
	})
	return migrator
}
