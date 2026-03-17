package store

import (
	"testing"
)

func TestCreateGeneralNote_Success(t *testing.T) {
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

	cat := "behavior"
	note, err := CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "test content", nil, &cat)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}
	if note.Content != "test content" {
		t.Errorf("expected content='test content', got %q", note.Content)
	}
	if note.Category == nil || *note.Category != "behavior" {
		t.Errorf("expected category=behavior, got %v", note.Category)
	}
	if note.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%s, got %s", user.ID, note.LoggedBy)
	}
	if note.UpdatedBy != nil {
		t.Errorf("expected updated_by=nil on creation, got %v", *note.UpdatedBy)
	}
}

func TestGetGeneralNoteByID_NotFound(t *testing.T) {
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

	_, err = GetGeneralNoteByID(db, baby.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent note")
	}
}

func TestListGeneralNotes_Empty(t *testing.T) {
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

	page, err := ListGeneralNotes(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListGeneralNotes failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 notes, got %d", len(page.Data))
	}
}

func TestListGeneralNotes_Pagination(t *testing.T) {
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
		_, err := CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "note", nil, nil)
		if err != nil {
			t.Fatalf("CreateGeneralNote failed: %v", err)
		}
	}

	page, err := ListGeneralNotes(db, baby.ID, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("ListGeneralNotes failed: %v", err)
	}
	if len(page.Data) != 3 {
		t.Errorf("expected 3 notes, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Error("expected non-nil next_cursor")
	}
}

func TestUpdateGeneralNote_Success(t *testing.T) {
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

	note, err := CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "original", nil, nil)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}

	cat := "sleep"
	updated, err := UpdateGeneralNote(db, baby.ID, note.ID, user.ID, "2025-07-01T11:00:00Z", "updated", nil, &cat)
	if err != nil {
		t.Fatalf("UpdateGeneralNote failed: %v", err)
	}
	if updated.Content != "updated" {
		t.Errorf("expected content=updated, got %q", updated.Content)
	}
	if updated.Category == nil || *updated.Category != "sleep" {
		t.Errorf("expected category=sleep, got %v", updated.Category)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%s, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateGeneralNote_NotFound(t *testing.T) {
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

	_, err = UpdateGeneralNote(db, baby.ID, "nonexistent", user.ID, "2025-07-01T11:00:00Z", "updated", nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent note")
	}
}

func TestDeleteGeneralNote_Success(t *testing.T) {
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

	note, err := CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "to delete", nil, nil)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}

	err = DeleteGeneralNote(db, baby.ID, note.ID)
	if err != nil {
		t.Fatalf("DeleteGeneralNote failed: %v", err)
	}

	_, err = GetGeneralNoteByID(db, baby.ID, note.ID)
	if err == nil {
		t.Error("expected note to be deleted")
	}
}

func TestDeleteGeneralNote_NotFound(t *testing.T) {
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

	err = DeleteGeneralNote(db, baby.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent note")
	}
}
