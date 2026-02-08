// Package migrate runs SQLite schema migrations using a versioned migration table.
// Migration files are named with a 4-digit prefix for order: 0001_name.sql, 0002_other.sql.
package migrate

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"regexp"
	"sort"
)

//go:embed sql/*.sql
var sqlFS embed.FS

const (
	migrationsDir = "sql"
	tableName     = "schema_migrations"
)

var migrationFileRe = regexp.MustCompile(`^(\d{4})_(.+)\.sql$`)

// Run ensures the schema_migrations table exists, then applies any embedded
// migrations that have not yet been run, in order by version.
func Run(db *sql.DB) error {
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	applied, err := appliedVersions(db)
	if err != nil {
		return fmt.Errorf("list applied migrations: %w", err)
	}

	entries, err := fs.ReadDir(sqlFS, migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var pending []migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		version, name, ok := parseMigrationFilename(e.Name())
		if !ok {
			continue
		}
		if applied[version] {
			continue
		}
		body, err := fs.ReadFile(sqlFS, migrationsDir+"/"+e.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", e.Name(), err)
		}
		pending = append(pending, migration{version: version, name: name, body: string(body)})
	}

	sort.Slice(pending, func(i, j int) bool { return pending[i].version < pending[j].version })

	for _, m := range pending {
		if err := apply(db, m); err != nil {
			return fmt.Errorf("apply %s: %w", m.version+"_"+m.name+".sql", err)
		}
		slog.Info("migration applied", "version", m.version, "name", m.name)
	}

	return nil
}

type migration struct {
	version string
	name    string
	body    string
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + tableName + ` (
			version   TEXT PRIMARY KEY,
			name      TEXT NOT NULL,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)
	`)
	return err
}

func appliedVersions(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT version FROM " + tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func parseMigrationFilename(filename string) (version, name string, ok bool) {
	m := migrationFileRe.FindStringSubmatch(filename)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

func apply(db *sql.DB, m migration) error {
	if _, err := db.Exec(m.body); err != nil {
		return err
	}
	_, err := db.Exec(
		"INSERT INTO "+tableName+" (version, name) VALUES (?, ?)",
		m.version, m.name,
	)
	return err
}
