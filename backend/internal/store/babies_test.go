package store

import (
	"database/sql"
	"testing"
	"time"
)

func TestCreateBaby_ReturnsWithULIDAndLinks(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Create a user
	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	if baby.ID == "" {
		t.Error("expected non-empty baby ID (ULID)")
	}
	if len(baby.ID) != 26 {
		t.Errorf("expected 26-char ULID, got %d chars: %q", len(baby.ID), baby.ID)
	}
	if baby.Name != "Luna" {
		t.Errorf("expected name=Luna, got %q", baby.Name)
	}
	if baby.Sex != "female" {
		t.Errorf("expected sex=female, got %q", baby.Sex)
	}
	if baby.DateOfBirth.Format("2006-01-02") != "2025-06-15" {
		t.Errorf("expected dob=2025-06-15, got %q", baby.DateOfBirth.Format("2006-01-02"))
	}
	if baby.DefaultCalPerFeed != 67 {
		t.Errorf("expected default_cal_per_feed=67, got %f", baby.DefaultCalPerFeed)
	}

	// Verify the creator is linked as a parent
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM baby_parents WHERE baby_id = ? AND user_id = ?", baby.ID, user.ID).Scan(&count)
	if err != nil {
		t.Fatalf("query baby_parents failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected creator linked as parent, got count=%d", count)
	}
}

func TestCreateBaby_WithOptionalFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	diagDate := "2025-07-01"
	kasaiDate := "2025-07-15"
	calPerFeed := 80.0
	notes := "Recovering well"

	baby, err := CreateBaby(db, user.ID, "Kai", "male", "2025-06-15", &diagDate, &kasaiDate, &calPerFeed, &notes)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	if baby.DiagnosisDate == nil {
		t.Fatal("expected non-nil diagnosis_date")
	}
	if baby.DiagnosisDate.Format("2006-01-02") != "2025-07-01" {
		t.Errorf("expected diagnosis_date=2025-07-01, got %q", baby.DiagnosisDate.Format("2006-01-02"))
	}
	if baby.KasaiDate == nil {
		t.Fatal("expected non-nil kasai_date")
	}
	if baby.KasaiDate.Format("2006-01-02") != "2025-07-15" {
		t.Errorf("expected kasai_date=2025-07-15, got %q", baby.KasaiDate.Format("2006-01-02"))
	}
	if baby.DefaultCalPerFeed != 80.0 {
		t.Errorf("expected default_cal_per_feed=80, got %f", baby.DefaultCalPerFeed)
	}
	if baby.Notes == nil || *baby.Notes != "Recovering well" {
		t.Errorf("expected notes='Recovering well', got %v", baby.Notes)
	}
}

func TestCreateBaby_InvalidSex(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	_, err = CreateBaby(db, user.ID, "Baby", "other", "2025-06-15", nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid sex, got nil")
	}
}

func TestCreateBaby_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateBaby(db, "u1", "Baby", "female", "2025-06-15", nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestGetBabyByID_Found(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	created, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	baby, err := GetBabyByID(db, created.ID)
	if err != nil {
		t.Fatalf("GetBabyByID failed: %v", err)
	}

	if baby.ID != created.ID {
		t.Errorf("expected ID=%q, got %q", created.ID, baby.ID)
	}
	if baby.Name != "Luna" {
		t.Errorf("expected name=Luna, got %q", baby.Name)
	}
}

func TestGetBabyByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetBabyByID(db, "nonexistent")
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetBabyByID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := GetBabyByID(db, "b1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestIsParentOfBaby_True(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	linked, err := IsParentOfBaby(db, user.ID, baby.ID)
	if err != nil {
		t.Fatalf("IsParentOfBaby failed: %v", err)
	}
	if !linked {
		t.Error("expected user to be linked to baby")
	}
}

func TestIsParentOfBaby_False(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	other, err := UpsertUser(db, "google2", "b@b.com", "Other")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	linked, err := IsParentOfBaby(db, other.ID, baby.ID)
	if err != nil {
		t.Fatalf("IsParentOfBaby failed: %v", err)
	}
	if linked {
		t.Error("expected user NOT to be linked to baby")
	}
}

func TestIsParentOfBaby_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := IsParentOfBaby(db, "u1", "b1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestUpdateBaby_AllFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	diagDate := "2025-07-01"
	kasaiDate := "2025-07-15"
	calPerFeed := 80.0
	notes := "Updated notes"

	updated, err := UpdateBaby(db, baby.ID, "Kai", "male", "2025-06-20", &diagDate, &kasaiDate, &calPerFeed, &notes)
	if err != nil {
		t.Fatalf("UpdateBaby failed: %v", err)
	}

	if updated.Name != "Kai" {
		t.Errorf("expected name=Kai, got %q", updated.Name)
	}
	if updated.Sex != "male" {
		t.Errorf("expected sex=male, got %q", updated.Sex)
	}
	if updated.DateOfBirth.Format("2006-01-02") != "2025-06-20" {
		t.Errorf("expected dob=2025-06-20, got %q", updated.DateOfBirth.Format("2006-01-02"))
	}
	if updated.DiagnosisDate == nil || updated.DiagnosisDate.Format("2006-01-02") != "2025-07-01" {
		t.Errorf("expected diagnosis_date=2025-07-01, got %v", updated.DiagnosisDate)
	}
	if updated.KasaiDate == nil || updated.KasaiDate.Format("2006-01-02") != "2025-07-15" {
		t.Errorf("expected kasai_date=2025-07-15, got %v", updated.KasaiDate)
	}
	if updated.DefaultCalPerFeed != 80.0 {
		t.Errorf("expected default_cal_per_feed=80, got %f", updated.DefaultCalPerFeed)
	}
	if updated.Notes == nil || *updated.Notes != "Updated notes" {
		t.Errorf("expected notes='Updated notes', got %v", updated.Notes)
	}
}

func TestUpdateBaby_ClearOptionalFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	diagDate := "2025-07-01"
	notes := "Some notes"
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", &diagDate, nil, nil, &notes)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	// Update with nil optional fields to clear them
	updated, err := UpdateBaby(db, baby.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateBaby failed: %v", err)
	}

	if updated.DiagnosisDate != nil {
		t.Errorf("expected nil diagnosis_date, got %v", updated.DiagnosisDate)
	}
	if updated.Notes != nil {
		t.Errorf("expected nil notes, got %v", updated.Notes)
	}
}

