package dbutil

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

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
	if err != nil {
		slog.Warn("database created by another instance",
			slog.String(`db`, dbName),
			slog.String(`err`, err.Error()),
		)
	}

	slog.Info("database created", "db", dbName)

	return nil
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

type ParseDSNInfo struct {
	User     string
	Password string
	Host     string
	Port     string
	DBName   string
	Params   map[string]string
}

// ParseDSN parse database DSN
func ParseDSN(dsn string) (*ParseDSNInfo, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}

	info := &ParseDSNInfo{
		Params: make(map[string]string),
	}
	if u.User != nil {
		info.User = u.User.Username()
		info.Password, _ = u.User.Password()
	}
	host := u.Host
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		info.Host = parts[0]
		info.Port = parts[1]
	} else {
		info.Host = host
	}
	info.DBName = strings.TrimPrefix(u.Path, "/")

	for k, v := range u.Query() {
		if len(v) > 0 {
			info.Params[k] = v[0]
		}
	}

	return info, nil
}
