package store

import (
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

func TestFindDuplicateLabResult_MatchWithinWindow(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-dup1", "dup1@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Baby", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	twoDaysAgo := time.Now().UTC().AddDate(0, 0, -2).Format(model.DateTimeFormat)
	unit := "U/L"
	existing, err := CreateLabResult(db, baby.ID, user.ID, twoDaysAgo, "ALT", "45", &unit, nil, nil)
	if err != nil {
		t.Fatalf("CreateLabResult: %v", err)
	}

	match, err := FindDuplicateLabResult(db, baby.ID, "ALT", "45", time.Now().UTC())
	if err != nil {
		t.Fatalf("FindDuplicateLabResult: %v", err)
	}
	if match == nil {
		t.Fatal("expected a match within +-3 day window")
	}
	if match.ID != existing.ID {
		t.Errorf("expected match ID %s, got %s", existing.ID, match.ID)
	}
}

func TestFindDuplicateLabResult_CaseInsensitive(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-dup2", "dup2@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Baby", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	twoDaysAgo := time.Now().UTC().AddDate(0, 0, -2).Format(model.DateTimeFormat)
	_, err = CreateLabResult(db, baby.ID, user.ID, twoDaysAgo, "alt", "45", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateLabResult: %v", err)
	}

	match, err := FindDuplicateLabResult(db, baby.ID, "ALT", "45", time.Now().UTC())
	if err != nil {
		t.Fatalf("FindDuplicateLabResult: %v", err)
	}
	if match == nil {
		t.Fatal("expected a case-insensitive match")
	}
}

func TestFindDuplicateLabResult_NoMatchOutsideWindow(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-dup3", "dup3@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Baby", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	tenDaysAgo := time.Now().UTC().AddDate(0, 0, -10).Format(model.DateTimeFormat)
	_, err = CreateLabResult(db, baby.ID, user.ID, tenDaysAgo, "ALT", "45", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateLabResult: %v", err)
	}

	match, err := FindDuplicateLabResult(db, baby.ID, "ALT", "45", time.Now().UTC())
	if err != nil {
		t.Fatalf("FindDuplicateLabResult: %v", err)
	}
	if match != nil {
		t.Error("expected no match outside +-3 day window")
	}
}

func TestFindDuplicateLabResult_NoMatchDifferentValue(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-dup4", "dup4@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Baby", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	twoDaysAgo := time.Now().UTC().AddDate(0, 0, -2).Format(model.DateTimeFormat)
	_, err = CreateLabResult(db, baby.ID, user.ID, twoDaysAgo, "ALT", "45", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateLabResult: %v", err)
	}

	match, err := FindDuplicateLabResult(db, baby.ID, "ALT", "50", time.Now().UTC())
	if err != nil {
		t.Fatalf("FindDuplicateLabResult: %v", err)
	}
	if match != nil {
		t.Error("expected no match for different value")
	}
}

func TestFindDuplicateLabResult_NoneExist(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-dup5", "dup5@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Baby", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	match, err := FindDuplicateLabResult(db, baby.ID, "ALT", "45", time.Now().UTC())
	if err != nil {
		t.Fatalf("FindDuplicateLabResult: %v", err)
	}
	if match != nil {
		t.Error("expected nil when no lab results exist")
	}
}
