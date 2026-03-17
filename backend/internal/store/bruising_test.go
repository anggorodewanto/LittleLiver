package store

import (
	"testing"
)

func TestCreateBruising_StoresFieldsCorrectly(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-br1", "br1@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	ts := "2025-07-01T10:30:00Z"
	sizeCm := 1.5
	color := "purple"
	notes := "noticed after bath"

	b, err := CreateBruising(db, baby.ID, user.ID, ts, "left_arm", "small_<1cm", &sizeCm, &color, &notes)
	if err != nil {
		t.Fatalf("CreateBruising failed: %v", err)
	}

	if b.ID == "" {
		t.Error("expected non-empty bruising ID")
	}
	if b.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, b.BabyID)
	}
	if b.Location != "left_arm" {
		t.Errorf("expected location=left_arm, got %q", b.Location)
	}
	if b.SizeEstimate != "small_<1cm" {
		t.Errorf("expected size_estimate=small_<1cm, got %q", b.SizeEstimate)
	}
	if b.SizeCm == nil || *b.SizeCm != 1.5 {
		t.Errorf("expected size_cm=1.5, got %v", b.SizeCm)
	}
	if b.Color == nil || *b.Color != "purple" {
		t.Errorf("expected color=purple, got %v", b.Color)
	}
	if b.Notes == nil || *b.Notes != "noticed after bath" {
		t.Errorf("expected notes='noticed after bath', got %v", b.Notes)
	}
	if b.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", b.UpdatedBy)
	}
}

func TestCreateBruising_MinimalFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-br2", "br2@b.com", "Parent2")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	b, err := CreateBruising(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "right_leg", "medium_1-3cm", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBruising failed: %v", err)
	}
	if b.SizeCm != nil {
		t.Errorf("expected nil size_cm, got %v", b.SizeCm)
	}
	if b.Color != nil {
		t.Errorf("expected nil color, got %v", b.Color)
	}
}

func TestGetBruisingByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetBruisingByID(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent bruising")
	}
}

func TestListBruising_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-br3", "br3@b.com", "Parent3")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	page, err := ListBruising(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListBruising failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 observations, got %d", len(page.Data))
	}
}

func TestListBruising_Pagination(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-br4", "br4@b.com", "Parent4")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := CreateBruising(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "arm", "small_<1cm", nil, nil, nil)
		if err != nil {
			t.Fatalf("CreateBruising failed: %v", err)
		}
	}

	page, err := ListBruising(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListBruising failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 observations, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Error("expected non-nil next_cursor")
	}
}

func TestUpdateBruising_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-br5", "br5@b.com", "Parent5")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	b, err := CreateBruising(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "arm", "small_<1cm", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBruising failed: %v", err)
	}

	sizeCm := 2.5
	color := "yellow-green"
	notes := "fading"
	updated, err := UpdateBruising(db, baby.ID, b.ID, user.ID, "2025-07-01T11:00:00Z", "left_arm", "medium_1-3cm", &sizeCm, &color, &notes)
	if err != nil {
		t.Fatalf("UpdateBruising failed: %v", err)
	}
	if updated.Location != "left_arm" {
		t.Errorf("expected location=left_arm, got %q", updated.Location)
	}
	if updated.SizeEstimate != "medium_1-3cm" {
		t.Errorf("expected size_estimate=medium_1-3cm, got %q", updated.SizeEstimate)
	}
	if updated.SizeCm == nil || *updated.SizeCm != 2.5 {
		t.Errorf("expected size_cm=2.5, got %v", updated.SizeCm)
	}
	if updated.Color == nil || *updated.Color != "yellow-green" {
		t.Errorf("expected color=yellow-green, got %v", updated.Color)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateBruising_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateBruising(db, "nonexistent", "nonexistent", "user1", "2025-07-01T10:30:00Z", "arm", "small_<1cm", nil, nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent bruising")
	}
}

func TestDeleteBruising_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-br6", "br6@b.com", "Parent6")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	b, err := CreateBruising(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "arm", "small_<1cm", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBruising failed: %v", err)
	}

	err = DeleteBruising(db, baby.ID, b.ID)
	if err != nil {
		t.Fatalf("DeleteBruising failed: %v", err)
	}

	_, err = GetBruisingByID(db, baby.ID, b.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteBruising_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	err := DeleteBruising(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent bruising")
	}
}

func TestCreateBruising_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateBruising(db, "baby1", "user1", "2025-07-01T10:30:00Z", "arm", "small_<1cm", nil, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListBruising_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListBruising(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListBruising_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListBruising(db, "baby1", &badDate, nil, nil, 50)
	if err == nil {
		t.Error("expected error for invalid from date")
	}
}

func TestListBruising_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListBruising(db, "baby1", nil, &badDate, nil, 50)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}

func TestDeleteBruising_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteBruising(db, "baby1", "b1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestUpdateBruising_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateBruising(db, "baby1", "b1", "user1", "2025-07-01T10:30:00Z", "arm", "small_<1cm", nil, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}
