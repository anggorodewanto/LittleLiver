package store

import (
	"testing"
)

func TestCreateImagingStudy_StoresFieldsCorrectly(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-img1", "img1@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	notes := "no acute findings"
	photoKeys := `["photos/abc.jpg","photos/def.pdf"]`
	study, err := CreateImagingStudy(db, baby.ID, user.ID, "2026-04-15T12:00:00Z", "2026-04-15", "CT", &notes, photoKeys)
	if err != nil {
		t.Fatalf("CreateImagingStudy failed: %v", err)
	}

	if study.ID == "" {
		t.Error("expected non-empty ID")
	}
	if study.BabyID != baby.ID {
		t.Errorf("baby_id mismatch: got %q want %q", study.BabyID, baby.ID)
	}
	if study.LoggedBy != user.ID {
		t.Errorf("logged_by mismatch: got %q want %q", study.LoggedBy, user.ID)
	}
	if study.StudyDate != "2026-04-15" {
		t.Errorf("study_date mismatch: got %q", study.StudyDate)
	}
	if study.StudyType != "CT" {
		t.Errorf("study_type mismatch: got %q", study.StudyType)
	}
	if study.Notes == nil || *study.Notes != "no acute findings" {
		t.Errorf("notes mismatch: got %v", study.Notes)
	}
	if study.PhotoKeys != photoKeys {
		t.Errorf("photo_keys mismatch: got %q", study.PhotoKeys)
	}
	if study.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", study.UpdatedBy)
	}
}

func TestCreateImagingStudy_NilNotes(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "google-img2", "img2@b.com", "P")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)

	study, err := CreateImagingStudy(db, baby.ID, user.ID, "2026-04-15T12:00:00Z", "2026-04-15", "Ultrasound", nil, `["photos/x.jpg"]`)
	if err != nil {
		t.Fatalf("CreateImagingStudy failed: %v", err)
	}
	if study.Notes != nil {
		t.Errorf("expected nil notes, got %v", study.Notes)
	}
}

func TestGetImagingStudyByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetImagingStudyByID(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent study")
	}
}

func TestListImagingStudies_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "google-img3", "img3@b.com", "P")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)

	page, err := ListImagingStudies(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListImagingStudies failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected empty list, got %d", len(page.Data))
	}
}

func TestListImagingStudies_Pagination(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "google-img4", "img4@b.com", "P")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)

	for i := 0; i < 3; i++ {
		_, err := CreateImagingStudy(db, baby.ID, user.ID, "2026-04-15T12:00:00Z", "2026-04-15", "MRI", nil, `["photos/x.jpg"]`)
		if err != nil {
			t.Fatalf("CreateImagingStudy %d failed: %v", i, err)
		}
	}

	page, err := ListImagingStudies(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListImagingStudies failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 studies, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Error("expected next_cursor")
	}
}

func TestUpdateImagingStudy_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "google-img5", "img5@b.com", "P")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)

	study, err := CreateImagingStudy(db, baby.ID, user.ID, "2026-04-15T12:00:00Z", "2026-04-15", "CT", nil, `["photos/old.jpg"]`)
	if err != nil {
		t.Fatalf("CreateImagingStudy failed: %v", err)
	}

	notes := "updated note"
	updated, err := UpdateImagingStudy(db, baby.ID, study.ID, user.ID, "2026-04-16T12:00:00Z", "2026-04-16", "Ultrasound", &notes, `["photos/new.jpg"]`)
	if err != nil {
		t.Fatalf("UpdateImagingStudy failed: %v", err)
	}
	if updated.StudyDate != "2026-04-16" {
		t.Errorf("study_date not updated: %q", updated.StudyDate)
	}
	if updated.StudyType != "Ultrasound" {
		t.Errorf("study_type not updated: %q", updated.StudyType)
	}
	if updated.Notes == nil || *updated.Notes != "updated note" {
		t.Errorf("notes not updated: %v", updated.Notes)
	}
	if updated.PhotoKeys != `["photos/new.jpg"]` {
		t.Errorf("photo_keys not updated: %q", updated.PhotoKeys)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("updated_by mismatch: %v", updated.UpdatedBy)
	}
}

func TestUpdateImagingStudy_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateImagingStudy(db, "nonexistent", "nonexistent", "u", "2026-04-15T12:00:00Z", "2026-04-15", "CT", nil, `[]`)
	if err == nil {
		t.Error("expected error for nonexistent study")
	}
}

func TestDeleteImagingStudy_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "google-img6", "img6@b.com", "P")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)

	study, err := CreateImagingStudy(db, baby.ID, user.ID, "2026-04-15T12:00:00Z", "2026-04-15", "CT", nil, `["photos/x.jpg"]`)
	if err != nil {
		t.Fatalf("CreateImagingStudy failed: %v", err)
	}

	if err := DeleteImagingStudy(db, baby.ID, study.ID); err != nil {
		t.Fatalf("DeleteImagingStudy failed: %v", err)
	}

	_, err = GetImagingStudyByID(db, baby.ID, study.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteImagingStudy_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	err := DeleteImagingStudy(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent study")
	}
}

func TestCreateImagingStudy_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateImagingStudy(db, "b", "u", "2026-04-15T12:00:00Z", "2026-04-15", "CT", nil, "[]")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListImagingStudies_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListImagingStudies(db, "b", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestUpdateImagingStudy_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateImagingStudy(db, "b", "x", "u", "2026-04-15T12:00:00Z", "2026-04-15", "CT", nil, "[]")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestDeleteImagingStudy_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteImagingStudy(db, "b", "x")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListImagingStudies_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	bad := "not-a-date"
	_, err := ListImagingStudies(db, "b", &bad, nil, nil, 50)
	if err == nil {
		t.Error("expected error for invalid from")
	}
}

func TestListImagingStudies_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	bad := "not-a-date"
	_, err := ListImagingStudies(db, "b", nil, &bad, nil, 50)
	if err == nil {
		t.Error("expected error for invalid to")
	}
}
