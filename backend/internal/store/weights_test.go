package store

import (
	"database/sql"
	"testing"
)

func TestCreateWeight_StoresFieldsCorrectly(t *testing.T) {
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

	ts := "2025-07-01T10:30:00Z"
	source := "home_scale"
	notes := "after feeding"

	w, err := CreateWeight(db, baby.ID, user.ID, ts, 4.25, &source, &notes)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}

	if w.ID == "" {
		t.Error("expected non-empty weight ID")
	}
	if len(w.ID) != 26 {
		t.Errorf("expected 26-char ULID, got %d chars", len(w.ID))
	}
	if w.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, w.BabyID)
	}
	if w.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%q, got %q", user.ID, w.LoggedBy)
	}
	if w.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", w.UpdatedBy)
	}
	if w.WeightKg != 4.25 {
		t.Errorf("expected weight_kg=4.25, got %f", w.WeightKg)
	}
	if w.MeasurementSource == nil || *w.MeasurementSource != "home_scale" {
		t.Errorf("expected measurement_source=home_scale, got %v", w.MeasurementSource)
	}
	if w.Notes == nil || *w.Notes != "after feeding" {
		t.Errorf("expected notes=after feeding, got %v", w.Notes)
	}
	if w.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if w.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestCreateWeight_NilOptionalFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google2", "b@b.com", "Parent2")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	w, err := CreateWeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 3.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}
	if w.MeasurementSource != nil {
		t.Errorf("expected nil measurement_source, got %v", w.MeasurementSource)
	}
	if w.Notes != nil {
		t.Errorf("expected nil notes, got %v", w.Notes)
	}
}

func TestGetWeightByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetWeightByID(db, "nonexistent-baby", "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent weight")
	}
}

func TestListWeights_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google3", "c@b.com", "Parent3")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	page, err := ListWeights(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListWeights failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 weights, got %d", len(page.Data))
	}
	if page.NextCursor != nil {
		t.Error("expected nil next_cursor")
	}
}

func TestListWeights_Pagination(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google4", "d@b.com", "Parent4")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := CreateWeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 4.0, nil, nil)
		if err != nil {
			t.Fatalf("CreateWeight failed: %v", err)
		}
	}

	page, err := ListWeights(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListWeights failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 weights, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Error("expected non-nil next_cursor")
	}
}

func TestUpdateWeight_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google5", "e@b.com", "Parent5")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	w, err := CreateWeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 4.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}

	src := "clinic"
	notes := "updated"
	updated, err := UpdateWeight(db, baby.ID, w.ID, user.ID, "2025-07-01T11:00:00Z", 4.5, &src, &notes)
	if err != nil {
		t.Fatalf("UpdateWeight failed: %v", err)
	}
	if updated.WeightKg != 4.5 {
		t.Errorf("expected weight_kg=4.5, got %f", updated.WeightKg)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateWeight_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateWeight(db, "nonexistent-baby", "nonexistent-id", "user1", "2025-07-01T10:30:00Z", 4.0, nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent weight")
	}
}

func TestDeleteWeight_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google6", "f@b.com", "Parent6")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	w, err := CreateWeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 4.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}

	err = DeleteWeight(db, baby.ID, w.ID)
	if err != nil {
		t.Fatalf("DeleteWeight failed: %v", err)
	}

	_, err = GetWeightByID(db, baby.ID, w.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteWeight_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	err := DeleteWeight(db, "nonexistent-baby", "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent weight")
	}
}

func TestListWeights_DateFilters(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google7", "g@b.com", "Parent7")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, err = CreateWeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 4.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}
	_, err = CreateWeight(db, baby.ID, user.ID, "2025-07-03T10:30:00Z", 4.1, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}

	from := "2025-07-02"
	page, err := ListWeights(db, baby.ID, &from, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListWeights failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 weight with from filter, got %d", len(page.Data))
	}
}

