// Package sqlite contiene la implementación concreta de las interfaces de
// internal/repository sobre SQLite (Plan Técnico sección 1: el repository
// pattern es lo que permite migrar a Postgres/MySQL sin tocar el resto del
// sistema).
//
// Nota sobre el driver: se usa github.com/mattn/go-sqlite3 (basado en CGO)
// en lugar de una alternativa pura-Go, porque en el entorno donde se generó
// este primer artefacto no había acceso de red a los módulos pure-Go
// habituales (modernc.org/sqlite vía gitlab.com, o cualquier paquete que
// dependa de golang.org/x/sys). Para compilar el proyecto hace falta un
// compilador de C disponible (en Windows: MSYS2/mingw-w64 o TDM-GCC) y
// `CGO_ENABLED=1` (que es el default de Go si detecta un compilador). Si
// más adelante se prefiere evitar CGO, alcanza con reemplazar este paquete
// por una implementación equivalente sobre un driver pure-Go: las
// interfaces de internal/repository no cambian.
package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"database/sql"

	_ "github.com/mattn/go-sqlite3" // registra el driver "sqlite3" en database/sql
)

// Open abre la conexión a la base SQLite en dsn (p. ej. "./finance.db") y
// habilita foreign keys, que SQLite trae deshabilitadas por defecto.
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("abriendo sqlite (%s): %w", dsn, err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("conectando a sqlite (%s): %w", dsn, err)
	}
	// SQLite no soporta bien la escritura concurrente desde múltiples
	// conexiones; para una app single-user (RNF-01) alcanza con serializar.
	db.SetMaxOpenConns(1)
	return db, nil
}

// Migrate aplica, en orden, los archivos *.up.sql de migrationsPath que
// todavía no figuren en la tabla schema_migrations.
//
// Es un runner deliberadamente simple (no golang-migrate) para no sumar una
// dependencia más mientras el acceso a proxy.golang.org esté bloqueado;
// sigue la misma convención de nombres (NNNN_descripcion.up.sql / .down.sql)
// por lo que adoptar golang-migrate más adelante no requiere reescribir los
// archivos de migración, sólo el runner.
func Migrate(ctx context.Context, db *sql.DB, migrationsPath string) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`); err != nil {
		return fmt.Errorf("creando schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("leyendo schema_migrations: %w", err)
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
		return fmt.Errorf("leyendo directorio de migraciones (%s): %w", migrationsPath, err)
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
			return fmt.Errorf("leyendo migración %s: %w", fileName, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("aplicando migración %s: %w", fileName, err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("registrando migración %s: %w", fileName, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}
