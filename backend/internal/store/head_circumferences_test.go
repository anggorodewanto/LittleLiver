package store

import (
	"database/sql"
	"testing"
)

func TestCreateHeadCircumference_StoresFieldsCorrectly(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc1", "hc1@test.com", "Parent")
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

	hc, err := CreateHeadCircumference(db, baby.ID, user.ID, ts, 35.5, &source, &notes)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}

	if hc.ID == "" {
		t.Error("expected non-empty head circumference ID")
	}
	if len(hc.ID) != 26 {
		t.Errorf("expected 26-char ULID, got %d chars", len(hc.ID))
	}
	if hc.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, hc.BabyID)
	}
	if hc.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%q, got %q", user.ID, hc.LoggedBy)
	}
	if hc.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", hc.UpdatedBy)
	}
	if hc.CircumferenceCm != 35.5 {
		t.Errorf("expected circumference_cm=35.5, got %f", hc.CircumferenceCm)
	}
	if hc.MeasurementSource == nil || *hc.MeasurementSource != "home_scale" {
		t.Errorf("expected measurement_source=home_scale, got %v", hc.MeasurementSource)
	}
	if hc.Notes == nil || *hc.Notes != "after feeding" {
		t.Errorf("expected notes=after feeding, got %v", hc.Notes)
	}
	if hc.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if hc.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestCreateHeadCircumference_NilOptionalFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc2", "hc2@test.com", "Parent2")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	hc, err := CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}
	if hc.MeasurementSource != nil {
		t.Errorf("expected nil measurement_source, got %v", hc.MeasurementSource)
	}
	if hc.Notes != nil {
		t.Errorf("expected nil notes, got %v", hc.Notes)
	}
}

func TestGetHeadCircumferenceByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetHeadCircumferenceByID(db, "nonexistent-baby", "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent head circumference")
	}
}

func TestListHeadCircumferences_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc3", "hc3@test.com", "Parent3")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	page, err := ListHeadCircumferences(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListHeadCircumferences failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 head circumferences, got %d", len(page.Data))
	}
	if page.NextCursor != nil {
		t.Error("expected nil next_cursor")
	}
}

func TestListHeadCircumferences_Pagination(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc4", "hc4@test.com", "Parent4")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for range 3 {
		_, err := CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
		if err != nil {
			t.Fatalf("CreateHeadCircumference failed: %v", err)
		}
	}

	page, err := ListHeadCircumferences(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListHeadCircumferences failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 head circumferences, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Error("expected non-nil next_cursor")
	}
}

func TestUpdateHeadCircumference_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc5", "hc5@test.com", "Parent5")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	hc, err := CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}

	src := "clinic"
	notes := "updated"
	updated, err := UpdateHeadCircumference(db, baby.ID, hc.ID, user.ID, "2025-07-01T11:00:00Z", 36.0, &src, &notes)
	if err != nil {
		t.Fatalf("UpdateHeadCircumference failed: %v", err)
	}
	if updated.CircumferenceCm != 36.0 {
		t.Errorf("expected circumference_cm=36.0, got %f", updated.CircumferenceCm)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateHeadCircumference_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateHeadCircumference(db, "nonexistent-baby", "nonexistent-id", "user1", "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent head circumference")
	}
}

func TestDeleteHeadCircumference_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc6", "hc6@test.com", "Parent6")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	hc, err := CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}

	err = DeleteHeadCircumference(db, baby.ID, hc.ID)
	if err != nil {
		t.Fatalf("DeleteHeadCircumference failed: %v", err)
	}

	_, err = GetHeadCircumferenceByID(db, baby.ID, hc.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteHeadCircumference_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	err := DeleteHeadCircumference(db, "nonexistent-baby", "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent head circumference")
	}
}

func TestListHeadCircumferences_DateFilters(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc7", "hc7@test.com", "Parent7")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, err = CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}
	_, err = CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-03T10:30:00Z", 35.8, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}

	from := "2025-07-02"
	page, err := ListHeadCircumferences(db, baby.ID, &from, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListHeadCircumferences failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 head circumference with from filter, got %d", len(page.Data))
	}
}

