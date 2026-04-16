package migrations

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrate_EmptyList(t *testing.T) {
	t.Parallel()
	db := openDB(t)
	if err := Migrate(db, nil, "sqlite"); err != nil {
		t.Fatalf("Migrate with nil: %v", err)
	}
	// Table should still be created
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows, got %d", count)
	}
}

func TestMigrate_AppliesMigrations(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{Version: 1, Name: "create_users", Statements: []string{
			"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
		}},
		{Version: 2, Name: "add_email", Statements: []string{
			"ALTER TABLE users ADD COLUMN email TEXT",
		}},
	}

	if err := Migrate(db, migrations, "sqlite"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Both migrations should be recorded
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 applied migrations, got %d", count)
	}

	// Table should exist and be usable
	if _, err := db.Exec("INSERT INTO users (name, email) VALUES ('alice', 'a@b.com')"); err != nil {
		t.Fatalf("insert into migrated table: %v", err)
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{Version: 1, Name: "create_foo", Statements: []string{
			"CREATE TABLE foo (id INTEGER PRIMARY KEY)",
		}},
	}

	// Apply twice
	if err := Migrate(db, migrations, "sqlite"); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(db, migrations, "sqlite"); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 migration record after double-apply, got %d", count)
	}
}

func TestMigrate_MultipleStatements(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{Version: 1, Name: "multi", Statements: []string{
			"CREATE TABLE items (id INTEGER PRIMARY KEY, val TEXT)",
			"INSERT INTO items (val) VALUES ('hello')",
			"INSERT INTO items (val) VALUES ('world')",
		}},
	}

	if err := Migrate(db, migrations, "sqlite"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}
}

func TestMigrate_SkipsBlankStatements(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{Version: 1, Name: "blanks", Statements: []string{
			"CREATE TABLE blank_test (id INTEGER PRIMARY KEY)",
			"",
			"   ",
		}},
	}

	if err := Migrate(db, migrations, "sqlite"); err != nil {
		t.Fatalf("Migrate with blanks: %v", err)
	}
}

func TestMigrate_RollsBackOnFailure(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{Version: 1, Name: "create_table", Statements: []string{
			"CREATE TABLE good (id INTEGER PRIMARY KEY)",
		}},
		{Version: 2, Name: "bad_migration", Statements: []string{
			"CREATE TABLE good (id INTEGER PRIMARY KEY)", // duplicate — will fail
		}},
	}

	err := Migrate(db, migrations, "sqlite")
	if err == nil {
		t.Fatal("expected error for duplicate table creation")
	}

	// First migration should have succeeded
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 applied migration (rollback), got %d", count)
	}

	// The good table should exist
	if _, err := db.Exec("INSERT INTO good (id) VALUES (1)"); err != nil {
		t.Fatalf("insert into good table: %v", err)
	}
}

func TestMigrate_VersionOrdering(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	// Apply version 1 only first
	if err := Migrate(db, []Migration{
		{Version: 1, Name: "step1", Statements: []string{"CREATE TABLE step1 (id INTEGER PRIMARY KEY)"}},
	}, "sqlite"); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}

	// Now apply 1 + 2 — only 2 should run
	if err := Migrate(db, []Migration{
		{Version: 1, Name: "step1", Statements: []string{"CREATE TABLE step1 (id INTEGER PRIMARY KEY)"}},
		{Version: 2, Name: "step2", Statements: []string{"CREATE TABLE step2 (id INTEGER PRIMARY KEY)"}},
	}, "sqlite"); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 migrations, got %d", count)
	}
}

func TestStatus_AllApplied(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{Version: 1, Name: "first", Statements: []string{"CREATE TABLE t1 (id INTEGER PRIMARY KEY)"}},
		{Version: 2, Name: "second", Statements: []string{"CREATE TABLE t2 (id INTEGER PRIMARY KEY)"}},
	}

	if err := Migrate(db, migrations, "sqlite"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	applied, pending, err := Status(db, migrations, "sqlite")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(applied) != 2 {
		t.Fatalf("expected 2 applied, got %d", len(applied))
	}
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending, got %d", len(pending))
	}
}

func TestStatus_NoneApplied(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{Version: 1, Name: "first", Statements: []string{"CREATE TABLE t1 (id INTEGER PRIMARY KEY)"}},
	}

	applied, pending, err := Status(db, migrations, "sqlite")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("expected 0 applied, got %d", len(applied))
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
}

