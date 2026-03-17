package store

import (
	"testing"
)

func TestCreateMedication_Success(t *testing.T) {
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

	schedule := `["08:00","20:00"]`
	tz := "America/New_York"
	med, err := CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", &schedule, &tz)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}
	if med.Name != "Ursodiol" {
		t.Errorf("expected name=Ursodiol, got %q", med.Name)
	}
	if med.Dose != "50mg" {
		t.Errorf("expected dose=50mg, got %q", med.Dose)
	}
	if med.Frequency != "twice_daily" {
		t.Errorf("expected frequency=twice_daily, got %q", med.Frequency)
	}
	if med.Schedule == nil || *med.Schedule != schedule {
		t.Errorf("expected schedule=%q, got %v", schedule, med.Schedule)
	}
	if med.Timezone == nil || *med.Timezone != tz {
		t.Errorf("expected timezone=%q, got %v", tz, med.Timezone)
	}
	if med.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%s, got %s", user.ID, med.LoggedBy)
	}
	if !med.Active {
		t.Error("expected active=true on creation")
	}
	if med.UpdatedBy != nil {
		t.Errorf("expected updated_by=nil on creation, got %v", *med.UpdatedBy)
	}
}

func TestCreateMedication_NilScheduleAndTimezone(t *testing.T) {
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

	med, err := CreateMedication(db, baby.ID, user.ID, "Vitamin D", "400IU", "once_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}
	if med.Schedule != nil {
		t.Errorf("expected nil schedule, got %v", *med.Schedule)
	}
	if med.Timezone != nil {
		t.Errorf("expected nil timezone, got %v", *med.Timezone)
	}
}

func TestGetMedicationByID_Success(t *testing.T) {
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

	created, err := CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	got, err := GetMedicationByID(db, baby.ID, created.ID)
	if err != nil {
		t.Fatalf("GetMedicationByID failed: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected id=%s, got %s", created.ID, got.ID)
	}
	if got.Name != "Ursodiol" {
		t.Errorf("expected name=Ursodiol, got %q", got.Name)
	}
}

func TestGetMedicationByID_NotFound(t *testing.T) {
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

	_, err = GetMedicationByID(db, baby.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent medication")
	}
}

func TestListMedications_ReturnsActiveAndInactive(t *testing.T) {
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

	// Create two medications
	med1, err := CreateMedication(db, baby.ID, user.ID, "Active Med", "10mg", "once_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}
	_, err = CreateMedication(db, baby.ID, user.ID, "Inactive Med", "20mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Deactivate the first one
	active := false
	_, err = UpdateMedication(db, baby.ID, med1.ID, user.ID, "Active Med", "10mg", "once_daily", nil, nil, &active)
	if err != nil {
		t.Fatalf("UpdateMedication failed: %v", err)
	}

	meds, err := ListMedications(db, baby.ID)
	if err != nil {
		t.Fatalf("ListMedications failed: %v", err)
	}
	if len(meds) != 2 {
		t.Errorf("expected 2 medications (active+inactive), got %d", len(meds))
	}
}

func TestListMedications_Empty(t *testing.T) {
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

	meds, err := ListMedications(db, baby.ID)
	if err != nil {
		t.Fatalf("ListMedications failed: %v", err)
	}
	if len(meds) != 0 {
		t.Errorf("expected 0 medications, got %d", len(meds))
	}
}

func TestUpdateMedication_Deactivate(t *testing.T) {
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

	med, err := CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	active := false
	updated, err := UpdateMedication(db, baby.ID, med.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil, &active)
	if err != nil {
		t.Fatalf("UpdateMedication failed: %v", err)
	}
	if updated.Active {
		t.Error("expected active=false after deactivation")
	}
}

func TestUpdateMedication_SetsUpdatedBy(t *testing.T) {
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

	med, err := CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}
	if med.UpdatedBy != nil {
		t.Errorf("expected updated_by=nil on creation, got %v", *med.UpdatedBy)
	}

	updated, err := UpdateMedication(db, baby.ID, med.ID, user.ID, "Ursodiol", "60mg", "twice_daily", nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateMedication failed: %v", err)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%s, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateMedication_TimezoneUpdate(t *testing.T) {
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

	oldTZ := "America/New_York"
	med, err := CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, &oldTZ)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	newTZ := "America/Los_Angeles"
	updated, err := UpdateMedication(db, baby.ID, med.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, &newTZ, nil)
	if err != nil {
		t.Fatalf("UpdateMedication failed: %v", err)
	}
	if updated.Timezone == nil || *updated.Timezone != newTZ {
		t.Errorf("expected timezone=%q, got %v", newTZ, updated.Timezone)
	}
}

func TestUpdateMedication_NotFound(t *testing.T) {
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

	_, err = UpdateMedication(db, baby.ID, "nonexistent", user.ID, "Name", "10mg", "once_daily", nil, nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent medication update")
	}
}

func TestMedicationsSchema_TableExists(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	assertTableExists(t, db, "medications")
}

func TestMedicationsSchema_Columns(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	expected := []string{
		"id", "baby_id", "logged_by", "updated_by",
		"name", "dose", "frequency", "schedule",
		"timezone", "active", "created_at", "updated_at",
	}
	assertColumns(t, db, "medications", expected)
}

func TestMedicationsSchema_IndexExists(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	var name string
	err := db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='index' AND name='idx_medications_baby_id'",
	).Scan(&name)
	if err != nil {
		t.Errorf("index idx_medications_baby_id does not exist: %v", err)
	}
}

func TestMedicationsSchema_CascadeDelete(t *testing.T) {
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

	_, err = CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication failed: %v", err)
	}

	// Delete baby should cascade to medications
	_, err = db.Exec("DELETE FROM babies WHERE id = ?", baby.ID)
	if err != nil {
		t.Fatalf("delete baby failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM medications WHERE baby_id = ?", baby.ID).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 medications after cascade delete, got %d", count)
	}
}
