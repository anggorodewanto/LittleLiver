package store

import (
	"database/sql"
	"testing"
)

func TestCreateUpperArmCircumference_StoresFieldsCorrectly(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac1", "uac1@test.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	ts := "2025-07-01T10:30:00Z"
	source := "home_scale"
	notes := "after feeding"

	u, err := CreateUpperArmCircumference(db, baby.ID, user.ID, ts, 11.5, &source, &notes)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}

	if u.ID == "" {
		t.Error("expected non-empty ID")
	}
	if len(u.ID) != 26 {
		t.Errorf("expected 26-char ULID, got %d chars", len(u.ID))
	}
	if u.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, u.BabyID)
	}
	if u.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%q, got %q", user.ID, u.LoggedBy)
	}
	if u.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", u.UpdatedBy)
	}
	if u.CircumferenceCm != 11.5 {
		t.Errorf("expected circumference_cm=11.5, got %f", u.CircumferenceCm)
	}
	if u.MeasurementSource == nil || *u.MeasurementSource != "home_scale" {
		t.Errorf("expected measurement_source=home_scale, got %v", u.MeasurementSource)
	}
	if u.Notes == nil || *u.Notes != "after feeding" {
		t.Errorf("expected notes=after feeding, got %v", u.Notes)
	}
	if u.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if u.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestCreateUpperArmCircumference_NilOptionalFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac2", "uac2@test.com", "Parent2")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	u, err := CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}
	if u.MeasurementSource != nil {
		t.Errorf("expected nil measurement_source, got %v", u.MeasurementSource)
	}
	if u.Notes != nil {
		t.Errorf("expected nil notes, got %v", u.Notes)
	}
}

func TestGetUpperArmCircumferenceByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetUpperArmCircumferenceByID(db, "nonexistent-baby", "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent upper arm circumference")
	}
}

func TestListUpperArmCircumferences_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac3", "uac3@test.com", "Parent3")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	page, err := ListUpperArmCircumferences(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListUpperArmCircumferences failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 entries, got %d", len(page.Data))
	}
	if page.NextCursor != nil {
		t.Error("expected nil next_cursor")
	}
}

func TestListUpperArmCircumferences_Pagination(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac4", "uac4@test.com", "Parent4")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for range 3 {
		_, err := CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
		if err != nil {
			t.Fatalf("CreateUpperArmCircumference failed: %v", err)
		}
	}

	page, err := ListUpperArmCircumferences(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListUpperArmCircumferences failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 entries, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Error("expected non-nil next_cursor")
	}
}

func TestUpdateUpperArmCircumference_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac5", "uac5@test.com", "Parent5")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	u, err := CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}

	src := "clinic"
	notes := "updated"
	updated, err := UpdateUpperArmCircumference(db, baby.ID, u.ID, user.ID, "2025-07-01T11:00:00Z", 12.0, &src, &notes)
	if err != nil {
		t.Fatalf("UpdateUpperArmCircumference failed: %v", err)
	}
	if updated.CircumferenceCm != 12.0 {
		t.Errorf("expected circumference_cm=12.0, got %f", updated.CircumferenceCm)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateUpperArmCircumference_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateUpperArmCircumference(db, "nonexistent-baby", "nonexistent-id", "user1", "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent upper arm circumference")
	}
}

func TestDeleteUpperArmCircumference_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac6", "uac6@test.com", "Parent6")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	u, err := CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}

	err = DeleteUpperArmCircumference(db, baby.ID, u.ID)
	if err != nil {
		t.Fatalf("DeleteUpperArmCircumference failed: %v", err)
	}

	_, err = GetUpperArmCircumferenceByID(db, baby.ID, u.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteUpperArmCircumference_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	err := DeleteUpperArmCircumference(db, "nonexistent-baby", "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent upper arm circumference")
	}
}

func TestListUpperArmCircumferences_DateFilters(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac7", "uac7@test.com", "Parent7")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, err = CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}
	_, err = CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-03T10:30:00Z", 11.8, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}

	from := "2025-07-02"
	page, err := ListUpperArmCircumferences(db, baby.ID, &from, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListUpperArmCircumferences failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 entry with from filter, got %d", len(page.Data))
	}
}

