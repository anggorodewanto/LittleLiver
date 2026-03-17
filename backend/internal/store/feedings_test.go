package store

import (
	"database/sql"
	"math"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

func TestCreateFeeding_StoresFieldsCorrectly(t *testing.T) {
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
	volMl := 120.0
	calDensity := 20.0
	durMin := 15
	notes := "tolerated well"

	feeding, err := CreateFeeding(db, baby.ID, user.ID, ts, "breast_milk", &volMl, &calDensity, &durMin, &notes, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	if feeding.ID == "" {
		t.Error("expected non-empty feeding ID")
	}
	if len(feeding.ID) != 26 {
		t.Errorf("expected 26-char ULID, got %d chars", len(feeding.ID))
	}
	if feeding.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, feeding.BabyID)
	}
	if feeding.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%q, got %q", user.ID, feeding.LoggedBy)
	}
	if feeding.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", feeding.UpdatedBy)
	}
	if feeding.FeedType != "breast_milk" {
		t.Errorf("expected feed_type=breast_milk, got %q", feeding.FeedType)
	}
	if feeding.VolumeMl == nil || *feeding.VolumeMl != 120.0 {
		t.Errorf("expected volume_ml=120, got %v", feeding.VolumeMl)
	}
	if feeding.CalDensity == nil || *feeding.CalDensity != 20.0 {
		t.Errorf("expected cal_density=20, got %v", feeding.CalDensity)
	}
	if feeding.DurationMin == nil || *feeding.DurationMin != 15 {
		t.Errorf("expected duration_min=15, got %v", feeding.DurationMin)
	}
	if feeding.Notes == nil || *feeding.Notes != "tolerated well" {
		t.Errorf("expected notes='tolerated well', got %v", feeding.Notes)
	}
	if feeding.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if feeding.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestCreateFeeding_NilOptionalFields(t *testing.T) {
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

	feeding, err := CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	if feeding.VolumeMl != nil {
		t.Errorf("expected nil volume_ml, got %v", feeding.VolumeMl)
	}
	if feeding.CalDensity != nil {
		t.Errorf("expected nil cal_density, got %v", feeding.CalDensity)
	}
	if feeding.DurationMin != nil {
		t.Errorf("expected nil duration_min, got %v", feeding.DurationMin)
	}
	if feeding.Notes != nil {
		t.Errorf("expected nil notes, got %v", feeding.Notes)
	}
}

func TestListFeedings_CursorPagination(t *testing.T) {
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

	// Create 5 feedings
	var ids []string
	for i := 0; i < 5; i++ {
		f, err := CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
		if err != nil {
			t.Fatalf("CreateFeeding %d failed: %v", i, err)
		}
		ids = append(ids, f.ID)
	}

	// First page with limit 3, no cursor
	page, err := ListFeedings(db, baby.ID, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("ListFeedings page 1 failed: %v", err)
	}
	if len(page.Data) != 3 {
		t.Fatalf("expected 3 items on page 1, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Fatal("expected non-nil next_cursor on page 1")
	}

	// Verify ORDER BY id DESC within the page
	for i := 1; i < len(page.Data); i++ {
		if page.Data[i].ID >= page.Data[i-1].ID {
			t.Errorf("expected descending ID order: %s should be < %s", page.Data[i].ID, page.Data[i-1].ID)
		}
	}

	// Second page using cursor
	page2, err := ListFeedings(db, baby.ID, nil, nil, page.NextCursor, 3)
	if err != nil {
		t.Fatalf("ListFeedings page 2 failed: %v", err)
	}
	if len(page2.Data) != 2 {
		t.Fatalf("expected 2 items on page 2, got %d", len(page2.Data))
	}
	if page2.NextCursor != nil {
		t.Error("expected nil next_cursor on last page")
	}

	// Verify no overlap between pages
	page1IDs := make(map[string]bool)
	for _, f := range page.Data {
		page1IDs[f.ID] = true
	}
	for _, f := range page2.Data {
		if page1IDs[f.ID] {
			t.Errorf("feeding %s appears on both pages", f.ID)
		}
	}

	// Verify page 2 IDs are all less than page 1 IDs (continuation)
	lastPage1ID := page.Data[len(page.Data)-1].ID
	for _, f := range page2.Data {
		if f.ID >= lastPage1ID {
			t.Errorf("page 2 item %s should be < last page 1 item %s", f.ID, lastPage1ID)
		}
	}
}

func TestListFeedings_DateFiltering(t *testing.T) {
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

	// Feedings on different dates
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-02T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-03T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	// Filter from 2025-07-01 to 2025-07-02 with UTC timezone
	loc := time.UTC
	from := "2025-07-01"
	to := "2025-07-02"
	page, err := ListFeedingsWithTZ(db, baby.ID, &from, &to, nil, 50, loc)
	if err != nil {
		t.Fatalf("ListFeedingsWithTZ failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 feedings in date range, got %d", len(page.Data))
	}
}

func TestListFeedings_DateFilteringWithTimezone(t *testing.T) {
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

	// A feeding at 2025-07-02 03:00 UTC = 2025-07-01 23:00 EDT (America/New_York is UTC-4 in summer)
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-02T03:00:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	// In UTC, this is July 2; in America/New_York, this is July 1
	loc, _ := time.LoadLocation("America/New_York")
	from := "2025-07-01"
	to := "2025-07-01"
	page, err := ListFeedingsWithTZ(db, baby.ID, &from, &to, nil, 50, loc)
	if err != nil {
		t.Fatalf("ListFeedingsWithTZ failed: %v", err)
	}
	// In New York timezone, 03:00 UTC is 23:00 EDT on July 1, so it should be included
	if len(page.Data) != 1 {
		t.Errorf("expected 1 feeding in NY timezone date range, got %d", len(page.Data))
	}
}

func TestGetFeedingByID_ReturnsSingleEntry(t *testing.T) {
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

	vol := 120.0
	created, err := CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "breast_milk", &vol, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	fetched, err := GetFeedingByID(db, baby.ID, created.ID)
	if err != nil {
		t.Fatalf("GetFeedingByID failed: %v", err)
	}

	if fetched.ID != created.ID {
		t.Errorf("expected ID=%q, got %q", created.ID, fetched.ID)
	}
	if fetched.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, fetched.BabyID)
	}
	if fetched.FeedType != "breast_milk" {
		t.Errorf("expected feed_type=breast_milk, got %q", fetched.FeedType)
	}
}

