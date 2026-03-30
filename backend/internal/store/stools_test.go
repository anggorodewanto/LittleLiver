package store

import (
	"database/sql"
	"testing"
)

func TestCreateStool_StoresFieldsCorrectly(t *testing.T) {
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
	colorLabel := "green"
	consistency := "soft"
	volume := "medium"
	notes := "normal stool"

	stool, err := CreateStool(db, baby.ID, user.ID, ts, 5, &colorLabel, &consistency, &volume, nil, &notes)
	if err != nil {
		t.Fatalf("CreateStool failed: %v", err)
	}

	if stool.ID == "" {
		t.Error("expected non-empty stool ID")
	}
	if len(stool.ID) != 26 {
		t.Errorf("expected 26-char ULID, got %d chars", len(stool.ID))
	}
	if stool.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, stool.BabyID)
	}
	if stool.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%q, got %q", user.ID, stool.LoggedBy)
	}
	if stool.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", stool.UpdatedBy)
	}
	if stool.ColorRating != 5 {
		t.Errorf("expected color_rating=5, got %d", stool.ColorRating)
	}
	if stool.ColorLabel == nil || *stool.ColorLabel != "green" {
		t.Errorf("expected color_label=green, got %v", stool.ColorLabel)
	}
	if stool.Consistency == nil || *stool.Consistency != "soft" {
		t.Errorf("expected consistency=soft, got %v", stool.Consistency)
	}
	if stool.VolumeEstimate == nil || *stool.VolumeEstimate != "medium" {
		t.Errorf("expected volume_estimate=medium, got %v", stool.VolumeEstimate)
	}
	if stool.Notes == nil || *stool.Notes != "normal stool" {
		t.Errorf("expected notes='normal stool', got %v", stool.Notes)
	}
	if stool.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if stool.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestCreateStool_NilOptionalFields(t *testing.T) {
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

	stool, err := CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 3, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool failed: %v", err)
	}

	if stool.ColorLabel != nil {
		t.Errorf("expected nil color_label, got %v", stool.ColorLabel)
	}
	if stool.Consistency != nil {
		t.Errorf("expected nil consistency, got %v", stool.Consistency)
	}
	if stool.VolumeEstimate != nil {
		t.Errorf("expected nil volume_estimate, got %v", stool.VolumeEstimate)
	}
	if stool.Notes != nil {
		t.Errorf("expected nil notes, got %v", stool.Notes)
	}
}

func TestCreateStool_InvalidColorRating(t *testing.T) {
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

	// color_rating = 0 should fail (CHECK constraint)
	_, err = CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 0, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for color_rating=0, got nil")
	}

	// color_rating = 8 should fail
	_, err = CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 8, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for color_rating=8, got nil")
	}
}

func TestCreateStool_ValidColorRatingBounds(t *testing.T) {
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

	// color_rating = 1 (min)
	s1, err := CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 1, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool with color_rating=1 failed: %v", err)
	}
	if s1.ColorRating != 1 {
		t.Errorf("expected color_rating=1, got %d", s1.ColorRating)
	}

	// color_rating = 7 (max)
	s7, err := CreateStool(db, baby.ID, user.ID, "2025-07-01T11:30:00Z", 7, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool with color_rating=7 failed: %v", err)
	}
	if s7.ColorRating != 7 {
		t.Errorf("expected color_rating=7, got %d", s7.ColorRating)
	}
}

func TestGetStoolByID_ReturnsSingleEntry(t *testing.T) {
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

	created, err := CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 4, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool failed: %v", err)
	}

	fetched, err := GetStoolByID(db, baby.ID, created.ID)
	if err != nil {
		t.Fatalf("GetStoolByID failed: %v", err)
	}

	if fetched.ID != created.ID {
		t.Errorf("expected ID=%q, got %q", created.ID, fetched.ID)
	}
	if fetched.ColorRating != 4 {
		t.Errorf("expected color_rating=4, got %d", fetched.ColorRating)
	}
}

