package store

import (
	"testing"
)

func TestCreateTemperature_StoresFieldsCorrectly(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-t1", "t1@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	ts := "2025-07-01T10:30:00Z"
	notes := "normal temp"

	temp, err := CreateTemperature(db, baby.ID, user.ID, ts, 37.2, "rectal", &notes)
	if err != nil {
		t.Fatalf("CreateTemperature failed: %v", err)
	}

	if temp.ID == "" {
		t.Error("expected non-empty temperature ID")
	}
	if temp.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, temp.BabyID)
	}
	if temp.Value != 37.2 {
		t.Errorf("expected value=37.2, got %f", temp.Value)
	}
	if temp.Method != "rectal" {
		t.Errorf("expected method=rectal, got %q", temp.Method)
	}
	if temp.Notes == nil || *temp.Notes != "normal temp" {
		t.Errorf("expected notes=normal temp, got %v", temp.Notes)
	}
	if temp.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", temp.UpdatedBy)
	}
}

func TestCreateTemperature_NilNotes(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-t2", "t2@b.com", "Parent2")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	temp, err := CreateTemperature(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 36.5, "axillary", nil)
	if err != nil {
		t.Fatalf("CreateTemperature failed: %v", err)
	}
	if temp.Notes != nil {
		t.Errorf("expected nil notes, got %v", temp.Notes)
	}
}

func TestGetTemperatureByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetTemperatureByID(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent temperature")
	}
}

func TestListTemperatures_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-t3", "t3@b.com", "Parent3")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	page, err := ListTemperatures(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListTemperatures failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 temperatures, got %d", len(page.Data))
	}
}

func TestListTemperatures_Pagination(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-t4", "t4@b.com", "Parent4")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := CreateTemperature(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 37.0, "rectal", nil)
		if err != nil {
			t.Fatalf("CreateTemperature failed: %v", err)
		}
	}

	page, err := ListTemperatures(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListTemperatures failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 temperatures, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Error("expected non-nil next_cursor")
	}
}

func TestUpdateTemperature_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-t5", "t5@b.com", "Parent5")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	temp, err := CreateTemperature(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 37.0, "rectal", nil)
	if err != nil {
		t.Fatalf("CreateTemperature failed: %v", err)
	}

	notes := "updated"
	updated, err := UpdateTemperature(db, baby.ID, temp.ID, user.ID, "2025-07-01T11:00:00Z", 38.5, "axillary", &notes)
	if err != nil {
		t.Fatalf("UpdateTemperature failed: %v", err)
	}
	if updated.Value != 38.5 {
		t.Errorf("expected value=38.5, got %f", updated.Value)
	}
	if updated.Method != "axillary" {
		t.Errorf("expected method=axillary, got %q", updated.Method)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateTemperature_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateTemperature(db, "nonexistent", "nonexistent", "user1", "2025-07-01T10:30:00Z", 37.0, "rectal", nil)
	if err == nil {
		t.Error("expected error for nonexistent temperature")
	}
}

func TestDeleteTemperature_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-t6", "t6@b.com", "Parent6")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	temp, err := CreateTemperature(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 37.0, "rectal", nil)
	if err != nil {
		t.Fatalf("CreateTemperature failed: %v", err)
	}

	err = DeleteTemperature(db, baby.ID, temp.ID)
	if err != nil {
		t.Fatalf("DeleteTemperature failed: %v", err)
	}

	_, err = GetTemperatureByID(db, baby.ID, temp.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteTemperature_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	err := DeleteTemperature(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent temperature")
	}
}

func TestCreateTemperature_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateTemperature(db, "baby1", "user1", "2025-07-01T10:30:00Z", 37.0, "rectal", nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListTemperatures_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListTemperatures(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListTemperatures_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListTemperatures(db, "baby1", &badDate, nil, nil, 50)
	if err == nil {
		t.Error("expected error for invalid from date")
	}
}

func TestListTemperatures_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListTemperatures(db, "baby1", nil, &badDate, nil, 50)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}

func TestDeleteTemperature_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteTemperature(db, "baby1", "t1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestUpdateTemperature_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateTemperature(db, "baby1", "t1", "user1", "2025-07-01T10:30:00Z", 37.0, "rectal", nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}
