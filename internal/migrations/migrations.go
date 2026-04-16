package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Migration describes one versioned, transactional schema change.
type Migration struct {
	Version     int
	Name       string
	Statements  []string
	Rollback    []string // Optional rollback statements
	Dialect    string    // Optional; if set, only applies to that dialect ("sqlite" or "postgres")
}

// Migrate applies all pending migrations in order.
func Migrate(db *sql.DB, migrations []Migration, dialect string) error {
	if dialect == "" {
		dialect = "sqlite"
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	for _, m := range migrations {
		// Skip migrations that don't match the current dialect
		if m.Dialect != "" && m.Dialect != dialect {
			continue
		}

		applied, err := isApplied(db, m.Version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", m.Version, err)
		}

		for _, stmt := range m.Statements {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err := tx.Exec(stmt); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("apply migration %d (%s): %w", m.Version, m.Name, err)
			}
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_migrations(version, name, applied_at) VALUES(?, ?, ?)`,
			m.Version, m.Name, time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.Version, err)
		}
	}

	return nil
}

// Rollback reverses a migration by version.
func Rollback(db *sql.DB, migrations []Migration, targetVersion int, dialect string) error {
	if dialect == "" {
		dialect = "sqlite"
	}

	// Find the migration to rollback
	var migration Migration
	found := false
	for _, m := range migrations {
		if m.Version == targetVersion {
			migration = m
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("migration version %d not found", targetVersion)
	}

	// Check if migration is applied
	applied, err := isApplied(db, targetVersion)
	if err != nil {
		return err
	}
	if !applied {
		return fmt.Errorf("migration %d is not applied", targetVersion)
	}

	// Check if rollback is available
	if len(migration.Rollback) == 0 {
		return fmt.Errorf("migration %d has no rollback defined", targetVersion)
	}

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin rollback %d: %w", targetVersion, err)
	}

	for _, stmt := range migration.Rollback {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("rollback migration %d (%s): %w", targetVersion, migration.Name, err)
		}
	}

	if _, err := tx.Exec(`DELETE FROM schema_migrations WHERE version = ?`, targetVersion); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("remove migration record %d: %w", targetVersion, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rollback %d: %w", targetVersion, err)
	}

	return nil
}

// RollbackLast undoes the most recent migration.
func RollbackLast(db *sql.DB, migrations []Migration, dialect string) error {
	if dialect == "" {
		dialect = "sqlite"
	}

	// Get the highest applied version
	var lastVersion int
	err := db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&lastVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("no migrations to rollback")
		}
		return fmt.Errorf("find last migration: %w", err)
	}

	return Rollback(db, migrations, lastVersion, dialect)
}

// Status returns applied and pending migrations.
func Status(db *sql.DB, migrations []Migration, dialect string) (applied, pending []Migration, err error) {
	if dialect == "" {
		dialect = "sqlite"
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)
	`); err != nil {
		return nil, nil, fmt.Errorf("create schema_migrations table: %w", err)
	}

	for _, m := range migrations {
		// Skip migrations that don't match the current dialect
		if m.Dialect != "" && m.Dialect != dialect {
			continue
		}
		ok, serr := isApplied(db, m.Version)
		if serr != nil {
			return nil, nil, serr
		}
		if ok {
			applied = append(applied, m)
		} else {
			pending = append(pending, m)
		}
	}
	return applied, pending, nil
}

func isApplied(db *sql.DB, version int) (bool, error) {
	var one int
	err := db.QueryRow(`SELECT 1 FROM schema_migrations WHERE version = ?`, version).Scan(&one)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, fmt.Errorf("check migration %d: %w", version, err)
}
