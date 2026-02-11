package migration

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
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
		slog.Warn("query dirty record", slog.String("warn", err.Error()))
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

	hasBeyondBaseline, err := hasMigrationBeyondBaseline(m.db)
	if err != nil {
		return err
	}

	// ⭐⭐⭐⭐⭐ STEP 1 — holding（forever）
	for _, mig := range migrations {

		if mig.Direction != types.Holding {
			continue
		}

		if err := m.execStateless(ctx, mig); err != nil {
			return err
		}
	}

	// ⭐⭐⭐⭐⭐ STEP 2 — up / baseline
	for _, mig := range migrations {

		if mig.Direction != types.Up {
			continue
		}

		// baseline only execute once
		if mig.Version == types.BaseLineVersion && hasBeyondBaseline {
			slog.Info("baseline skipped: database already initialized")
			continue
		}

		if checksum, ok := applied[mig.Version]; ok {

			if mig.Version == types.BaseLineVersion {
				continue
			}

			if checksum != mig.Checksum {
				slog.Warn(ErrChecksumMismatch.Error(),
					slog.Int64("version", mig.Version),
					slog.String("exists", checksum),
					slog.String("new", mig.Checksum),
				)
			}

			continue
		}

		if err := m.apply(ctx, mig); err != nil {
			return err
		}
	}

	for _, mig := range migrations {

		if mig.Direction != types.Final {
			continue
		}

		if err := m.execStateless(ctx, mig); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) apply(ctx context.Context, mig types.Migration) error {

	start := time.Now()

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.Exec(m.dialect.InsertMigrationSql(),
		mig.Version,
		mig.ChangeLog,
		mig.Checksum,
	)

	if err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.ExecContext(ctx, mig.SQL); err != nil {
		tx.Rollback()
		return err
	}

	elapsed := time.Since(start).Seconds()

	_, err = tx.Exec(m.dialect.UpdateMigrationSql(),
		mig.ChangeLog,
		elapsed,
		mig.Version,
	)

	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (m *Migrator) execStateless(ctx context.Context, mig types.Migration) error {

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, mig.SQL); err != nil {
		tx.Rollback()
		return fmt.Errorf("migration %d (%s) failed: %w",
			mig.Version,
			mig.Direction,
			err,
		)
	}

	return tx.Commit()
}

func hasMigrationBeyondBaseline(db *sql.DB) (bool, error) {

	var exists bool

	err := db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM migrations WHERE version > 0
        )
    `).Scan(&exists)

	return exists, err
}
