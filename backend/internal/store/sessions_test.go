package store

import (
	"database/sql"
	"testing"
	"time"
)

func TestParseTime_Unparseable(t *testing.T) {
	t.Parallel()

	_, err := ParseTime("not-a-date")
	if err == nil {
		t.Fatal("expected error for unparseable time, got nil")
	}
}

func TestParseTime_ValidFormats(t *testing.T) {
	t.Parallel()

	cases := []string{
		"2025-01-15 10:30:00",
		"2025-01-15T10:30:00Z",
		"2025-01-15T10:30:00",
	}
	for _, tc := range cases {
		_, err := ParseTime(tc)
		if err != nil {
			t.Errorf("ParseTime(%q) returned error: %v", tc, err)
		}
	}
}

func TestGenerateToken_ReturnsHexString(t *testing.T) {
	t.Parallel()

	tok, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken failed: %v", err)
	}
	// 32 bytes hex-encoded = 64 characters
	if len(tok) != 64 {
		t.Errorf("expected 64-char hex token, got %d chars", len(tok))
	}
}

func TestCreateSession_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateSession(db, "u1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestGetSessionByID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := GetSessionByID(db, "some-id")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestDeleteSession_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteSession(db, "some-id")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestGetSessionByID_UnparseableExpiresAt(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	// Insert a session with an expires_at that will pass the > comparison
	// (starts with a high year) but can't be parsed by parseTime
	_, err = db.Exec(
		"INSERT INTO sessions (id, user_id, token, expires_at, created_at) VALUES ('s1', 'u1', 'tok1', '9999-99-99 99:99:99', '2025-01-01 00:00:00')",
	)
	if err != nil {
		t.Fatalf("insert session failed: %v", err)
	}

	_, err = GetSessionByID(db, "s1")
	if err == nil {
		t.Fatal("expected error for unparseable expires_at, got nil")
	}
}

func TestGetSessionByID_UnparseableCreatedAt(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	// expires_at is valid and future, but created_at is unparseable
	_, err = db.Exec(
		"INSERT INTO sessions (id, user_id, token, expires_at, created_at) VALUES ('s1', 'u1', 'tok1', '2099-01-01 00:00:00', 'not-a-date')",
	)
	if err != nil {
		t.Fatalf("insert session failed: %v", err)
	}

	_, err = GetSessionByID(db, "s1")
	if err == nil {
		t.Fatal("expected error for unparseable created_at, got nil")
	}
}

func TestExtendSession_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := ExtendSession(db, "some-id")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestCreateSession_InvalidUser(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// No such user - should fail due to FK constraint
	_, err := CreateSession(db, "nonexistent-user")
	if err == nil {
		t.Fatal("expected error for nonexistent user, got nil")
	}
}

func TestDeleteSession_NonexistentID(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Deleting nonexistent session should not error (no rows affected is OK)
	err := DeleteSession(db, "nonexistent")
	if err != nil {
		t.Fatalf("DeleteSession on nonexistent ID should not error, got: %v", err)
	}
}

func TestExtendSession_NonexistentID(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Extending nonexistent session should not error (no rows affected is OK)
	err := ExtendSession(db, "nonexistent")
	if err != nil {
		t.Fatalf("ExtendSession on nonexistent ID should not error, got: %v", err)
	}
}

func TestCreateSession(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Insert a user first
	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	sess, err := CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if sess.UserID != "u1" {
		t.Errorf("expected user_id=u1, got %q", sess.UserID)
	}
	if sess.ID == "" {
		t.Error("expected non-empty session ID")
	}
	if sess.Token == "" {
		t.Error("expected non-empty session token")
	}
	if sess.ExpiresAt.Before(time.Now().Add(29 * 24 * time.Hour)) {
		t.Error("expected expires_at to be ~30 days from now")
	}
}

func TestGetSessionByID(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	created, err := CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	got, err := GetSessionByID(db, created.ID)
	if err != nil {
		t.Fatalf("GetSessionByID failed: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("expected ID=%q, got %q", created.ID, got.ID)
	}
	if got.UserID != "u1" {
		t.Errorf("expected user_id=u1, got %q", got.UserID)
	}
	if got.Token != created.Token {
		t.Errorf("expected token=%q, got %q", created.Token, got.Token)
	}
}

func TestGetSessionByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetSessionByID(db, "nonexistent")
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetSessionByID_Expired(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	// Insert an expired session directly
	_, err = db.Exec(
		"INSERT INTO sessions (id, user_id, token, expires_at) VALUES ('s1', 'u1', 'tok1', ?)",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
	)
	if err != nil {
		t.Fatalf("insert expired session failed: %v", err)
	}

	_, err = GetSessionByID(db, "s1")
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows for expired session, got %v", err)
	}
}

func TestDeleteSession(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	created, err := CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = DeleteSession(db, created.ID)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	_, err = GetSessionByID(db, created.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestExtendSession(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	created, err := CreateSession(db, "u1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Manually set expires_at to 1 day from now (simulating time passing)
	_, err = db.Exec("UPDATE sessions SET expires_at = ? WHERE id = ?",
		time.Now().Add(1*24*time.Hour).UTC().Format(time.DateTime), created.ID)
	if err != nil {
		t.Fatalf("manual update failed: %v", err)
	}

	err = ExtendSession(db, created.ID)
	if err != nil {
		t.Fatalf("ExtendSession failed: %v", err)
	}

	got, err := GetSessionByID(db, created.ID)
	if err != nil {
		t.Fatalf("GetSessionByID after extend failed: %v", err)
	}

	if got.ExpiresAt.Before(time.Now().Add(29 * 24 * time.Hour)) {
		t.Error("expected expires_at to be extended to ~30 days from now")
	}
}
