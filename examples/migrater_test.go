package examples

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"testing"

	_ "github.com/lib/pq"

	"github.com/gobkc/migration"
	"github.com/gobkc/migration/dialect"
	"github.com/gobkc/migration/source"
)

//go:embed testdata/*
var Files embed.FS

func TestNewMigrator(t *testing.T) {
	dsn := `postgres://postgres:postgres@cfg-envs:5432/%s`
	db, err := sql.Open("postgres", fmt.Sprintf(dsn, `?sslmode=disable`))
	if err != nil {
		panic(err)
	}
	// defer db.Exec("DROP DATABASE IF EXISTS migration_test")
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", "migration_test").Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}

	if !exists {
		_, err = db.Exec("CREATE DATABASE migration_test")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Database created!")
	}
	db.Close()

	dsn = fmt.Sprintf(dsn, `migration_test?sslmode=disable`)
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}
	m := migration.New(
		db,
		dialect.Postgres{},
		source.NewEmbed(Files),
	)

	if err := m.Up(context.Background()); err != nil {
		log.Fatal(err)
	}
}