func TestStatus_PartiallyApplied(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	allMigrations := []Migration{
		{Version: 1, Name: "first", Statements: []string{"CREATE TABLE t1 (id INTEGER PRIMARY KEY)"}},
		{Version: 2, Name: "second", Statements: []string{"CREATE TABLE t2 (id INTEGER PRIMARY KEY)"}},
		{Version: 3, Name: "third", Statements: []string{"CREATE TABLE t3 (id INTEGER PRIMARY KEY)"}},
	}

	// Apply only the first
	if err := Migrate(db, allMigrations[:1], "sqlite"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	applied, pending, err := Status(db, allMigrations, "sqlite")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(applied) != 1 || applied[0].Version != 1 {
		t.Fatalf("expected 1 applied (v1), got %v", applied)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pending))
	}
}

func TestStatus_EmptyMigrations(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	applied, pending, err := Status(db, nil, "sqlite")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("expected 0 applied, got %d", len(applied))
	}
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending, got %d", len(pending))
	}
}

func TestRollback_Success(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{
			Version: 1,
			Name:    "create_users",
			Statements: []string{
				"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
			},
			Rollback: []string{
				"DROP TABLE users",
			},
		},
	}

	// Apply migration
	if err := Migrate(db, migrations, "sqlite"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Verify table exists
	if _, err := db.Exec("INSERT INTO users (name) VALUES ('alice')"); err != nil {
		t.Fatalf("insert before rollback: %v", err)
	}

	// Rollback
	if err := Rollback(db, migrations, 1, "sqlite"); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	// Table should be gone
	if _, err := db.Exec("SELECT * FROM users"); err == nil {
		t.Fatal("expected error accessing dropped table")
	}

	// Migration record should be removed
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 migration records after rollback, got %d", count)
	}
}

func TestRollback_NoRollbackDefined(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{
			Version: 1,
			Name:    "create_users",
			Statements: []string{
				"CREATE TABLE users (id INTEGER PRIMARY KEY)",
			},
			// No Rollback defined
		},
	}

	if err := Migrate(db, migrations, "sqlite"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	err := Rollback(db, migrations, 1, "sqlite")
	if err == nil {
		t.Fatal("expected error for migration without rollback")
	}
}

func TestRollback_NotApplied(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{
			Version: 1,
			Name:    "create_users",
			Statements: []string{
				"CREATE TABLE users (id INTEGER PRIMARY KEY)",
			},
			Rollback: []string{
				"DROP TABLE users",
			},
		},
	}

	err := Rollback(db, migrations, 1, "sqlite")
	if err == nil {
		t.Fatal("expected error for non-applied migration")
	}
}

func TestRollback_VersionNotFound(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{
			Version: 1,
			Name:    "create_users",
			Statements: []string{
				"CREATE TABLE users (id INTEGER PRIMARY KEY)",
			},
			Rollback: []string{"DROP TABLE users"},
		},
	}

	err := Rollback(db, migrations, 999, "sqlite")
	if err == nil {
		t.Fatal("expected error for non-existent version")
	}
}

func TestRollbackLast_Success(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	migrations := []Migration{
		{
			Version: 1,
			Name:    "create_users",
			Statements: []string{
				"CREATE TABLE users (id INTEGER PRIMARY KEY)",
			},
			Rollback: []string{"DROP TABLE users"},
		},
		{
			Version: 2,
			Name:    "create_orders",
			Statements: []string{
				"CREATE TABLE orders (id INTEGER PRIMARY KEY)",
			},
			Rollback: []string{"DROP TABLE orders"},
		},
	}

	// Apply both migrations
	if err := Migrate(db, migrations, "sqlite"); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Rollback last (version 2)
	if err := RollbackLast(db, migrations, "sqlite"); err != nil {
		t.Fatalf("RollbackLast: %v", err)
	}

	// Version 1 should remain
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 migration record, got %d", count)
	}

	// Version 2 table should be gone
	if _, err := db.Exec("SELECT * FROM orders"); err == nil {
		t.Fatal("expected error accessing dropped table")
	}

	// Version 1 table should still exist
	if _, err := db.Exec("INSERT INTO users (id) VALUES (1)"); err != nil {
		t.Fatalf("users table should exist: %v", err)
	}
}

func TestRollbackLast_NoMigrations(t *testing.T) {
	t.Parallel()
	db := openDB(t)

	err := RollbackLast(db, nil, "sqlite")
	if err == nil {
		t.Fatal("expected error when no migrations exist")
	}
}
