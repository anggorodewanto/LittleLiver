package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenDB_InMemory(t *testing.T) {
	t.Parallel()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB(:memory:) returned error: %v", err)
	}
	defer db.Close()

	// Verify WAL mode is enabled
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("PRAGMA journal_mode query failed: %v", err)
	}
	// In-memory databases use "memory" journal mode, not "wal"
	// WAL is only applicable to file-based databases
	if journalMode != "memory" {
		t.Errorf("expected journal_mode=memory for in-memory DB, got %q", journalMode)
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("PRAGMA foreign_keys query failed: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("expected foreign_keys=1, got %d", fkEnabled)
	}
}

func TestOpenDB_FileWAL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB(%s) returned error: %v", dbPath, err)
	}
	defer db.Close()

	// File-based DB should use WAL mode
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("PRAGMA journal_mode query failed: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected journal_mode=wal, got %q", journalMode)
	}
}

func TestRunMigrations_AppliesInOrder(t *testing.T) {
	t.Parallel()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	migDir := t.TempDir()
	// Create two migration files
	writeFile(t, filepath.Join(migDir, "001_first.sql"), "CREATE TABLE test_a (id TEXT PRIMARY KEY);")
	writeFile(t, filepath.Join(migDir, "002_second.sql"), "CREATE TABLE test_b (id TEXT PRIMARY KEY);")

	err = RunMigrations(db, migDir)
	if err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Both tables should exist
	assertTableExists(t, db, "test_a")
	assertTableExists(t, db, "test_b")
}

func TestRunMigrations_Idempotent(t *testing.T) {
	t.Parallel()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	migDir := t.TempDir()
	writeFile(t, filepath.Join(migDir, "001_test.sql"), "CREATE TABLE test_idem (id TEXT PRIMARY KEY);")

	err = RunMigrations(db, migDir)
	if err != nil {
		t.Fatalf("first RunMigrations failed: %v", err)
	}

	// Running again should not error
	err = RunMigrations(db, migDir)
	if err != nil {
		t.Fatalf("second RunMigrations failed: %v", err)
	}
}

func TestRunMigrations_EmptyDir(t *testing.T) {
	t.Parallel()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	migDir := t.TempDir()
	err = RunMigrations(db, migDir)
	if err != nil {
		t.Fatalf("RunMigrations on empty dir failed: %v", err)
	}
}

func TestRunMigrations_InvalidSQL(t *testing.T) {
	t.Parallel()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	migDir := t.TempDir()
	writeFile(t, filepath.Join(migDir, "001_bad.sql"), "THIS IS NOT VALID SQL;")

	err = RunMigrations(db, migDir)
	if err == nil {
		t.Fatal("expected RunMigrations to fail on invalid SQL, but it succeeded")
	}
}

func TestCoreSchema_TablesExist(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	tables := []string{"users", "babies", "baby_parents", "invites", "sessions", "push_subscriptions"}
	for _, table := range tables {
		assertTableExists(t, db, table)
	}
}

func TestCoreSchema_UsersColumns(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	expected := []string{"id", "google_id", "email", "name", "timezone", "created_at"}
	assertColumns(t, db, "users", expected)
}

func TestCoreSchema_BabiesColumns(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	expected := []string{"id", "name", "sex", "date_of_birth", "diagnosis_date", "kasai_date", "default_cal_per_feed", "notes", "created_at"}
	assertColumns(t, db, "babies", expected)
}

func TestCoreSchema_BabyParentsColumns(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	expected := []string{"baby_id", "user_id", "role", "joined_at"}
	assertColumns(t, db, "baby_parents", expected)
}

func TestCoreSchema_InvitesColumns(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	expected := []string{"code", "baby_id", "created_by", "used_by", "used_at", "expires_at", "created_at"}
	assertColumns(t, db, "invites", expected)
}

func TestCoreSchema_SessionsColumns(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	expected := []string{"id", "user_id", "token", "expires_at", "created_at"}
	assertColumns(t, db, "sessions", expected)
}

func TestCoreSchema_PushSubscriptionsColumns(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	expected := []string{"id", "user_id", "endpoint", "p256dh", "auth", "created_at"}
	assertColumns(t, db, "push_subscriptions", expected)
}

func TestCoreSchema_SentinelUserExists(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	var id, googleID, name string
	err := db.QueryRow("SELECT id, google_id, name FROM users WHERE id = 'deleted_user'").Scan(&id, &googleID, &name)
	if err != nil {
		t.Fatalf("sentinel user query failed: %v", err)
	}
	if id != "deleted_user" {
		t.Errorf("expected sentinel id=deleted_user, got %q", id)
	}
	if googleID != "__sentinel__" {
		t.Errorf("expected sentinel google_id=__sentinel__, got %q", googleID)
	}
	if name != "Deleted User" {
		t.Errorf("expected sentinel name=Deleted User, got %q", name)
	}
}