func TestGetFeedingByID_NotFound(t *testing.T) {
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

	_, err = GetFeedingByID(db, baby.ID, "nonexistent")
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetFeedingByID_WrongBaby(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby1, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	baby2, err := CreateBaby(db, user.ID, "Kai", "male", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	feeding, err := CreateFeeding(db, baby1.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	// Try to get feeding with wrong baby ID
	_, err = GetFeedingByID(db, baby2.ID, feeding.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows for wrong baby, got %v", err)
	}
}

func TestUpdateFeeding_SetsUpdatedAt(t *testing.T) {
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

	created, err := CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	originalUpdatedAt := created.UpdatedAt

	// Update with new values
	newVol := 150.0
	newNotes := "updated notes"
	updated, err := UpdateFeeding(db, baby.ID, created.ID, user.ID, "2025-07-01T11:00:00Z", "breast_milk", &newVol, nil, nil, &newNotes, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("UpdateFeeding failed: %v", err)
	}

	if updated.FeedType != "breast_milk" {
		t.Errorf("expected feed_type=breast_milk, got %q", updated.FeedType)
	}
	if updated.VolumeMl == nil || *updated.VolumeMl != 150.0 {
		t.Errorf("expected volume_ml=150, got %v", updated.VolumeMl)
	}
	if updated.Notes == nil || *updated.Notes != "updated notes" {
		t.Errorf("expected notes='updated notes', got %v", updated.Notes)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}

	// updated_at should be >= original (SQLite CURRENT_TIMESTAMP granularity is seconds)
	if updated.UpdatedAt.Before(originalUpdatedAt) {
		t.Errorf("expected updated_at >= original, got %v < %v", updated.UpdatedAt, originalUpdatedAt)
	}
}

func TestUpdateFeeding_NotFound(t *testing.T) {
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

	_, err = UpdateFeeding(db, baby.ID, "nonexistent", user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err == nil {
		t.Fatal("expected error for nonexistent feeding, got nil")
	}
}

func TestDeleteFeeding_RemovesEntry(t *testing.T) {
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

	feeding, err := CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	err = DeleteFeeding(db, baby.ID, feeding.ID)
	if err != nil {
		t.Fatalf("DeleteFeeding failed: %v", err)
	}

	_, err = GetFeedingByID(db, baby.ID, feeding.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteFeeding_NotFound(t *testing.T) {
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

	err = DeleteFeeding(db, baby.ID, "nonexistent")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent feeding, got nil")
	}
}

func TestFeedingsTable_HasBabyTimestampIndex(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	var name string
	err := db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='index' AND name='idx_feedings_baby_timestamp'",
	).Scan(&name)
	if err != nil {
		t.Fatalf("expected idx_feedings_baby_timestamp index to exist, got error: %v", err)
	}
	if name != "idx_feedings_baby_timestamp" {
		t.Errorf("expected index name=idx_feedings_baby_timestamp, got %q", name)
	}
}

func TestFeedingsTable_Columns(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	expected := []string{
		"id", "baby_id", "logged_by", "updated_by", "timestamp",
		"feed_type", "volume_ml", "cal_density", "calories",
		"used_default_cal", "duration_min", "notes", "created_at", "updated_at",
	}
	assertColumns(t, db, "feedings", expected)
}

func TestCreateFeeding_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateFeeding(db, "b1", "u1", "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestListFeedings_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListFeedings(db, "b1", nil, nil, nil, 50)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestUpdateFeeding_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateFeeding(db, "b1", "f1", "u1", "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestDeleteFeeding_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteFeeding(db, "b1", "f1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestGetFeedingByID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := GetFeedingByID(db, "b1", "f1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestListFeedings_WithFromOnly(t *testing.T) {
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

	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-03T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	from := "2025-07-02"
	page, err := ListFeedings(db, baby.ID, &from, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListFeedings failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 feeding from July 2 onward, got %d", len(page.Data))
	}
}

func TestListFeedings_WithToOnly(t *testing.T) {
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

	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-03T10:30:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	to := "2025-07-02"
	page, err := ListFeedings(db, baby.ID, nil, &to, nil, 50)
	if err != nil {
		t.Fatalf("ListFeedings failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 feeding through July 2, got %d", len(page.Data))
	}
}

func TestListFeedings_EmptyResult(t *testing.T) {
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

	page, err := ListFeedings(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListFeedings failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 feedings, got %d", len(page.Data))
	}
	if page.NextCursor != nil {
		t.Error("expected nil next_cursor for empty result")
	}
}

// --- Calorie calculation integration tests ---

func TestCreateFeeding_FormulaWithCalDensity_CalculatesCalories(t *testing.T) {
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

	vol := 120.0
	calDen := 24.0
	feeding, err := CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", &vol, &calDen, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	expected := 120.0 * (24.0 / model.MlPerOz)
	if feeding.Calories == nil {
		t.Fatal("expected non-nil calories")
	}
	if math.Abs(*feeding.Calories-expected) > 0.01 {
		t.Errorf("expected calories ~%.2f, got %.2f", expected, *feeding.Calories)
	}
	if feeding.UsedDefaultCal {
		t.Error("expected used_default_cal=false")
	}
}

func TestCreateFeeding_BreastMilkNoCalDensity_DefaultsTo20(t *testing.T) {
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

	vol := 100.0
	feeding, err := CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "breast_milk", &vol, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	expected := 100.0 * (20.0 / model.MlPerOz)
	if feeding.Calories == nil {
		t.Fatal("expected non-nil calories")
	}
	if math.Abs(*feeding.Calories-expected) > 0.01 {
		t.Errorf("expected calories ~%.2f, got %.2f", expected, *feeding.Calories)
	}
	if feeding.UsedDefaultCal {
		t.Error("expected used_default_cal=false for breast_milk with volume")
	}
	if feeding.CalDensity == nil || *feeding.CalDensity != 20.0 {
		t.Errorf("expected cal_density=20.0, got %v", feeding.CalDensity)
	}
}

func TestCreateFeeding_BreastDirect_UsesDefaultCalPerFeed(t *testing.T) {
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

	feeding, err := CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "breast_milk", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	if feeding.Calories == nil {
		t.Fatal("expected non-nil calories for breast-direct")
	}
	if *feeding.Calories != model.DefaultCalPerFeed {
		t.Errorf("expected calories=%.2f, got %.2f", model.DefaultCalPerFeed, *feeding.Calories)
	}
	if !feeding.UsedDefaultCal {
		t.Error("expected used_default_cal=true for breast-direct")
	}
}

func TestCreateFeeding_BreastDirectWithCalDensity_ReturnsError(t *testing.T) {
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

	calDen := 24.0
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "breast_milk", nil, &calDen, nil, nil, model.DefaultCalPerFeed)
	if err == nil {
		t.Fatal("expected error for breast-direct with cal_density, got nil")
	}
}

func TestUpdateFeeding_RecalculatesCalories(t *testing.T) {
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

	vol := 120.0
	calDen := 20.0
	feeding, err := CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "formula", &vol, &calDen, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding failed: %v", err)
	}

	// Update with different cal_density
	newCalDen := 24.0
	updated, err := UpdateFeeding(db, baby.ID, feeding.ID, user.ID, "2025-07-01T10:30:00Z", "formula", &vol, &newCalDen, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("UpdateFeeding failed: %v", err)
	}

	expected := 120.0 * (24.0 / model.MlPerOz)
	if updated.Calories == nil {
		t.Fatal("expected non-nil calories after update")
	}
	if math.Abs(*updated.Calories-expected) > 0.01 {
		t.Errorf("expected calories ~%.2f, got %.2f", expected, *updated.Calories)
	}
}

func TestRecalculateFeedingCalories_UpdatesAffectedEntries(t *testing.T) {
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

	// Create 2 breast-direct feedings (used_default_cal=true)
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "breast_milk", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding 1 failed: %v", err)
	}
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-01T14:30:00Z", "breast_milk", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding 2 failed: %v", err)
	}

	// Create 1 formula feeding (used_default_cal=false) — should NOT be affected
	vol := 120.0
	calDen := 24.0
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-01T18:30:00Z", "formula", &vol, &calDen, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding 3 failed: %v", err)
	}

	newDefault := 80.0
	count, err := RecalculateFeedingCalories(db, baby.ID, newDefault)
	if err != nil {
		t.Fatalf("RecalculateFeedingCalories failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 recalculated entries, got %d", count)
	}

	// Verify the breast-direct feedings were updated
	page, err := ListFeedings(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListFeedings failed: %v", err)
	}
	for _, f := range page.Data {
		if f.UsedDefaultCal {
			if f.Calories == nil || *f.Calories != 80.0 {
				t.Errorf("expected calories=80.0 for recalculated breast-direct, got %v", f.Calories)
			}
		}
	}
}