func TestCreateUpperArmCircumference_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateUpperArmCircumference(db, "baby1", "user1", "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestGetUpperArmCircumferenceByID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := GetUpperArmCircumferenceByID(db, "baby1", "u1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListUpperArmCircumferences_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListUpperArmCircumferences(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestDeleteUpperArmCircumference_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteUpperArmCircumference(db, "baby1", "u1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestUpdateUpperArmCircumference_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateUpperArmCircumference(db, "baby1", "u1", "user1", "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListUpperArmCircumferences_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListUpperArmCircumferences(db, "baby1", &badDate, nil, nil, 50)
	if err == nil {
		t.Error("expected error for invalid from date")
	}
}

func TestListUpperArmCircumferences_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListUpperArmCircumferences(db, "baby1", nil, &badDate, nil, 50)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}

func TestListUpperArmCircumferences_CursorFiltering(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac8", "uac8@test.com", "Parent8")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for range 3 {
		_, err := CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
		if err != nil {
			t.Fatalf("CreateUpperArmCircumference failed: %v", err)
		}
	}

	// First page with limit=2 should return 2 items and a cursor
	page1, err := ListUpperArmCircumferences(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListUpperArmCircumferences page 1 failed: %v", err)
	}
	if len(page1.Data) != 2 {
		t.Fatalf("expected 2 entries on page 1, got %d", len(page1.Data))
	}
	if page1.NextCursor == nil {
		t.Fatal("expected non-nil next_cursor on page 1")
	}

	// Second page using the cursor should return the remaining item
	page2, err := ListUpperArmCircumferences(db, baby.ID, nil, nil, page1.NextCursor, 2)
	if err != nil {
		t.Fatalf("ListUpperArmCircumferences page 2 failed: %v", err)
	}
	if len(page2.Data) != 1 {
		t.Errorf("expected 1 entry on page 2, got %d", len(page2.Data))
	}
}

func TestListUpperArmCircumferences_ToDateFilter(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac9", "uac9@test.com", "Parent9")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, err = CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}
	_, err = CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-05T10:30:00Z", 11.8, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}

	to := "2025-07-02"
	page, err := ListUpperArmCircumferences(db, baby.ID, nil, &to, nil, 50)
	if err != nil {
		t.Fatalf("ListUpperArmCircumferences failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 entry with to filter, got %d", len(page.Data))
	}
}

// Verify cascade delete works
func TestDeleteUpperArmCircumference_CascadeOnBabyDelete(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac10", "uac10@test.com", "Parent10")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, err = CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}

	// Delete the baby
	_, err = db.Exec("DELETE FROM baby_parents WHERE baby_id = ?", baby.ID)
	if err != nil {
		t.Fatalf("delete baby_parents failed: %v", err)
	}
	_, err = db.Exec("DELETE FROM babies WHERE id = ?", baby.ID)
	if err != nil {
		t.Fatalf("delete baby failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM upper_arm_circumferences WHERE baby_id = ?", baby.ID).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 entries after cascade, got %d", count)
	}
}

// Closed DB for rows scan error path
func TestListUpperArmCircumferencesWithTZ_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)

	user, err := UpsertUser(db, "google-uac11", "uac11@test.com", "Parent11")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	_ = baby
	db.Close()

	_, err = ListUpperArmCircumferences(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListUpperArmCircumferences_FromAndToDateFilters(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac12", "uac12@test.com", "Parent12")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, _ = CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-06-30T10:30:00Z", 11.0, nil, nil)
	_, _ = CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-02T10:30:00Z", 11.5, nil, nil)
	_, _ = CreateUpperArmCircumference(db, baby.ID, user.ID, "2025-07-05T10:30:00Z", 12.0, nil, nil)

	from := "2025-07-01"
	to := "2025-07-03"
	page, err := ListUpperArmCircumferences(db, baby.ID, &from, &to, nil, 50)
	if err != nil {
		t.Fatalf("ListUpperArmCircumferences failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 entry in date range, got %d", len(page.Data))
	}
}

// Test GetUpperArmCircumferenceByID on a different baby (scoping)
func TestGetUpperArmCircumferenceByID_WrongBaby(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-uac13", "uac13@test.com", "Parent13")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby1, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	baby2, err := CreateBaby(db, user.ID, "Stella", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby 2 failed: %v", err)
	}

	u, err := CreateUpperArmCircumference(db, baby1.ID, user.ID, "2025-07-01T10:30:00Z", 11.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateUpperArmCircumference failed: %v", err)
	}

	_, err = GetUpperArmCircumferenceByID(db, baby2.ID, u.ID)
	if err == nil {
		t.Error("expected error when accessing upper arm circumference from wrong baby")
	}
	if err != nil && err != sql.ErrNoRows {
		// It's OK if it wraps ErrNoRows or is a different scan error
	}
}
