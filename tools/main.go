package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed sql/schema.sql
var schemaSQL string

//go:embed sql/seed.sql
var seedSQL string

//go:embed sql/post-seed.sql
var postSeedSQL string

func main() {
	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath == "" {
		dbPath = "../dev/sqlite/app.db"
	}
	dbPath = filepath.Clean(dbPath)

	conn, err := Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db open: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Error("db close", "err", closeErr)
		}
	}()

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <command>\n", os.Args[0])
		os.Exit(1)
	}

	switch os.Args[1] {
	case "schema":
		if _, err := conn.Exec(schemaSQL); err != nil {
			fmt.Fprintf(os.Stderr, "db exec: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("schema applied")
	case "seed":
		if _, err := conn.Exec(seedSQL); err != nil {
			fmt.Fprintf(os.Stderr, "db exec: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("seed applied")
		rows, err := conn.Query(postSeedSQL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "db query: %v\n", err)
			os.Exit(1)
		}
		defer rows.Close()
		for rows.Next() {
			var tableName string
			var rowsCount int
			if err := rows.Scan(&tableName, &rowsCount); err != nil {
				fmt.Fprintf(os.Stderr, "db scan: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Seeded %d rows in %s\n", rowsCount, tableName)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func Open(dbPath string) (*sql.DB, error) {
	dsn, err := buildDSN(dbPath)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}

	// Validate connectivity early
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db ping: %w", err)
	}

	return db, nil
}

func buildDSN(dbPath string) (string, error) {

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
	if strings.HasPrefix(dbPath, "file:") {
		sep := "?"
		if strings.Contains(dbPath, "?") {
			sep = "&"
		}
		return dbPath + sep + strings.Join(params, "&"), nil
	}

	// Default: plain file path. You can also use "file:" prefix; both work.
	return fmt.Sprintf("file:%s?%s", dbPath, strings.Join(params, "&")), nil
}