func TestCreateWeight_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateWeight(db, "baby1", "user1", "2025-07-01T10:30:00Z", 4.0, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestGetWeightByID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := GetWeightByID(db, "baby1", "w1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListWeights_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListWeights(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestDeleteWeight_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteWeight(db, "baby1", "w1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestUpdateWeight_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateWeight(db, "baby1", "w1", "user1", "2025-07-01T10:30:00Z", 4.0, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListWeights_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListWeights(db, "baby1", &badDate, nil, nil, 50)
	if err == nil {
		t.Error("expected error for invalid from date")
	}
}

func TestListWeights_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListWeights(db, "baby1", nil, &badDate, nil, 50)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}

func TestListWeights_CursorFiltering(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google8", "h@b.com", "Parent8")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := CreateWeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 4.0, nil, nil)
		if err != nil {
			t.Fatalf("CreateWeight failed: %v", err)
		}
	}

	// First page with limit=2 should return 2 items and a cursor
	page1, err := ListWeights(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListWeights page 1 failed: %v", err)
	}
	if len(page1.Data) != 2 {
		t.Fatalf("expected 2 weights on page 1, got %d", len(page1.Data))
	}
	if page1.NextCursor == nil {
		t.Fatal("expected non-nil next_cursor on page 1")
	}

	// Second page using the cursor should return the remaining item
	page2, err := ListWeights(db, baby.ID, nil, nil, page1.NextCursor, 2)
	if err != nil {
		t.Fatalf("ListWeights page 2 failed: %v", err)
	}
	if len(page2.Data) != 1 {
		t.Errorf("expected 1 weight on page 2, got %d", len(page2.Data))
	}
}

func TestListWeights_ToDateFilter(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google9", "i@b.com", "Parent9")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, err = CreateWeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 4.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}
	_, err = CreateWeight(db, baby.ID, user.ID, "2025-07-05T10:30:00Z", 4.1, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}

	to := "2025-07-02"
	page, err := ListWeights(db, baby.ID, nil, &to, nil, 50)
	if err != nil {
		t.Fatalf("ListWeights failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 weight with to filter, got %d", len(page.Data))
	}
}

// Verify cascade delete works
func TestDeleteWeight_CascadeOnBabyDelete(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google10", "j@b.com", "Parent10")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, err = CreateWeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 4.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
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
	err = db.QueryRow("SELECT COUNT(*) FROM weights WHERE baby_id = ?", baby.ID).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 weights after cascade, got %d", count)
	}
}

// Closed DB for rows scan error path
func TestListWeightsWithTZ_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)

	user, err := UpsertUser(db, "google11", "k@b.com", "Parent11")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	_ = baby
	db.Close()

	_, err = ListWeights(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListWeights_FromAndToDateFilters(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google12", "l@b.com", "Parent12")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, _ = CreateWeight(db, baby.ID, user.ID, "2025-06-30T10:30:00Z", 3.9, nil, nil)
	_, _ = CreateWeight(db, baby.ID, user.ID, "2025-07-02T10:30:00Z", 4.0, nil, nil)
	_, _ = CreateWeight(db, baby.ID, user.ID, "2025-07-05T10:30:00Z", 4.1, nil, nil)

	from := "2025-07-01"
	to := "2025-07-03"
	page, err := ListWeights(db, baby.ID, &from, &to, nil, 50)
	if err != nil {
		t.Fatalf("ListWeights failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 weight in date range, got %d", len(page.Data))
	}
}

// Test GetWeightByID on a different baby (scoping)
func TestGetWeightByID_WrongBaby(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google13", "m@b.com", "Parent13")
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

	w, err := CreateWeight(db, baby1.ID, user.ID, "2025-07-01T10:30:00Z", 4.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight failed: %v", err)
	}

	_, err = GetWeightByID(db, baby2.ID, w.ID)
	if err == nil {
		t.Error("expected error when accessing weight from wrong baby")
	}
	if err != nil && err != sql.ErrNoRows {
		// It's OK if it wraps ErrNoRows or is a different scan error
	}
}
