package store

import (
	"testing"
)

func TestCreateSkinObservation_StoresFieldsCorrectly(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-skin1", "skin1@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	ts := "2025-07-01T10:30:00Z"
	jaundice := "mild_face"
	rashes := "mild rash on cheeks"
	bruising := "small bruise on arm"
	notes := "normal skin check"

	s, err := CreateSkinObservation(db, baby.ID, user.ID, ts, &jaundice, true, &rashes, &bruising, &notes)
	if err != nil {
		t.Fatalf("CreateSkinObservation failed: %v", err)
	}

	if s.ID == "" {
		t.Error("expected non-empty skin observation ID")
	}
	if s.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, s.BabyID)
	}
	if s.JaundiceLevel == nil || *s.JaundiceLevel != "mild_face" {
		t.Errorf("expected jaundice_level=mild_face, got %v", s.JaundiceLevel)
	}
	if s.ScleralIcterus != true {
		t.Errorf("expected scleral_icterus=true, got %v", s.ScleralIcterus)
	}
	if s.Rashes == nil || *s.Rashes != "mild rash on cheeks" {
		t.Errorf("expected rashes='mild rash on cheeks', got %v", s.Rashes)
	}
	if s.Bruising == nil || *s.Bruising != "small bruise on arm" {
		t.Errorf("expected bruising='small bruise on arm', got %v", s.Bruising)
	}
	if s.Notes == nil || *s.Notes != "normal skin check" {
		t.Errorf("expected notes='normal skin check', got %v", s.Notes)
	}
	if s.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", s.UpdatedBy)
	}
}

func TestCreateSkinObservation_MinimalFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-skin2", "skin2@b.com", "Parent2")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	s, err := CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSkinObservation failed: %v", err)
	}
	if s.JaundiceLevel != nil {
		t.Errorf("expected nil jaundice_level, got %v", s.JaundiceLevel)
	}
	if s.ScleralIcterus != false {
		t.Errorf("expected scleral_icterus=false, got %v", s.ScleralIcterus)
	}
}

func TestGetSkinObservationByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetSkinObservationByID(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent skin observation")
	}
}

func TestListSkinObservations_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-skin3", "skin3@b.com", "Parent3")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	page, err := ListSkinObservations(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListSkinObservations failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 observations, got %d", len(page.Data))
	}
}

func TestListSkinObservations_Pagination(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-skin4", "skin4@b.com", "Parent4")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, false, nil, nil, nil)
		if err != nil {
			t.Fatalf("CreateSkinObservation failed: %v", err)
		}
	}

	page, err := ListSkinObservations(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListSkinObservations failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 observations, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Error("expected non-nil next_cursor")
	}
}

func TestUpdateSkinObservation_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-skin5", "skin5@b.com", "Parent5")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	s, err := CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSkinObservation failed: %v", err)
	}

	jaundice := "severe_limbs_and_trunk"
	notes := "updated"
	updated, err := UpdateSkinObservation(db, baby.ID, s.ID, user.ID, "2025-07-01T11:00:00Z", &jaundice, true, nil, nil, &notes)
	if err != nil {
		t.Fatalf("UpdateSkinObservation failed: %v", err)
	}
	if updated.JaundiceLevel == nil || *updated.JaundiceLevel != "severe_limbs_and_trunk" {
		t.Errorf("expected jaundice_level=severe_limbs_and_trunk, got %v", updated.JaundiceLevel)
	}
	if updated.ScleralIcterus != true {
		t.Errorf("expected scleral_icterus=true, got %v", updated.ScleralIcterus)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateSkinObservation_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateSkinObservation(db, "nonexistent", "nonexistent", "user1", "2025-07-01T10:30:00Z", nil, false, nil, nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent skin observation")
	}
}

func TestDeleteSkinObservation_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-skin6", "skin6@b.com", "Parent6")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	s, err := CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", nil, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSkinObservation failed: %v", err)
	}

	err = DeleteSkinObservation(db, baby.ID, s.ID)
	if err != nil {
		t.Fatalf("DeleteSkinObservation failed: %v", err)
	}

	_, err = GetSkinObservationByID(db, baby.ID, s.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteSkinObservation_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	err := DeleteSkinObservation(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent skin observation")
	}
}

func TestCreateSkinObservation_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateSkinObservation(db, "baby1", "user1", "2025-07-01T10:30:00Z", nil, false, nil, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListSkinObservations_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListSkinObservations(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListSkinObservations_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListSkinObservations(db, "baby1", &badDate, nil, nil, 50)
	if err == nil {
		t.Error("expected error for invalid from date")
	}
}

func TestListSkinObservations_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListSkinObservations(db, "baby1", nil, &badDate, nil, 50)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}

func TestDeleteSkinObservation_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteSkinObservation(db, "baby1", "s1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestUpdateSkinObservation_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateSkinObservation(db, "baby1", "s1", "user1", "2025-07-01T10:30:00Z", nil, false, nil, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}
