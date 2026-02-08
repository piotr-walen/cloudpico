package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cloudpico-tools/migrate"

	_ "github.com/mattn/go-sqlite3"
)

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
		fmt.Fprintf(os.Stderr, "usage: %s <command>\n  migrate  apply pending schema/seed migrations\n", os.Args[0])
		os.Exit(1)
	}

	switch os.Args[1] {
	case "migrate":
		if err := migrate.Run(conn); err != nil {
			fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("migrations applied")
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

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db ping: %w", err)
	}

	return db, nil
}

func buildDSN(dbPath string) (string, error) {
	params := []string{
		"_foreign_keys=on",
		"_busy_timeout=5000",
		"_journal_mode=WAL",
	}

	if strings.HasPrefix(dbPath, "file:") {
		sep := "?"
		if strings.Contains(dbPath, "?") {
			sep = "&"
		}
		return dbPath + sep + strings.Join(params, "&"), nil
	}

	return fmt.Sprintf("file:%s?%s", dbPath, strings.Join(params, "&")), nil
}
