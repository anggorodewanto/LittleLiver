package store

import (
	"database/sql"
	"testing"
	"time"
)

func TestDeleteAccount_ReturnsNoErrorForValidUser(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}
}

func TestDeleteAccount_UserRecordDeleted(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = 'u1'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected user to be deleted, but found %d records", count)
	}
}

func TestDeleteAccount_SentinelUserPreserved(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = 'deleted_user'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected sentinel user to still exist, found %d", count)
	}
}

func TestDeleteAccount_SessionsCascadeDeleted(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO sessions (id, user_id, token, expires_at) VALUES ('s1', 'u1', 'tok1', '2099-01-01')")
	if err != nil {
		t.Fatalf("insert session failed: %v", err)
	}

	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id = 'u1'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected sessions to be cascade deleted, found %d", count)
	}
}

func TestDeleteAccount_BabyParentsCascadeDeleted(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Create two users linked to same baby (so baby is NOT deleted)
	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test1')")
	if err != nil {
		t.Fatalf("insert user1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u2', 'g2', 'b@b.com', 'Test2')")
	if err != nil {
		t.Fatalf("insert user2 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Baby', 'female', '2025-01-01')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert bp1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u2')")
	if err != nil {
		t.Fatalf("insert bp2 failed: %v", err)
	}

	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	// baby_parents for u1 should be gone
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM baby_parents WHERE user_id = 'u1'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected baby_parents for u1 to be cascade deleted, found %d", count)
	}

	// baby should still exist (u2 is still linked)
	err = db.QueryRow("SELECT COUNT(*) FROM babies WHERE id = 'b1'").Scan(&count)
	if err != nil {
		t.Fatalf("count babies failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected baby to still exist, found %d", count)
	}
}

func TestDeleteAccount_PushSubscriptionsCascadeDeleted(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth) VALUES ('ps1', 'u1', 'https://push.example.com', 'key', 'auth')")
	if err != nil {
		t.Fatalf("insert push sub failed: %v", err)
	}

	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM push_subscriptions WHERE user_id = 'u1'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected push_subscriptions to be cascade deleted, found %d", count)
	}
}

func TestDeleteAccount_LastParentBabyDeleted(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

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
		t.Fatalf("insert bp failed: %v", err)
	}

	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM babies WHERE id = 'b1'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected last-parent baby to be deleted, found %d", count)
	}
}