func TestUpdateBaby_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateBaby(db, "nonexistent", "Name", "female", "2025-06-15", nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent baby, got nil")
	}
}

func TestUpdateBaby_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateBaby(db, "b1", "Name", "female", "2025-06-15", nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestUpdateBaby_NotesPreserved(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	notes := "Important clinical notes about the baby's recovery after Kasai procedure"
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, &notes)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	// Fetch and verify notes
	fetched, err := GetBabyByID(db, baby.ID)
	if err != nil {
		t.Fatalf("GetBabyByID failed: %v", err)
	}
	if fetched.Notes == nil || *fetched.Notes != notes {
		t.Errorf("expected notes=%q, got %v", notes, fetched.Notes)
	}
}

func TestGetBabiesByUserID_OnlyLinkedBabies(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user1, err := UpsertUser(db, "google1", "a@b.com", "Parent1")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	user2, err := UpsertUser(db, "google2", "b@b.com", "Parent2")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	_, err = CreateBaby(db, user1.ID, "Baby1", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby 1 failed: %v", err)
	}
	_, err = CreateBaby(db, user1.ID, "Baby2", "male", "2025-02-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby 2 failed: %v", err)
	}
	_, err = CreateBaby(db, user2.ID, "Baby3", "female", "2025-03-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby 3 failed: %v", err)
	}

	babies1, err := GetBabiesByUserID(db, user1.ID)
	if err != nil {
		t.Fatalf("GetBabiesByUserID for user1 failed: %v", err)
	}
	if len(babies1) != 2 {
		t.Errorf("expected 2 babies for user1, got %d", len(babies1))
	}

	babies2, err := GetBabiesByUserID(db, user2.ID)
	if err != nil {
		t.Fatalf("GetBabiesByUserID for user2 failed: %v", err)
	}
	if len(babies2) != 1 {
		t.Errorf("expected 1 baby for user2, got %d", len(babies2))
	}
}

func TestGetBabyByID_WithOptionalFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	diagDate := "2025-07-01"
	kasaiDate := "2025-07-15"
	calPerFeed := 80.0
	notes := "Notes"

	created, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", &diagDate, &kasaiDate, &calPerFeed, &notes)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	baby, err := GetBabyByID(db, created.ID)
	if err != nil {
		t.Fatalf("GetBabyByID failed: %v", err)
	}

	if baby.DiagnosisDate == nil {
		t.Error("expected non-nil diagnosis_date")
	}
	if baby.KasaiDate == nil {
		t.Error("expected non-nil kasai_date")
	}
	if baby.DefaultCalPerFeed != 80.0 {
		t.Errorf("expected default_cal_per_feed=80, got %f", baby.DefaultCalPerFeed)
	}
	if baby.Notes == nil || *baby.Notes != "Notes" {
		t.Errorf("expected notes='Notes', got %v", baby.Notes)
	}
	if baby.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	_ = time.Now() // reference time import
}