func TestCoreSchema_PushSubscriptionsEndpointUnique(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Insert a user first (for FK)
	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	// Insert first push subscription
	_, err = db.Exec("INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth) VALUES ('ps1', 'u1', 'https://example.com/push', 'key1', 'auth1')")
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	// Insert duplicate endpoint should fail
	_, err = db.Exec("INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth) VALUES ('ps2', 'u1', 'https://example.com/push', 'key2', 'auth2')")
	if err == nil {
		t.Fatal("expected UNIQUE constraint violation on push_subscriptions.endpoint, but insert succeeded")
	}
}

func TestCoreSchema_BabiesSexCheck(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Invalid sex value should fail
	_, err := db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Test', 'other', '2025-01-01')")
	if err == nil {
		t.Fatal("expected CHECK constraint violation on babies.sex, but insert succeeded")
	}

	// Valid sex value should succeed
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Test', 'male', '2025-01-01')")
	if err != nil {
		t.Fatalf("insert with valid sex failed: %v", err)
	}
}

func TestCoreSchema_SessionsTokenUnique(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	_, err = db.Exec("INSERT INTO sessions (id, user_id, token, expires_at) VALUES ('s1', 'u1', 'tok1', '2099-01-01')")
	if err != nil {
		t.Fatalf("first session insert failed: %v", err)
	}

	_, err = db.Exec("INSERT INTO sessions (id, user_id, token, expires_at) VALUES ('s2', 'u1', 'tok1', '2099-01-01')")
	if err == nil {
		t.Fatal("expected UNIQUE constraint violation on sessions.token, but insert succeeded")
	}
}

func TestCoreSchema_UsersGoogleIDUnique(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test1')")
	if err != nil {
		t.Fatalf("first user insert failed: %v", err)
	}

	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u2', 'g1', 'b@b.com', 'Test2')")
	if err == nil {
		t.Fatal("expected UNIQUE constraint violation on users.google_id, but insert succeeded")
	}
}

func TestCoreSchema_BabyParentsCascadeDelete(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Insert user and baby
	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Baby', 'female', '2025-01-01')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert baby_parents failed: %v", err)
	}

	// Delete baby should cascade
	_, err = db.Exec("DELETE FROM babies WHERE id = 'b1'")
	if err != nil {
		t.Fatalf("delete baby failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM baby_parents WHERE baby_id = 'b1'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 baby_parents after cascade delete, got %d", count)
	}
}

func TestRunMigrations_NonExistentDir(t *testing.T) {
	t.Parallel()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	err = RunMigrations(db, "/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for non-existent migrations dir, got nil")
	}
}

func TestRunMigrations_SkipsNonSQLAndDirs(t *testing.T) {
	t.Parallel()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	migDir := t.TempDir()
	// Create a subdirectory (should be skipped)
	if err := os.Mkdir(filepath.Join(migDir, "subdir"), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	// Create a non-.sql file (should be skipped)
	writeFile(t, filepath.Join(migDir, "readme.txt"), "not a migration")
	// Create an actual migration
	writeFile(t, filepath.Join(migDir, "001_test.sql"), "CREATE TABLE skip_test (id TEXT PRIMARY KEY);")

	err = RunMigrations(db, migDir)
	if err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	assertTableExists(t, db, "skip_test")
}

func TestOpenDB_InvalidPath(t *testing.T) {
	t.Parallel()
	// Try to open a DB in a non-existent directory - should fail on Ping
	_, err := OpenDB("/nonexistent/dir/that/cannot/exist/test.db")
	if err == nil {
		t.Fatal("expected error for invalid DB path, got nil")
	}
}

func TestRunMigrations_ClosedDB(t *testing.T) {
	t.Parallel()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	db.Close() // close it immediately

	migDir := t.TempDir()
	writeFile(t, filepath.Join(migDir, "001_test.sql"), "CREATE TABLE t (id TEXT);")

	err = RunMigrations(db, migDir)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

// --- helpers ---

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}

	migDir := filepath.Join(findProjectRoot(t), "migrations")
	err = RunMigrations(db, migDir)
	if err != nil {
		db.Close()
		t.Fatalf("RunMigrations failed: %v", err)
	}
	return db
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	// We're in internal/store, so go up two levels to backend/
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	// Navigate up from internal/store to backend
	root := filepath.Join(dir, "..", "..")
	// Verify migrations dir exists
	migDir := filepath.Join(root, "migrations")
	if _, err := os.Stat(migDir); os.IsNotExist(err) {
		t.Fatalf("migrations dir not found at %s", migDir)
	}
	return root
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile(%s) failed: %v", path, err)
	}
}

func assertTableExists(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
	if err != nil {
		t.Errorf("table %q does not exist: %v", tableName, err)
	}
}

func assertColumns(t *testing.T, db *sql.DB, tableName string, expected []string) {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s) failed: %v", tableName, err)
	}
	defer rows.Close()

	got := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		got[name] = true
	}

	for _, col := range expected {
		if !got[col] {
			t.Errorf("table %s: expected column %q not found. Got columns: %v", tableName, col, keys(got))
		}
	}

	if len(got) != len(expected) {
		t.Errorf("table %s: expected %d columns, got %d. Columns: %v", tableName, len(expected), len(got), keys(got))
	}
}

func keys(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