func TestGetStoolByID_NotFound(t *testing.T) {
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

	_, err = GetStoolByID(db, baby.ID, "nonexistent")
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetStoolByID_WrongBaby(t *testing.T) {
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

	stool, err := CreateStool(db, baby1.ID, user.ID, "2025-07-01T10:30:00Z", 3, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool failed: %v", err)
	}

	_, err = GetStoolByID(db, baby2.ID, stool.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows for wrong baby, got %v", err)
	}
}

func TestListStools_CursorPagination(t *testing.T) {
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

	for i := 0; i < 5; i++ {
		_, err := CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 3, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("CreateStool %d failed: %v", i, err)
		}
	}

	page, err := ListStools(db, baby.ID, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("ListStools page 1 failed: %v", err)
	}
	if len(page.Data) != 3 {
		t.Fatalf("expected 3 items on page 1, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Fatal("expected non-nil next_cursor on page 1")
	}

	// Verify ORDER BY id DESC
	for i := 1; i < len(page.Data); i++ {
		if page.Data[i].ID >= page.Data[i-1].ID {
			t.Errorf("expected descending ID order: %s should be < %s", page.Data[i].ID, page.Data[i-1].ID)
		}
	}

	page2, err := ListStools(db, baby.ID, nil, nil, page.NextCursor, 3)
	if err != nil {
		t.Fatalf("ListStools page 2 failed: %v", err)
	}
	if len(page2.Data) != 2 {
		t.Fatalf("expected 2 items on page 2, got %d", len(page2.Data))
	}
	if page2.NextCursor != nil {
		t.Error("expected nil next_cursor on last page")
	}
}

func TestListStools_EmptyResult(t *testing.T) {
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

	page, err := ListStools(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListStools failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 stools, got %d", len(page.Data))
	}
	if page.NextCursor != nil {
		t.Error("expected nil next_cursor for empty result")
	}
}

func TestListStools_DateFiltering(t *testing.T) {
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

	_, err = CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 3, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool failed: %v", err)
	}
	_, err = CreateStool(db, baby.ID, user.ID, "2025-07-02T10:30:00Z", 4, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool failed: %v", err)
	}
	_, err = CreateStool(db, baby.ID, user.ID, "2025-07-03T10:30:00Z", 5, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool failed: %v", err)
	}

	from := "2025-07-01"
	to := "2025-07-02"
	page, err := ListStools(db, baby.ID, &from, &to, nil, 50)
	if err != nil {
		t.Fatalf("ListStools failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 stools in date range, got %d", len(page.Data))
	}
}

func TestUpdateStool_SetsUpdatedAt(t *testing.T) {
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

	created, err := CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 3, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool failed: %v", err)
	}

	newLabel := "brown"
	newNotes := "updated notes"
	updated, err := UpdateStool(db, baby.ID, created.ID, user.ID, "2025-07-01T11:00:00Z", 6, &newLabel, nil, nil, nil, &newNotes)
	if err != nil {
		t.Fatalf("UpdateStool failed: %v", err)
	}

	if updated.ColorRating != 6 {
		t.Errorf("expected color_rating=6, got %d", updated.ColorRating)
	}
	if updated.ColorLabel == nil || *updated.ColorLabel != "brown" {
		t.Errorf("expected color_label=brown, got %v", updated.ColorLabel)
	}
	if updated.Notes == nil || *updated.Notes != "updated notes" {
		t.Errorf("expected notes='updated notes', got %v", updated.Notes)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}
	if updated.UpdatedAt.Before(created.UpdatedAt) {
		t.Errorf("expected updated_at >= original, got %v < %v", updated.UpdatedAt, created.UpdatedAt)
	}
}

func TestUpdateStool_NotFound(t *testing.T) {
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

	_, err = UpdateStool(db, baby.ID, "nonexistent", user.ID, "2025-07-01T10:30:00Z", 3, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent stool, got nil")
	}
}

func TestDeleteStool_RemovesEntry(t *testing.T) {
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

	stool, err := CreateStool(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 3, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool failed: %v", err)
	}

	err = DeleteStool(db, baby.ID, stool.ID)
	if err != nil {
		t.Fatalf("DeleteStool failed: %v", err)
	}

	_, err = GetStoolByID(db, baby.ID, stool.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteStool_NotFound(t *testing.T) {
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

	err = DeleteStool(db, baby.ID, "nonexistent")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent stool, got nil")
	}
}

func TestStoolsTable_HasBabyTimestampIndex(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	var name string
	err := db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='index' AND name='idx_stools_baby_timestamp'",
	).Scan(&name)
	if err != nil {
		t.Fatalf("expected idx_stools_baby_timestamp index to exist, got error: %v", err)
	}
}

func TestStoolsTable_Columns(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	expected := []string{
		"id", "baby_id", "logged_by", "updated_by", "timestamp",
		"color_rating", "color_label", "consistency", "volume_estimate",
		"volume_ml", "photo_keys", "notes", "created_at", "updated_at",
	}
	assertColumns(t, db, "stools", expected)
}

func TestCreateStool_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateStool(db, "b1", "u1", "2025-07-01T10:30:00Z", 3, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestListStools_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListStools(db, "b1", nil, nil, nil, 50)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestGetStoolByID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := GetStoolByID(db, "b1", "s1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestUpdateStool_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateStool(db, "b1", "s1", "u1", "2025-07-01T10:30:00Z", 3, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestDeleteStool_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteStool(db, "b1", "s1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}