func TestDeleteAccount_SharedBabyNotDeleted(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test1')")
	if err != nil {
		t.Fatalf("insert user1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u2', 'g2', 'b@b.com', 'Test2')")
	if err != nil {
		t.Fatalf("insert user2 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Baby', 'male', '2025-01-01')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert bp1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u2')")
	if err != nil {
		t.Fatalf("insert bp2 failed: %v", err)
	}

	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM babies WHERE id = 'b1'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected shared baby to still exist, found %d", count)
	}
}

func TestDeleteAccount_InvitesCreatedByUserDeleted(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test1')")
	if err != nil {
		t.Fatalf("insert user1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u2', 'g2', 'b@b.com', 'Test2')")
	if err != nil {
		t.Fatalf("insert user2 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Baby', 'female', '2025-01-01')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert bp1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u2')")
	if err != nil {
		t.Fatalf("insert bp2 failed: %v", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour).UTC().Format(time.DateTime)
	_, err = db.Exec("INSERT INTO invites (code, baby_id, created_by, expires_at) VALUES ('111111', 'b1', 'u1', ?)", expiresAt)
	if err != nil {
		t.Fatalf("insert invite failed: %v", err)
	}

	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM invites WHERE created_by = 'u1'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected invites created by u1 to be deleted, found %d", count)
	}
}

func TestDeleteAccount_InvitesUsedByAnonymized(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test1')")
	if err != nil {
		t.Fatalf("insert user1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u2', 'g2', 'b@b.com', 'Test2')")
	if err != nil {
		t.Fatalf("insert user2 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Baby', 'female', '2025-01-01')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u2')")
	if err != nil {
		t.Fatalf("insert bp failed: %v", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour).UTC().Format(time.DateTime)
	usedAt := time.Now().UTC().Format(time.DateTime)
	_, err = db.Exec("INSERT INTO invites (code, baby_id, created_by, used_by, used_at, expires_at) VALUES ('222222', 'b1', 'u2', 'u1', ?, ?)", usedAt, expiresAt)
	if err != nil {
		t.Fatalf("insert invite failed: %v", err)
	}

	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	var usedBy sql.NullString
	err = db.QueryRow("SELECT used_by FROM invites WHERE code = '222222'").Scan(&usedBy)
	if err != nil {
		t.Fatalf("query invite failed: %v", err)
	}
	if !usedBy.Valid || usedBy.String != "deleted_user" {
		t.Errorf("expected used_by='deleted_user', got %v", usedBy)
	}
}

func TestDeleteAccount_TableDrivenAnonymization(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Create a test table with logged_by and updated_by columns
	_, err := db.Exec(`CREATE TABLE test_metrics (
		id TEXT PRIMARY KEY,
		baby_id TEXT,
		logged_by TEXT REFERENCES users(id),
		updated_by TEXT REFERENCES users(id),
		value REAL
	)`)
	if err != nil {
		t.Fatalf("create test_metrics failed: %v", err)
	}

	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	_, err = db.Exec("INSERT INTO test_metrics (id, baby_id, logged_by, updated_by, value) VALUES ('m1', 'b1', 'u1', 'u1', 42.0)")
	if err != nil {
		t.Fatalf("insert metric failed: %v", err)
	}

	// Pass the test table in the anonymization list
	err = DeleteAccount(db, "u1", []string{"test_metrics"})
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	var loggedBy, updatedBy string
	err = db.QueryRow("SELECT logged_by, updated_by FROM test_metrics WHERE id = 'm1'").Scan(&loggedBy, &updatedBy)
	if err != nil {
		t.Fatalf("query metric failed: %v", err)
	}
	if loggedBy != "deleted_user" {
		t.Errorf("expected logged_by='deleted_user', got %q", loggedBy)
	}
	if updatedBy != "deleted_user" {
		t.Errorf("expected updated_by='deleted_user', got %q", updatedBy)
	}
}

func TestDeleteAccount_TableDrivenAnonymizationMultipleTables(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Create two test tables
	_, err := db.Exec(`CREATE TABLE test_feedings (
		id TEXT PRIMARY KEY,
		logged_by TEXT REFERENCES users(id),
		updated_by TEXT REFERENCES users(id)
	)`)
	if err != nil {
		t.Fatalf("create test_feedings failed: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE test_stools (
		id TEXT PRIMARY KEY,
		logged_by TEXT REFERENCES users(id),
		updated_by TEXT REFERENCES users(id)
	)`)
	if err != nil {
		t.Fatalf("create test_stools failed: %v", err)
	}

	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	_, err = db.Exec("INSERT INTO test_feedings (id, logged_by, updated_by) VALUES ('f1', 'u1', 'u1')")
	if err != nil {
		t.Fatalf("insert feeding failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO test_stools (id, logged_by, updated_by) VALUES ('s1', 'u1', 'u1')")
	if err != nil {
		t.Fatalf("insert stool failed: %v", err)
	}

	err = DeleteAccount(db, "u1", []string{"test_feedings", "test_stools"})
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	var loggedBy string
	err = db.QueryRow("SELECT logged_by FROM test_feedings WHERE id = 'f1'").Scan(&loggedBy)
	if err != nil {
		t.Fatalf("query feeding failed: %v", err)
	}
	if loggedBy != "deleted_user" {
		t.Errorf("expected feedings logged_by='deleted_user', got %q", loggedBy)
	}

	err = db.QueryRow("SELECT logged_by FROM test_stools WHERE id = 's1'").Scan(&loggedBy)
	if err != nil {
		t.Fatalf("query stool failed: %v", err)
	}
	if loggedBy != "deleted_user" {
		t.Errorf("expected stools logged_by='deleted_user', got %q", loggedBy)
	}
}

func TestDeleteAccount_NilAnonymizationTables(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	// nil tables list should work fine (no anonymization needed)
	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount with nil tables failed: %v", err)
	}
}

func TestDeleteAccount_EmptyAnonymizationTables(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	// empty tables list should work fine
	err = DeleteAccount(db, "u1", []string{})
	if err != nil {
		t.Fatalf("DeleteAccount with empty tables failed: %v", err)
	}
}

func TestDeleteAccount_OtherUsersDataPreserved(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`CREATE TABLE test_data (
		id TEXT PRIMARY KEY,
		logged_by TEXT REFERENCES users(id),
		updated_by TEXT REFERENCES users(id)
	)`)
	if err != nil {
		t.Fatalf("create test_data failed: %v", err)
	}

	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test1')")
	if err != nil {
		t.Fatalf("insert user1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u2', 'g2', 'b@b.com', 'Test2')")
	if err != nil {
		t.Fatalf("insert user2 failed: %v", err)
	}

	_, err = db.Exec("INSERT INTO test_data (id, logged_by, updated_by) VALUES ('d1', 'u1', 'u1')")
	if err != nil {
		t.Fatalf("insert d1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO test_data (id, logged_by, updated_by) VALUES ('d2', 'u2', 'u2')")
	if err != nil {
		t.Fatalf("insert d2 failed: %v", err)
	}

	err = DeleteAccount(db, "u1", []string{"test_data"})
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	// u2's data should be unchanged
	var loggedBy string
	err = db.QueryRow("SELECT logged_by FROM test_data WHERE id = 'd2'").Scan(&loggedBy)
	if err != nil {
		t.Fatalf("query d2 failed: %v", err)
	}
	if loggedBy != "u2" {
		t.Errorf("expected u2's logged_by preserved as 'u2', got %q", loggedBy)
	}
}

func TestDeleteAccount_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteAccount(db, "u1", nil)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestDeleteAccount_MixedBabies_LastParentDeletedSharedPreserved(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test1')")
	if err != nil {
		t.Fatalf("insert user1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u2', 'g2', 'b@b.com', 'Test2')")
	if err != nil {
		t.Fatalf("insert user2 failed: %v", err)
	}

	// b1: only u1 (should be deleted)
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Solo Baby', 'male', '2025-01-01')")
	if err != nil {
		t.Fatalf("insert b1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert bp b1-u1 failed: %v", err)
	}

	// b2: u1 and u2 (should be preserved)
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b2', 'Shared Baby', 'female', '2025-06-01')")
	if err != nil {
		t.Fatalf("insert b2 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b2', 'u1')")
	if err != nil {
		t.Fatalf("insert bp b2-u1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b2', 'u2')")
	if err != nil {
		t.Fatalf("insert bp b2-u2 failed: %v", err)
	}

	err = DeleteAccount(db, "u1", nil)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	// b1 should be deleted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM babies WHERE id = 'b1'").Scan(&count)
	if err != nil {
		t.Fatalf("count b1 failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected solo baby b1 to be deleted, found %d", count)
	}

	// b2 should still exist
	err = db.QueryRow("SELECT COUNT(*) FROM babies WHERE id = 'b2'").Scan(&count)
	if err != nil {
		t.Fatalf("count b2 failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected shared baby b2 to still exist, found %d", count)
	}
}

func TestDeleteAccount_AnonymizeOnlyAffectsDeletedUser(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`CREATE TABLE test_logs (
		id TEXT PRIMARY KEY,
		logged_by TEXT REFERENCES users(id),
		updated_by TEXT REFERENCES users(id)
	)`)
	if err != nil {
		t.Fatalf("create test_logs failed: %v", err)
	}

	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test1')")
	if err != nil {
		t.Fatalf("insert user1 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u2', 'g2', 'b@b.com', 'Test2')")
	if err != nil {
		t.Fatalf("insert user2 failed: %v", err)
	}

	// Entry logged by u1 but updated by u2
	_, err = db.Exec("INSERT INTO test_logs (id, logged_by, updated_by) VALUES ('l1', 'u1', 'u2')")
	if err != nil {
		t.Fatalf("insert log failed: %v", err)
	}

	err = DeleteAccount(db, "u1", []string{"test_logs"})
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}

	var loggedBy, updatedBy string
	err = db.QueryRow("SELECT logged_by, updated_by FROM test_logs WHERE id = 'l1'").Scan(&loggedBy, &updatedBy)
	if err != nil {
		t.Fatalf("query log failed: %v", err)
	}
	if loggedBy != "deleted_user" {
		t.Errorf("expected logged_by='deleted_user', got %q", loggedBy)
	}
	if updatedBy != "u2" {
		t.Errorf("expected updated_by='u2' (preserved), got %q", updatedBy)
	}
}
