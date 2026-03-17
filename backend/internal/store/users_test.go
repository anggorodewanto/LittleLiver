package store

import (
	"database/sql"
	"testing"
)

func TestUpsertUser_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpsertUser(db, "google123", "test@example.com", "Test User")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestUpsertUser_Insert(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google123", "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	if user.GoogleID != "google123" {
		t.Errorf("expected google_id=google123, got %q", user.GoogleID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email=test@example.com, got %q", user.Email)
	}
	if user.Name != "Test User" {
		t.Errorf("expected name=Test User, got %q", user.Name)
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
}

func TestUpsertUser_Update(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	first, err := UpsertUser(db, "google123", "old@example.com", "Old Name")
	if err != nil {
		t.Fatalf("first UpsertUser failed: %v", err)
	}

	second, err := UpsertUser(db, "google123", "new@example.com", "New Name")
	if err != nil {
		t.Fatalf("second UpsertUser failed: %v", err)
	}

	if second.ID != first.ID {
		t.Errorf("expected same ID on upsert, got first=%q second=%q", first.ID, second.ID)
	}
	if second.Email != "new@example.com" {
		t.Errorf("expected updated email=new@example.com, got %q", second.Email)
	}
	if second.Name != "New Name" {
		t.Errorf("expected updated name=New Name, got %q", second.Name)
	}
}

func TestGetUserByID_Found(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Insert a user
	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	user, err := GetUserByID(db, "u1")
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if user.ID != "u1" {
		t.Errorf("expected ID=u1, got %q", user.ID)
	}
	if user.Email != "a@b.com" {
		t.Errorf("expected email=a@b.com, got %q", user.Email)
	}
	if user.Name != "Test" {
		t.Errorf("expected name=Test, got %q", user.Name)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetUserByID(db, "nonexistent")
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetUserByID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := GetUserByID(db, "u1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestGetUserByID_WithTimezone(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name, timezone) VALUES ('u1', 'g1', 'a@b.com', 'Test', 'America/New_York')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	user, err := GetUserByID(db, "u1")
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if user.Timezone == nil || *user.Timezone != "America/New_York" {
		t.Errorf("expected timezone=America/New_York, got %v", user.Timezone)
	}
}

func TestUpdateUserTimezone(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	err = UpdateUserTimezone(db, "u1", "Europe/London")
	if err != nil {
		t.Fatalf("UpdateUserTimezone failed: %v", err)
	}

	user, err := GetUserByID(db, "u1")
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if user.Timezone == nil || *user.Timezone != "Europe/London" {
		t.Errorf("expected timezone=Europe/London, got %v", user.Timezone)
	}
}

func TestUpdateUserTimezone_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := UpdateUserTimezone(db, "u1", "Europe/London")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestGetBabiesByUserID_NoBabies(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	babies, err := GetBabiesByUserID(db, "u1")
	if err != nil {
		t.Fatalf("GetBabiesByUserID failed: %v", err)
	}
	if len(babies) != 0 {
		t.Errorf("expected 0 babies, got %d", len(babies))
	}
}

func TestGetBabiesByUserID_WithBabies(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Baby1', 'male', '2025-01-01')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b2', 'Baby2', 'female', '2025-06-01')")
	if err != nil {
		t.Fatalf("insert baby2 failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert baby_parents failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b2', 'u1')")
	if err != nil {
		t.Fatalf("insert baby_parents2 failed: %v", err)
	}

	babies, err := GetBabiesByUserID(db, "u1")
	if err != nil {
		t.Fatalf("GetBabiesByUserID failed: %v", err)
	}
	if len(babies) != 2 {
		t.Fatalf("expected 2 babies, got %d", len(babies))
	}
}

func TestGetBabiesByUserID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := GetBabiesByUserID(db, "u1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestGetBabiesByUserID_WithOptionalFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth, diagnosis_date, kasai_date, notes) VALUES ('b1', 'Baby1', 'male', '2025-01-01', '2025-01-15', '2025-02-01', 'some notes')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert baby_parents failed: %v", err)
	}

	babies, err := GetBabiesByUserID(db, "u1")
	if err != nil {
		t.Fatalf("GetBabiesByUserID failed: %v", err)
	}
	if len(babies) != 1 {
		t.Fatalf("expected 1 baby, got %d", len(babies))
	}
	b := babies[0]
	if b.DiagnosisDate == nil {
		t.Error("expected non-nil diagnosis_date")
	}
	if b.KasaiDate == nil {
		t.Error("expected non-nil kasai_date")
	}
	if b.Notes == nil || *b.Notes != "some notes" {
		t.Errorf("expected notes='some notes', got %v", b.Notes)
	}
}

func TestGetBabiesByUserID_UnparseableDOB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	// Insert baby with unparseable date_of_birth
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth) VALUES ('b1', 'Baby1', 'male', 'not-a-date')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert baby_parents failed: %v", err)
	}

	_, err = GetBabiesByUserID(db, "u1")
	if err == nil {
		t.Fatal("expected error for unparseable date_of_birth, got nil")
	}
}

func TestGetBabiesByUserID_UnparseableDiagnosisDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth, diagnosis_date) VALUES ('b1', 'Baby1', 'male', '2025-01-01', 'not-a-date')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert baby_parents failed: %v", err)
	}

	_, err = GetBabiesByUserID(db, "u1")
	if err == nil {
		t.Fatal("expected error for unparseable diagnosis_date, got nil")
	}
}

func TestGetBabiesByUserID_UnparseableKasaiDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth, kasai_date) VALUES ('b1', 'Baby1', 'male', '2025-01-01', 'not-a-date')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert baby_parents failed: %v", err)
	}

	_, err = GetBabiesByUserID(db, "u1")
	if err == nil {
		t.Fatal("expected error for unparseable kasai_date, got nil")
	}
}

func TestGetBabiesByUserID_UnparseableCreatedAt(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO babies (id, name, sex, date_of_birth, created_at) VALUES ('b1', 'Baby1', 'male', '2025-01-01', 'not-a-date')")
	if err != nil {
		t.Fatalf("insert baby failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES ('b1', 'u1')")
	if err != nil {
		t.Fatalf("insert baby_parents failed: %v", err)
	}

	_, err = GetBabiesByUserID(db, "u1")
	if err == nil {
		t.Fatal("expected error for unparseable created_at, got nil")
	}
}
