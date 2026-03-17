package store

import (
	"testing"
)

func TestCreateAbdomen_StoresFieldsCorrectly(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-a1", "a1@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	ts := "2025-07-01T10:30:00Z"
	girth := 32.5
	notes := "normal"

	a, err := CreateAbdomen(db, baby.ID, user.ID, ts, "soft", false, &girth, &notes)
	if err != nil {
		t.Fatalf("CreateAbdomen failed: %v", err)
	}

	if a.ID == "" {
		t.Error("expected non-empty abdomen ID")
	}
	if a.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, a.BabyID)
	}
	if a.Firmness != "soft" {
		t.Errorf("expected firmness=soft, got %q", a.Firmness)
	}
	if a.Tenderness != false {
		t.Errorf("expected tenderness=false, got %v", a.Tenderness)
	}
	if a.GirthCm == nil || *a.GirthCm != 32.5 {
		t.Errorf("expected girth_cm=32.5, got %v", a.GirthCm)
	}
	if a.Notes == nil || *a.Notes != "normal" {
		t.Errorf("expected notes=normal, got %v", a.Notes)
	}
	if a.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", a.UpdatedBy)
	}
}

func TestCreateAbdomen_WithTenderness(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-a2", "a2@b.com", "Parent2")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	a, err := CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "firm", true, nil, nil)
	if err != nil {
		t.Fatalf("CreateAbdomen failed: %v", err)
	}
	if a.Tenderness != true {
		t.Errorf("expected tenderness=true, got %v", a.Tenderness)
	}
	if a.GirthCm != nil {
		t.Errorf("expected nil girth_cm, got %v", a.GirthCm)
	}
}

func TestGetAbdomenByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetAbdomenByID(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent abdomen")
	}
}

func TestListAbdomen_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-a3", "a3@b.com", "Parent3")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	page, err := ListAbdomen(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListAbdomen failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 observations, got %d", len(page.Data))
	}
}

func TestListAbdomen_Pagination(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-a4", "a4@b.com", "Parent4")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "soft", false, nil, nil)
		if err != nil {
			t.Fatalf("CreateAbdomen failed: %v", err)
		}
	}

	page, err := ListAbdomen(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListAbdomen failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 observations, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Error("expected non-nil next_cursor")
	}
}

func TestUpdateAbdomen_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-a5", "a5@b.com", "Parent5")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	a, err := CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "soft", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAbdomen failed: %v", err)
	}

	girth := 34.0
	notes := "updated"
	updated, err := UpdateAbdomen(db, baby.ID, a.ID, user.ID, "2025-07-01T11:00:00Z", "firm", true, &girth, &notes)
	if err != nil {
		t.Fatalf("UpdateAbdomen failed: %v", err)
	}
	if updated.Firmness != "firm" {
		t.Errorf("expected firmness=firm, got %q", updated.Firmness)
	}
	if updated.Tenderness != true {
		t.Errorf("expected tenderness=true, got %v", updated.Tenderness)
	}
	if updated.GirthCm == nil || *updated.GirthCm != 34.0 {
		t.Errorf("expected girth_cm=34.0, got %v", updated.GirthCm)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateAbdomen_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateAbdomen(db, "nonexistent", "nonexistent", "user1", "2025-07-01T10:30:00Z", "soft", false, nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent abdomen")
	}
}

func TestDeleteAbdomen_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-a6", "a6@b.com", "Parent6")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	a, err := CreateAbdomen(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "soft", false, nil, nil)
	if err != nil {
		t.Fatalf("CreateAbdomen failed: %v", err)
	}

	err = DeleteAbdomen(db, baby.ID, a.ID)
	if err != nil {
		t.Fatalf("DeleteAbdomen failed: %v", err)
	}

	_, err = GetAbdomenByID(db, baby.ID, a.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteAbdomen_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	err := DeleteAbdomen(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent abdomen")
	}
}

func TestCreateAbdomen_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateAbdomen(db, "baby1", "user1", "2025-07-01T10:30:00Z", "soft", false, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListAbdomen_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListAbdomen(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListAbdomen_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListAbdomen(db, "baby1", &badDate, nil, nil, 50)
	if err == nil {
		t.Error("expected error for invalid from date")
	}
}

func TestListAbdomen_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListAbdomen(db, "baby1", nil, &badDate, nil, 50)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}

func TestDeleteAbdomen_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteAbdomen(db, "baby1", "a1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestUpdateAbdomen_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateAbdomen(db, "baby1", "a1", "user1", "2025-07-01T10:30:00Z", "soft", false, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}
