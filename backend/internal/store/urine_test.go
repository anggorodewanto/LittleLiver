package store

import (
	"database/sql"
	"testing"
)

func TestCreateUrine_StoresFieldsCorrectly(t *testing.T) {
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
	color := "pale_yellow"
	notes := "normal output"

	urine, err := CreateUrine(db, baby.ID, user.ID, ts, &color, &notes)
	if err != nil {
		t.Fatalf("CreateUrine failed: %v", err)
	}

	if urine.ID == "" {
		t.Error("expected non-empty urine ID")
	}
	if len(urine.ID) != 26 {
		t.Errorf("expected 26-char ULID, got %d chars", len(urine.ID))
	}
	if urine.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, urine.BabyID)
	}
	if urine.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%q, got %q", user.ID, urine.LoggedBy)
	}
	if urine.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", urine.UpdatedBy)
	}
	if urine.Color == nil || *urine.Color != "pale_yellow" {
		t.Errorf("expected color=pale_yellow, got %v", urine.Color)
	}
	if urine.Notes == nil || *urine.Notes != "normal output" {
		t.Errorf("expected notes='normal output', got %v", urine.Notes)
	}
	if urine.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if urine.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestCreateUrine_NilOptionalFields(t *testing.T) {
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

	urine, err := CreateUrine(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, nil)
	if err != nil {
		t.Fatalf("CreateUrine failed: %v", err)
	}

	if urine.Color != nil {
		t.Errorf("expected nil color, got %v", urine.Color)
	}
	if urine.Notes != nil {
		t.Errorf("expected nil notes, got %v", urine.Notes)
	}
}

func TestGetUrineByID_ReturnsSingleEntry(t *testing.T) {
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

	color := "clear"
	created, err := CreateUrine(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", &color, nil)
	if err != nil {
		t.Fatalf("CreateUrine failed: %v", err)
	}

	fetched, err := GetUrineByID(db, baby.ID, created.ID)
	if err != nil {
		t.Fatalf("GetUrineByID failed: %v", err)
	}

	if fetched.ID != created.ID {
		t.Errorf("expected ID=%q, got %q", created.ID, fetched.ID)
	}
	if fetched.Color == nil || *fetched.Color != "clear" {
		t.Errorf("expected color=clear, got %v", fetched.Color)
	}
}

func TestGetUrineByID_NotFound(t *testing.T) {
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

	_, err = GetUrineByID(db, baby.ID, "nonexistent")
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetUrineByID_WrongBaby(t *testing.T) {
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

	urine, err := CreateUrine(db, baby1.ID, user.ID, "2025-07-01T10:30:00Z", nil, nil)
	if err != nil {
		t.Fatalf("CreateUrine failed: %v", err)
	}

	_, err = GetUrineByID(db, baby2.ID, urine.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows for wrong baby, got %v", err)
	}
}

func TestListUrine_CursorPagination(t *testing.T) {
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
		_, err := CreateUrine(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, nil)
		if err != nil {
			t.Fatalf("CreateUrine %d failed: %v", i, err)
		}
	}

	page, err := ListUrine(db, baby.ID, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("ListUrine page 1 failed: %v", err)
	}
	if len(page.Data) != 3 {
		t.Fatalf("expected 3 items on page 1, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Fatal("expected non-nil next_cursor on page 1")
	}

	for i := 1; i < len(page.Data); i++ {
		if page.Data[i].ID >= page.Data[i-1].ID {
			t.Errorf("expected descending ID order: %s should be < %s", page.Data[i].ID, page.Data[i-1].ID)
		}
	}

	page2, err := ListUrine(db, baby.ID, nil, nil, page.NextCursor, 3)
	if err != nil {
		t.Fatalf("ListUrine page 2 failed: %v", err)
	}
	if len(page2.Data) != 2 {
		t.Fatalf("expected 2 items on page 2, got %d", len(page2.Data))
	}
	if page2.NextCursor != nil {
		t.Error("expected nil next_cursor on last page")
	}
}

func TestListUrine_EmptyResult(t *testing.T) {
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

	page, err := ListUrine(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListUrine failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 urine entries, got %d", len(page.Data))
	}
	if page.NextCursor != nil {
		t.Error("expected nil next_cursor for empty result")
	}
}

func TestListUrine_DateFiltering(t *testing.T) {
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

	_, err = CreateUrine(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, nil)
	if err != nil {
		t.Fatalf("CreateUrine failed: %v", err)
	}
	_, err = CreateUrine(db, baby.ID, user.ID, "2025-07-02T10:30:00Z", nil, nil)
	if err != nil {
		t.Fatalf("CreateUrine failed: %v", err)
	}
	_, err = CreateUrine(db, baby.ID, user.ID, "2025-07-03T10:30:00Z", nil, nil)
	if err != nil {
		t.Fatalf("CreateUrine failed: %v", err)
	}

	from := "2025-07-01"
	to := "2025-07-02"
	page, err := ListUrine(db, baby.ID, &from, &to, nil, 50)
	if err != nil {
		t.Fatalf("ListUrine failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 urine entries in date range, got %d", len(page.Data))
	}
}

func TestUpdateUrine_SetsUpdatedAt(t *testing.T) {
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

	created, err := CreateUrine(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, nil)
	if err != nil {
		t.Fatalf("CreateUrine failed: %v", err)
	}

	newColor := "dark_yellow"
	newNotes := "updated notes"
	updated, err := UpdateUrine(db, baby.ID, created.ID, user.ID, "2025-07-01T11:00:00Z", &newColor, &newNotes)
	if err != nil {
		t.Fatalf("UpdateUrine failed: %v", err)
	}

	if updated.Color == nil || *updated.Color != "dark_yellow" {
		t.Errorf("expected color=dark_yellow, got %v", updated.Color)
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

func TestUpdateUrine_NotFound(t *testing.T) {
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

	_, err = UpdateUrine(db, baby.ID, "nonexistent", user.ID, "2025-07-01T10:30:00Z", nil, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent urine, got nil")
	}
}

func TestDeleteUrine_RemovesEntry(t *testing.T) {
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

	urine, err := CreateUrine(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, nil)
	if err != nil {
		t.Fatalf("CreateUrine failed: %v", err)
	}

	err = DeleteUrine(db, baby.ID, urine.ID)
	if err != nil {
		t.Fatalf("DeleteUrine failed: %v", err)
	}

	_, err = GetUrineByID(db, baby.ID, urine.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteUrine_NotFound(t *testing.T) {
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

	err = DeleteUrine(db, baby.ID, "nonexistent")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent urine, got nil")
	}
}

func TestUrineTable_HasBabyTimestampIndex(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	var name string
	err := db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='index' AND name='idx_urine_baby_timestamp'",
	).Scan(&name)
	if err != nil {
		t.Fatalf("expected idx_urine_baby_timestamp index to exist, got error: %v", err)
	}
}

func TestUrineTable_Columns(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	expected := []string{
		"id", "baby_id", "logged_by", "updated_by", "timestamp",
		"color", "notes", "created_at", "updated_at",
	}
	assertColumns(t, db, "urine", expected)
}

func TestCreateUrine_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateUrine(db, "b1", "u1", "2025-07-01T10:30:00Z", nil, nil)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestListUrine_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListUrine(db, "b1", nil, nil, nil, 50)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestGetUrineByID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := GetUrineByID(db, "b1", "u1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestUpdateUrine_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateUrine(db, "b1", "u1", "user1", "2025-07-01T10:30:00Z", nil, nil)
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestDeleteUrine_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteUrine(db, "b1", "u1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}
