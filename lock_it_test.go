package migration_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/gobkc/migration"
	"github.com/gobkc/migration/dialect"
	"github.com/gobkc/migration/source"
	"github.com/gobkc/migration/types"
)

type staticSource struct {
	migrations []types.Migration
}

func (s staticSource) Migrations() ([]types.Migration, error) {
	return s.migrations, nil
}

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("MIGRATION_TEST_DSN not set")
	}
	return dsn
}

func openDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func runUp(t *testing.T, dsn string, src source.Source) error {
	t.Helper()
	db := openDB(t, dsn)
	m := migration.New(db, dialect.Postgres{}, src)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return m.Up(ctx)
}

const legacyHoldingSQL = `
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'migrations' AND column_name = 'last_migrate'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'migrations' AND column_name = 'applied_at'
    ) THEN
        ALTER TABLE migrations RENAME COLUMN last_migrate TO applied_at;
    END IF;
END $$;
ALTER TABLE migrations ADD COLUMN IF NOT EXISTS checksum VARCHAR(64) DEFAULT '';
ALTER TABLE migrations ADD COLUMN IF NOT EXISTS execution_time_s double precision DEFAULT 0;
ALTER TABLE migrations ADD COLUMN IF NOT EXISTS dirty boolean DEFAULT false;
ALTER TABLE migrations ALTER COLUMN checksum SET NOT NULL;
ALTER TABLE migrations ALTER COLUMN execution_time_s SET NOT NULL;
ALTER TABLE migrations ALTER COLUMN dirty SET NOT NULL;
CREATE TABLE IF NOT EXISTS migration_run (id serial PRIMARY KEY, started_at timestamptz, ended_at timestamptz);
INSERT INTO migration_run (started_at) VALUES (clock_timestamp());
SELECT pg_sleep(0.5);
UPDATE migration_run SET ended_at = clock_timestamp() WHERE id = (SELECT max(id) FROM migration_run);
`

func TestPostgresLockSerializesConcurrentAcquisition(t *testing.T) {
	dsn := testDSN(t)
	first := openDB(t, dsn)
	second := openDB(t, dsn)

	unlock, err := dialect.Postgres{}.Lock(context.Background(), first)
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	acquired := make(chan error, 1)
	go func() {
		unlockSecond, err := dialect.Postgres{}.Lock(ctx, second)
		if err != nil {
			acquired <- err
			return
		}
		acquired <- unlockSecond(context.Background())
	}()

	select {
	case err := <-acquired:
		if err == nil {
			t.Fatal("second lock acquired while the first was held")
		}
		t.Fatalf("second lock failed while the first was held: %v", err)
	case <-time.After(300 * time.Millisecond):
	}

	if err := unlock(context.Background()); err != nil {
		t.Fatalf("release first lock: %v", err)
	}

	select {
	case err := <-acquired:
		if err != nil {
			t.Fatalf("second lock failed after release: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second lock was not acquired after the first was released")
	}
}

type noLockDialect struct{}

func (noLockDialect) Name() string               { return "no-lock" }
func (noLockDialect) Placeholder(int) string     { return "?" }
func (noLockDialect) Now() string                { return "NOW()" }
func (noLockDialect) BoolType() string           { return "BOOLEAN" }
func (noLockDialect) InsertMigrationSql() string { return "" }
func (noLockDialect) UpdateMigrationSql() string { return "" }

func TestUpRejectsDialectWithoutLock(t *testing.T) {
	dsn := testDSN(t)
	db := openDB(t, dsn)
	m := migration.New(db, noLockDialect{}, staticSource{})
	if err := m.Up(context.Background()); !errors.Is(err, migration.ErrLockUnsupported) {
		t.Fatalf("expected ErrLockUnsupported, got %v", err)
	}
}

func TestUpSerializesConcurrentInstancesOnLegacySchema(t *testing.T) {
	dsn := testDSN(t)
	db := openDB(t, dsn)
	if _, err := db.Exec(`
DROP TABLE IF EXISTS migration_run;
DROP TABLE IF EXISTS migrations;
CREATE TABLE migrations (version BIGINT PRIMARY KEY, last_migrate TIMESTAMP NOT NULL);
`); err != nil {
		t.Fatalf("prepare legacy schema: %v", err)
	}

	src := staticSource{migrations: []types.Migration{{
		Version:   1,
		Direction: types.Holding,
		SQL:       legacyHoldingSQL,
	}}}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = runUp(t, dsn, src)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("instance %d: %v", i, err)
		}
	}

	var runs, overlaps int
	if err := db.QueryRow(`SELECT count(*) FROM migration_run`).Scan(&runs); err != nil {
		t.Fatalf("count runs: %v", err)
	}
	if err := db.QueryRow(`
SELECT count(*)
FROM migration_run a
JOIN migration_run b ON a.id < b.id
WHERE a.ended_at > b.started_at
`).Scan(&overlaps); err != nil {
		t.Fatalf("count overlaps: %v", err)
	}
	if runs != 2 {
		t.Fatalf("expected 2 migration runs, got %d", runs)
	}
	if overlaps != 0 {
		t.Fatalf("holding migrations overlapped: %d", overlaps)
	}
}
