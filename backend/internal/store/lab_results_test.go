package store

import (
	"testing"
)

func TestCreateLabResult_StoresFieldsCorrectly(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-lab1", "lab1@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	ts := "2025-07-01T10:30:00Z"
	unit := "mg/dL"
	normalRange := "0.1-1.2"
	notes := "slightly elevated"

	l, err := CreateLabResult(db, baby.ID, user.ID, ts, "total_bilirubin", "2.5", &unit, &normalRange, &notes)
	if err != nil {
		t.Fatalf("CreateLabResult failed: %v", err)
	}

	if l.ID == "" {
		t.Error("expected non-empty lab result ID")
	}
	if l.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, l.BabyID)
	}
	if l.TestName != "total_bilirubin" {
		t.Errorf("expected test_name=total_bilirubin, got %q", l.TestName)
	}
	if l.Value != "2.5" {
		t.Errorf("expected value=2.5, got %q", l.Value)
	}
	if l.Unit == nil || *l.Unit != "mg/dL" {
		t.Errorf("expected unit=mg/dL, got %v", l.Unit)
	}
	if l.NormalRange == nil || *l.NormalRange != "0.1-1.2" {
		t.Errorf("expected normal_range=0.1-1.2, got %v", l.NormalRange)
	}
	if l.Notes == nil || *l.Notes != "slightly elevated" {
		t.Errorf("expected notes='slightly elevated', got %v", l.Notes)
	}
	if l.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", l.UpdatedBy)
	}
}

func TestCreateLabResult_ArbitraryTestName(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-lab2", "lab2@b.com", "Parent2")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	// EAV pattern: test_name can be anything
	l, err := CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "GGT", "150", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateLabResult failed: %v", err)
	}
	if l.TestName != "GGT" {
		t.Errorf("expected test_name=GGT, got %q", l.TestName)
	}
	if l.Value != "150" {
		t.Errorf("expected value=150, got %q", l.Value)
	}
	if l.Unit != nil {
		t.Errorf("expected nil unit, got %v", l.Unit)
	}
}

func TestGetLabResultByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetLabResultByID(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent lab result")
	}
}

func TestListLabResults_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-lab3", "lab3@b.com", "Parent3")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	page, err := ListLabResults(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListLabResults failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 results, got %d", len(page.Data))
	}
}

func TestListLabResults_Pagination(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-lab4", "lab4@b.com", "Parent4")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "ALT", "45", nil, nil, nil)
		if err != nil {
			t.Fatalf("CreateLabResult failed: %v", err)
		}
	}

	page, err := ListLabResults(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListLabResults failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 results, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Error("expected non-nil next_cursor")
	}
}

func TestUpdateLabResult_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-lab5", "lab5@b.com", "Parent5")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	l, err := CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "ALT", "45", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateLabResult failed: %v", err)
	}

	unit := "U/L"
	normalRange := "7-56"
	notes := "within normal range"
	updated, err := UpdateLabResult(db, baby.ID, l.ID, user.ID, "2025-07-01T11:00:00Z", "ALT", "42", &unit, &normalRange, &notes)
	if err != nil {
		t.Fatalf("UpdateLabResult failed: %v", err)
	}
	if updated.Value != "42" {
		t.Errorf("expected value=42, got %q", updated.Value)
	}
	if updated.Unit == nil || *updated.Unit != "U/L" {
		t.Errorf("expected unit=U/L, got %v", updated.Unit)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateLabResult_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateLabResult(db, "nonexistent", "nonexistent", "user1", "2025-07-01T10:30:00Z", "ALT", "45", nil, nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent lab result")
	}
}

func TestDeleteLabResult_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-lab6", "lab6@b.com", "Parent6")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	l, err := CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "ALT", "45", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateLabResult failed: %v", err)
	}

	err = DeleteLabResult(db, baby.ID, l.ID)
	if err != nil {
		t.Fatalf("DeleteLabResult failed: %v", err)
	}

	_, err = GetLabResultByID(db, baby.ID, l.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteLabResult_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	err := DeleteLabResult(db, "nonexistent", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent lab result")
	}
}

func TestCreateLabResult_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateLabResult(db, "baby1", "user1", "2025-07-01T10:30:00Z", "ALT", "45", nil, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListLabResults_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListLabResults(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListLabResults_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListLabResults(db, "baby1", &badDate, nil, nil, 50)
	if err == nil {
		t.Error("expected error for invalid from date")
	}
}

func TestListLabResults_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListLabResults(db, "baby1", nil, &badDate, nil, 50)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}

