package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cloudpico-server/internal/config"

	_ "github.com/mattn/go-sqlite3"
)

func Open(cfg config.Config) (*sql.DB, error) {

	dsn, err := buildDSN(cfg)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(cfg.SQLiteDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}

	// Pooling (SQLite is typically best with low concurrency; tune if needed)
	if cfg.SQLiteMaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.SQLiteMaxOpenConns)
	}
	if cfg.SQLiteMaxIdleConns >= 0 {
		db.SetMaxIdleConns(cfg.SQLiteMaxIdleConns)
	}
	if cfg.SQLiteConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.SQLiteConnMaxLifetime)
	}

	// Validate connectivity early
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db ping: %w", err)
	}

	return db, nil
}

func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

func buildDSN(cfg config.Config) (string, error) {
	if cfg.SQLiteDSN != "" {
		return cfg.SQLiteDSN, nil
	}

	// Ensure directory exists for file-backed sqlite db
	path := cfg.SQLitePath
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// Reasonable defaults:
	// - foreign_keys=on: enforce FK constraints
	// - busy_timeout: helps with "database is locked" under concurrent dev use
	// - journal_mode=WAL: better concurrent reads/writes in dev
	// NOTE: journal_mode via DSN is supported by mattn/go-sqlite3.
	params := []string{
		"_foreign_keys=on",
		"_busy_timeout=5000",
		"_journal_mode=WAL",
	}

	// If caller provided something like "file:/data/app.db?x=y" as Path, don’t double-wrap
	if strings.HasPrefix(path, "file:") {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		return path + sep + strings.Join(params, "&"), nil
	}

	// Default: plain file path. You can also use "file:" prefix; both work.
	return fmt.Sprintf("file:%s?%s", path, strings.Join(params, "&")), nil
}