func TestCreateHeadCircumference_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateHeadCircumference(db, "baby1", "user1", "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestGetHeadCircumferenceByID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := GetHeadCircumferenceByID(db, "baby1", "hc1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListHeadCircumferences_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListHeadCircumferences(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestDeleteHeadCircumference_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteHeadCircumference(db, "baby1", "hc1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestUpdateHeadCircumference_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateHeadCircumference(db, "baby1", "hc1", "user1", "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListHeadCircumferences_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListHeadCircumferences(db, "baby1", &badDate, nil, nil, 50)
	if err == nil {
		t.Error("expected error for invalid from date")
	}
}

func TestListHeadCircumferences_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListHeadCircumferences(db, "baby1", nil, &badDate, nil, 50)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}

func TestListHeadCircumferences_CursorFiltering(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc8", "hc8@test.com", "Parent8")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for range 3 {
		_, err := CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
		if err != nil {
			t.Fatalf("CreateHeadCircumference failed: %v", err)
		}
	}

	// First page with limit=2 should return 2 items and a cursor
	page1, err := ListHeadCircumferences(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListHeadCircumferences page 1 failed: %v", err)
	}
	if len(page1.Data) != 2 {
		t.Fatalf("expected 2 head circumferences on page 1, got %d", len(page1.Data))
	}
	if page1.NextCursor == nil {
		t.Fatal("expected non-nil next_cursor on page 1")
	}

	// Second page using the cursor should return the remaining item
	page2, err := ListHeadCircumferences(db, baby.ID, nil, nil, page1.NextCursor, 2)
	if err != nil {
		t.Fatalf("ListHeadCircumferences page 2 failed: %v", err)
	}
	if len(page2.Data) != 1 {
		t.Errorf("expected 1 head circumference on page 2, got %d", len(page2.Data))
	}
}

func TestListHeadCircumferences_ToDateFilter(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc9", "hc9@test.com", "Parent9")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, err = CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}
	_, err = CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-05T10:30:00Z", 35.8, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}

	to := "2025-07-02"
	page, err := ListHeadCircumferences(db, baby.ID, nil, &to, nil, 50)
	if err != nil {
		t.Fatalf("ListHeadCircumferences failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 head circumference with to filter, got %d", len(page.Data))
	}
}

// Verify cascade delete works
func TestDeleteHeadCircumference_CascadeOnBabyDelete(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc10", "hc10@test.com", "Parent10")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, err = CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
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
	err = db.QueryRow("SELECT COUNT(*) FROM head_circumferences WHERE baby_id = ?", baby.ID).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 head circumferences after cascade, got %d", count)
	}
}

// Closed DB for rows scan error path
func TestListHeadCircumferencesWithTZ_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)

	user, err := UpsertUser(db, "google-hc11", "hc11@test.com", "Parent11")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	_ = baby
	db.Close()

	_, err = ListHeadCircumferences(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListHeadCircumferences_FromAndToDateFilters(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc12", "hc12@test.com", "Parent12")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, _ = CreateHeadCircumference(db, baby.ID, user.ID, "2025-06-30T10:30:00Z", 34.5, nil, nil)
	_, _ = CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-02T10:30:00Z", 35.5, nil, nil)
	_, _ = CreateHeadCircumference(db, baby.ID, user.ID, "2025-07-05T10:30:00Z", 36.0, nil, nil)

	from := "2025-07-01"
	to := "2025-07-03"
	page, err := ListHeadCircumferences(db, baby.ID, &from, &to, nil, 50)
	if err != nil {
		t.Fatalf("ListHeadCircumferences failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 head circumference in date range, got %d", len(page.Data))
	}
}

// Test GetHeadCircumferenceByID on a different baby (scoping)
func TestGetHeadCircumferenceByID_WrongBaby(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-hc13", "hc13@test.com", "Parent13")
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

	hc, err := CreateHeadCircumference(db, baby1.ID, user.ID, "2025-07-01T10:30:00Z", 35.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeadCircumference failed: %v", err)
	}

	_, err = GetHeadCircumferenceByID(db, baby2.ID, hc.ID)
	if err == nil {
		t.Error("expected error when accessing head circumference from wrong baby")
	}
	if err != nil && err != sql.ErrNoRows {
		// It's OK if it wraps ErrNoRows or is a different scan error
	}
}
