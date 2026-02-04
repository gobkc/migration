package dbutil

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
)

func EnsureDatabase(dsn string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	parsed, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("invalid dsn: %w", err)
	}

	dbName := strings.TrimPrefix(parsed.Path, "/")
	if dbName == "" {
		return fmt.Errorf("dsn must include database name")
	}

	adminURL := *parsed
	adminURL.Path = "/"

	conn, err := pgx.Connect(ctx, adminURL.String())
	if err != nil {
		return fmt.Errorf("connect postgres failed: %w", err)
	}
	defer conn.Close(ctx)

	var exists bool
	err = conn.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)`,
		dbName,
	).Scan(&exists)

	if err != nil {
		return fmt.Errorf("check database existence failed: %w", err)
	}

	if exists {
		slog.Debug("database already exists", "db", dbName)
		return nil
	}

	createSQL := "CREATE DATABASE " + quoteIdentifier(dbName)

	_, err = conn.Exec(ctx, createSQL)
	if err == nil {
		slog.Info("database created", "db", dbName)
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case `42P04`, `23505`:
			slog.Warn("database created by another instance", "db", dbName, "code", pgErr.Code)
			return nil
		}
	}

	return fmt.Errorf("create database failed: %w", err)
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