func TestDeleteLabResult_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteLabResult(db, "baby1", "l1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListDistinctLabTests_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-labdist1", "labdist1@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	suggestions, err := ListDistinctLabTests(db, baby.ID)
	if err != nil {
		t.Fatalf("ListDistinctLabTests failed: %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(suggestions))
	}
}

func TestListDistinctLabTests_ReturnsDistinctTests(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-labdist2", "labdist2@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	unit1 := "mg/dL"
	range1 := "0.1-1.2"
	unit2 := "U/L"
	// Create two different tests, ALT logged twice
	_, _ = CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:00:00Z", "total_bilirubin", "2.5", &unit1, &range1, nil)
	_, _ = CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:00:00Z", "ALT", "45", &unit2, nil, nil)
	_, _ = CreateLabResult(db, baby.ID, user.ID, "2025-07-02T10:00:00Z", "ALT", "42", &unit2, nil, nil)

	suggestions, err := ListDistinctLabTests(db, baby.ID)
	if err != nil {
		t.Fatalf("ListDistinctLabTests failed: %v", err)
	}
	if len(suggestions) != 2 {
		t.Fatalf("expected 2 distinct suggestions, got %d", len(suggestions))
	}
}

func TestListDistinctLabTests_MostRecentUnitWins(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-labdist3", "labdist3@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	oldUnit := "old_unit"
	newUnit := "new_unit"
	newRange := "10-50"
	// Older entry with old unit
	_, _ = CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:00:00Z", "ALT", "45", &oldUnit, nil, nil)
	// Newer entry with new unit and range
	_, _ = CreateLabResult(db, baby.ID, user.ID, "2025-07-02T10:00:00Z", "ALT", "42", &newUnit, &newRange, nil)

	suggestions, err := ListDistinctLabTests(db, baby.ID)
	if err != nil {
		t.Fatalf("ListDistinctLabTests failed: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	if suggestions[0].Unit == nil || *suggestions[0].Unit != "new_unit" {
		t.Errorf("expected unit=new_unit, got %v", suggestions[0].Unit)
	}
	if suggestions[0].NormalRange == nil || *suggestions[0].NormalRange != "10-50" {
		t.Errorf("expected normal_range=10-50, got %v", suggestions[0].NormalRange)
	}
}

func TestListDistinctLabTests_BabyIsolation(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-labdist4", "labdist4@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby1, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	baby2, err := CreateBaby(db, user.ID, "Stella", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	unit := "mg/dL"
	_, _ = CreateLabResult(db, baby1.ID, user.ID, "2025-07-01T10:00:00Z", "total_bilirubin", "2.5", &unit, nil, nil)
	_, _ = CreateLabResult(db, baby2.ID, user.ID, "2025-07-01T10:00:00Z", "ALT", "45", nil, nil, nil)

	suggestions, err := ListDistinctLabTests(db, baby1.ID)
	if err != nil {
		t.Fatalf("ListDistinctLabTests failed: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion for baby1, got %d", len(suggestions))
	}
	if suggestions[0].TestName != "total_bilirubin" {
		t.Errorf("expected test_name=total_bilirubin, got %q", suggestions[0].TestName)
	}
}

func TestListDistinctLabTests_NullUnitAndRange(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-labdist5", "labdist5@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, _ = CreateLabResult(db, baby.ID, user.ID, "2025-07-01T10:00:00Z", "INR", "1.1", nil, nil, nil)

	suggestions, err := ListDistinctLabTests(db, baby.ID)
	if err != nil {
		t.Fatalf("ListDistinctLabTests failed: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	if suggestions[0].TestName != "INR" {
		t.Errorf("expected test_name=INR, got %q", suggestions[0].TestName)
	}
	if suggestions[0].Unit != nil {
		t.Errorf("expected nil unit, got %v", suggestions[0].Unit)
	}
	if suggestions[0].NormalRange != nil {
		t.Errorf("expected nil normal_range, got %v", suggestions[0].NormalRange)
	}
}

func TestListDistinctLabTests_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListDistinctLabTests(db, "baby1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestUpdateLabResult_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateLabResult(db, "baby1", "l1", "user1", "2025-07-01T10:30:00Z", "ALT", "45", nil, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}
