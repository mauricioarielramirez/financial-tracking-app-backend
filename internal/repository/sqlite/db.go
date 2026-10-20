// Package sqlite contains the concrete implementation of the
// internal/repository interfaces over SQLite (Technical Plan section 1:
// the repository pattern is what allows migrating to Postgres/MySQL
// without touching the rest of the system).
//
// Note on the driver: github.com/mattn/go-sqlite3 (CGO-based) is used
// instead of a pure-Go alternative, because the environment where this
// first artifact was generated had no network access to the usual pure-Go
// modules (modernc.org/sqlite via gitlab.com, or any package depending on
// golang.org/x/sys). Building the project requires an available C
// compiler (on Windows: MSYS2/mingw-w64 or TDM-GCC) and `CGO_ENABLED=1`
// (which is Go's default if it detects a compiler). If avoiding CGO is
// preferred later on, it's enough to replace this package with an
// equivalent implementation over a pure-Go driver: the
// internal/repository interfaces don't change.
package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"database/sql"

	_ "github.com/mattn/go-sqlite3" // registers the "sqlite3" driver in database/sql
)

// Open opens the connection to the SQLite database at dsn (e.g.
// "./finance.db") and enables foreign keys, which SQLite ships disabled by
// default.
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("abriendo sqlite (%s): %w", dsn, err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("conectando a sqlite (%s): %w", dsn, err)
	}
	// SQLite doesn't handle concurrent writes from multiple connections
	// well; for a single-user app (RNF-01) serializing is enough.
	db.SetMaxOpenConns(1)
	return db, nil
}

// Migrate applies, in order, the *.up.sql files in migrationsPath that
// aren't already listed in the schema_migrations table.
//
// It's a deliberately simple runner (not golang-migrate) so as not to add
// another dependency while access to proxy.golang.org is blocked; it
// follows the same naming convention (NNNN_description.up.sql / .down.sql)
// so adopting golang-migrate later doesn't require rewriting the
// migration files, just the runner.
func Migrate(ctx context.Context, db *sql.DB, migrationsPath string) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`); err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("reading schema_migrations: %w", err)
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()

	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("reading migrations directory (%s): %w", migrationsPath, err)
	}

	var upFiles []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		upFiles = append(upFiles, e.Name())
	}
	sort.Strings(upFiles)

	for _, fileName := range upFiles {
		version := strings.TrimSuffix(fileName, ".up.sql")
		if applied[version] {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsPath, fileName))
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", fileName, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("applying migration %s: %w", fileName, err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("recording migration %s: %w", fileName, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}
