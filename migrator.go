package migration

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gobkc/migration/dialect"
	"github.com/gobkc/migration/source"
	"github.com/gobkc/migration/types"
)

type Migrator struct {
	db      *sql.DB
	dialect dialect.Dialect
	source  source.Source
}

func New(db *sql.DB, d dialect.Dialect, s source.Source) *Migrator {
	return &Migrator{
		db:      db,
		dialect: d,
		source:  s,
	}
}

func (m *Migrator) Up(ctx context.Context) error {

	if err := ensureTable(m.db, m.dialect); err != nil {
		return err
	}

	dirty, err := isDirty(m.db)
	if err != nil {
		return err
	}

	if dirty {
		return ErrDirtyDatabase
	}

	locked, err := tryLock(m.db)
	if err != nil {
		return err
	}

	if !locked {
		return ErrLocked
	}

	defer unlock(m.db)

	migrations, err := m.source.Migrations()
	if err != nil {
		return err
	}

	applied, err := getApplied(m.db)
	if err != nil {
		return err
	}

	for _, mig := range migrations {

		if mig.Direction != "up" {
			continue
		}

		isBaseline := mig.Version == types.BaseLineVersion
		if c, ok := applied[mig.Version]; ok {
			if c != mig.Checksum {
				if isBaseline == false {
					return ErrChecksumMismatch
				}
			} else {
				continue
			}
		}

		if err := m.apply(ctx, mig, isBaseline); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) apply(ctx context.Context, mig types.Migration, skipInsert bool) error {

	start := time.Now()

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if skipInsert == false {
		_, err = tx.Exec(m.dialect.InsertMigrationSql(), mig.Version, mig.ChangeLog, mig.Checksum)
	}

	if err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.ExecContext(ctx, mig.SQL); err != nil {
		fmt.Println("invalid sql:\n", mig.SQL)
		tx.Rollback()
		return err
	}

	elapsed := time.Since(start).Seconds()

	_, err = tx.Exec(m.dialect.UpdateMigrationSql(), mig.ChangeLog, elapsed, mig.Version)

	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
