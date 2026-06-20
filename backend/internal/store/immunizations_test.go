package store

import (
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/immunization"
	"github.com/ablankz/LittleLiver/backend/internal/model"
)

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

// --- CRUD ---

func TestCreateImmunization_StoresFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "googleIm1", "im1@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	rec, err := CreateImmunization(db, baby.ID, user.ID, "DTP_HB_HIB", "DTP-HB-Hib (Pentavalent)", intPtr(1), "2025-03-02", strPtr("Clinic A"), strPtr("LOT123"), strPtr("tolerated well"))
	if err != nil {
		t.Fatalf("CreateImmunization failed: %v", err)
	}
	if len(rec.ID) != 26 {
		t.Errorf("expected 26-char ULID, got %d", len(rec.ID))
	}
	if rec.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, rec.BabyID)
	}
	if rec.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%q, got %q", user.ID, rec.LoggedBy)
	}
	if rec.VaccineCode != "DTP_HB_HIB" {
		t.Errorf("expected vaccine_code=DTP_HB_HIB, got %q", rec.VaccineCode)
	}
	if rec.VaccineName != "DTP-HB-Hib (Pentavalent)" {
		t.Errorf("unexpected vaccine_name %q", rec.VaccineName)
	}
	if rec.DoseNumber == nil || *rec.DoseNumber != 1 {
		t.Errorf("expected dose_number=1, got %v", rec.DoseNumber)
	}
	if rec.AdministeredDate != "2025-03-02" {
		t.Errorf("expected administered_date=2025-03-02, got %q", rec.AdministeredDate)
	}
	if rec.Provider == nil || *rec.Provider != "Clinic A" {
		t.Errorf("expected provider=Clinic A, got %v", rec.Provider)
	}
	if rec.LotNumber == nil || *rec.LotNumber != "LOT123" {
		t.Errorf("expected lot_number=LOT123, got %v", rec.LotNumber)
	}
	if rec.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", rec.UpdatedBy)
	}
	if rec.CreatedAt.IsZero() || rec.UpdatedAt.IsZero() {
		t.Error("expected non-zero timestamps")
	}
}

func TestCreateImmunization_NilOptional(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "googleIm2", "im2@b.com", "Parent")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-01-01", nil, nil, nil, nil)

	rec, err := CreateImmunization(db, baby.ID, user.ID, "BCG", "BCG", nil, "2025-01-02", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateImmunization failed: %v", err)
	}
	if rec.DoseNumber != nil {
		t.Errorf("expected nil dose_number, got %v", rec.DoseNumber)
	}
	if rec.Provider != nil || rec.LotNumber != nil || rec.Notes != nil {
		t.Error("expected nil optional fields")
	}
}

func TestGetImmunizationByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetImmunizationByID(db, "nope-baby", "nope-id")
	if err == nil {
		t.Error("expected error for nonexistent immunization")
	}
}

func TestListImmunizations_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "googleIm3", "im3@b.com", "Parent")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-01-01", nil, nil, nil, nil)

	list, err := ListImmunizations(db, baby.ID)
	if err != nil {
		t.Fatalf("ListImmunizations failed: %v", err)
	}
	if list == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(list) != 0 {
		t.Errorf("expected 0 records, got %d", len(list))
	}
}

func TestListImmunizations_OrdersByDateDesc(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "googleIm4", "im4@b.com", "Parent")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-01-01", nil, nil, nil, nil)

	_, _ = CreateImmunization(db, baby.ID, user.ID, "HB0", "Hepatitis B", intPtr(1), "2025-01-01", nil, nil, nil)
	_, _ = CreateImmunization(db, baby.ID, user.ID, "DTP_HB_HIB", "Penta", intPtr(1), "2025-03-01", nil, nil, nil)
	_, _ = CreateImmunization(db, baby.ID, user.ID, "DTP_HB_HIB", "Penta", intPtr(2), "2025-02-01", nil, nil, nil)

	list, err := ListImmunizations(db, baby.ID)
	if err != nil {
		t.Fatalf("ListImmunizations failed: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 records, got %d", len(list))
	}
	if list[0].AdministeredDate != "2025-03-01" {
		t.Errorf("expected newest first (2025-03-01), got %q", list[0].AdministeredDate)
	}
	if list[2].AdministeredDate != "2025-01-01" {
		t.Errorf("expected oldest last (2025-01-01), got %q", list[2].AdministeredDate)
	}
}

func TestUpdateImmunization_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "googleIm5", "im5@b.com", "Parent")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-01-01", nil, nil, nil, nil)

	rec, _ := CreateImmunization(db, baby.ID, user.ID, "BCG", "BCG", nil, "2025-01-02", nil, nil, nil)

	updated, err := UpdateImmunization(db, baby.ID, rec.ID, user.ID, "BCG", "BCG", intPtr(1), "2025-01-03", strPtr("Clinic B"), nil, strPtr("redness"))
	if err != nil {
		t.Fatalf("UpdateImmunization failed: %v", err)
	}
	if updated.AdministeredDate != "2025-01-03" {
		t.Errorf("expected updated date 2025-01-03, got %q", updated.AdministeredDate)
	}
	if updated.DoseNumber == nil || *updated.DoseNumber != 1 {
		t.Errorf("expected dose_number=1, got %v", updated.DoseNumber)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateImmunization_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateImmunization(db, "nope-baby", "nope-id", "u1", "BCG", "BCG", nil, "2025-01-02", nil, nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent immunization")
	}
}

func TestDeleteImmunization_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "googleIm6", "im6@b.com", "Parent")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-01-01", nil, nil, nil, nil)

	rec, _ := CreateImmunization(db, baby.ID, user.ID, "BCG", "BCG", nil, "2025-01-02", nil, nil, nil)

	if err := DeleteImmunization(db, baby.ID, rec.ID); err != nil {
		t.Fatalf("DeleteImmunization failed: %v", err)
	}
	if _, err := GetImmunizationByID(db, baby.ID, rec.ID); err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteImmunization_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	if err := DeleteImmunization(db, "nope-baby", "nope-id"); err == nil {
		t.Error("expected error for nonexistent immunization")
	}
}

func TestGetImmunizationByID_WrongBaby(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "googleIm7", "im7@b.com", "Parent")
	baby1, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-01-01", nil, nil, nil, nil)
	baby2, _ := CreateBaby(db, user.ID, "Stella", "female", "2025-01-01", nil, nil, nil, nil)

	rec, _ := CreateImmunization(db, baby1.ID, user.ID, "BCG", "BCG", nil, "2025-01-02", nil, nil, nil)

	if _, err := GetImmunizationByID(db, baby2.ID, rec.ID); err == nil {
		t.Error("expected error accessing record from wrong baby")
	}
}

func TestImmunization_CascadeOnBabyDelete(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "googleIm8", "im8@b.com", "Parent")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-01-01", nil, nil, nil, nil)
	_, _ = CreateImmunization(db, baby.ID, user.ID, "BCG", "BCG", nil, "2025-01-02", nil, nil, nil)

	if _, err := db.Exec("DELETE FROM baby_parents WHERE baby_id = ?", baby.ID); err != nil {
		t.Fatalf("delete baby_parents failed: %v", err)
	}
	if _, err := db.Exec("DELETE FROM babies WHERE id = ?", baby.ID); err != nil {
		t.Fatalf("delete baby failed: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM immunizations WHERE baby_id = ?", baby.ID).Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 immunizations after cascade, got %d", count)
	}
}

func TestCreateImmunization_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	if _, err := CreateImmunization(db, "b1", "u1", "BCG", "BCG", nil, "2025-01-02", nil, nil, nil); err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListImmunizations_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	if _, err := ListImmunizations(db, "b1"); err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestGetImmunizationByID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	if _, err := GetImmunizationByID(db, "b1", "i1"); err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestUpdateImmunization_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	if _, err := UpdateImmunization(db, "b1", "i1", "u1", "BCG", "BCG", nil, "2025-01-02", nil, nil, nil); err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestDeleteImmunization_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	if err := DeleteImmunization(db, "b1", "i1"); err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestGetImmunizationSchedule_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	baby := &model.Baby{ID: "b1", DateOfBirth: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	if _, err := GetImmunizationSchedule(db, baby, time.Now(), time.UTC); err == nil {
		t.Error("expected error with closed DB")
	}
}

// --- BuildImmunizationSchedule (pure) ---

func testEntries() []immunization.ScheduleEntry {
	return []immunization.ScheduleEntry{
		{Code: "A", Name: "Vacc A", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 0, AgeLabel: "birth", Mandatory: true},
		{Code: "A", Name: "Vacc A", DoseNumber: 2, DoseLabel: "Dose 2", AgeMonths: 2, AgeLabel: "2 mo", Mandatory: true},
		{Code: "B", Name: "Vacc B", DoseNumber: 1, DoseLabel: "Dose 1", AgeMonths: 9, AgeLabel: "9 mo", Mandatory: false},
	}
}

func findSlot(slots []ImmunizationSlot, code string, dose int) *ImmunizationSlot {
	for i := range slots {
		if slots[i].Code == code && slots[i].DoseNumber == dose {
			return &slots[i]
		}
	}
	return nil
}

func TestBuildImmunizationSchedule_StatusForNewborn(t *testing.T) {
	t.Parallel()
	dob := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	asOf := dob // same day

	slots := BuildImmunizationSchedule(testEntries(), nil, dob, asOf, time.UTC)
	if len(slots) != 3 {
		t.Fatalf("expected 3 slots, got %d", len(slots))
	}

	a1 := findSlot(slots, "A", 1)
	if a1 == nil || a1.Status != ImmunizationStatusDue {
		t.Errorf("expected A/1 due (age 0, today), got %+v", a1)
	}
	a2 := findSlot(slots, "A", 2)
	if a2 == nil || a2.Status != ImmunizationStatusUpcoming {
		t.Errorf("expected A/2 upcoming (age 2, future), got %+v", a2)
	}
	if a2 != nil && a2.DueDate != "2025-08-01" {
		t.Errorf("expected A/2 due_date 2025-08-01, got %q", a2.DueDate)
	}
}

func TestBuildImmunizationSchedule_MarksDone(t *testing.T) {
	t.Parallel()
	dob := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	asOf := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)

	records := []model.Immunization{
		{ID: "rec-a1", VaccineCode: "A", VaccineName: "Vacc A", DoseNumber: intPtr(1), AdministeredDate: "2025-01-02"},
	}

	slots := BuildImmunizationSchedule(testEntries(), records, dob, asOf, time.UTC)

	a1 := findSlot(slots, "A", 1)
	if a1 == nil || a1.Status != ImmunizationStatusDone {
		t.Fatalf("expected A/1 done, got %+v", a1)
	}
	if a1.AdministeredDate == nil || *a1.AdministeredDate != "2025-01-02" {
		t.Errorf("expected administered_date 2025-01-02, got %v", a1.AdministeredDate)
	}
	if a1.RecordID == nil || *a1.RecordID != "rec-a1" {
		t.Errorf("expected record_id rec-a1, got %v", a1.RecordID)
	}

	a2 := findSlot(slots, "A", 2)
	if a2 == nil || a2.Status != ImmunizationStatusDue {
		t.Errorf("expected A/2 due (age 2 < asOf, not done), got %+v", a2)
	}
	b1 := findSlot(slots, "B", 1)
	if b1 == nil || b1.Status != ImmunizationStatusUpcoming {
		t.Errorf("expected B/1 upcoming (age 9, future), got %+v", b1)
	}
}

func TestBuildImmunizationSchedule_OffScheduleRecord(t *testing.T) {
	t.Parallel()
	dob := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	asOf := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)

	records := []model.Immunization{
		{ID: "rec-z", VaccineCode: "Z", VaccineName: "Travel Vaccine", DoseNumber: intPtr(1), AdministeredDate: "2025-02-15"},
	}

	slots := BuildImmunizationSchedule(testEntries(), records, dob, asOf, time.UTC)

	z := findSlot(slots, "Z", 1)
	if z == nil {
		t.Fatalf("expected off-schedule slot for Z, not found")
	}
	if !z.OffSchedule {
		t.Error("expected OffSchedule=true for unmatched record")
	}
	if z.Status != ImmunizationStatusDone {
		t.Errorf("expected off-schedule record done, got %q", z.Status)
	}
	if z.Name != "Travel Vaccine" {
		t.Errorf("expected name from record, got %q", z.Name)
	}
}

func TestBuildImmunizationSchedule_NilDoseRecord(t *testing.T) {
	t.Parallel()
	dob := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	asOf := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)

	records := []model.Immunization{
		{ID: "rec-nil", VaccineCode: "X", VaccineName: "Unknown", DoseNumber: nil, AdministeredDate: "2025-02-01"},
	}

	slots := BuildImmunizationSchedule(testEntries(), records, dob, asOf, time.UTC)
	x := findSlot(slots, "X", 0)
	if x == nil || !x.OffSchedule || x.Status != ImmunizationStatusDone {
		t.Errorf("expected off-schedule done slot for nil-dose record, got %+v", x)
	}
}

func TestBuildImmunizationSchedule_NoDOBNoDueDateCrash(t *testing.T) {
	t.Parallel()
	// zero DOB should not panic; due dates computed from zero time.
	slots := BuildImmunizationSchedule(testEntries(), nil, time.Time{}, time.Now(), time.UTC)
	if len(slots) != 3 {
		t.Fatalf("expected 3 slots, got %d", len(slots))
	}
}

func TestBuildImmunizationSchedule_RealScheduleHasMandatoryAndOptional(t *testing.T) {
	t.Parallel()
	dob := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	slots := BuildImmunizationSchedule(immunization.Schedule(), nil, dob, dob, time.UTC)

	var mandatory, optional int
	for _, s := range slots {
		if s.Mandatory {
			mandatory++
		} else {
			optional++
		}
	}
	if mandatory == 0 {
		t.Error("expected at least one mandatory slot")
	}
	if optional == 0 {
		t.Error("expected at least one optional slot")
	}
}

// --- GetImmunizationSchedule (DB wrapper) ---

func TestGetImmunizationSchedule_Integration(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, _ := UpsertUser(db, "googleIm9", "im9@b.com", "Parent")
	baby, _ := CreateBaby(db, user.ID, "Luna", "female", "2025-01-01", nil, nil, nil, nil)

	_, _ = CreateImmunization(db, baby.ID, user.ID, "HB0", "Hepatitis B (birth dose)", intPtr(1), "2025-01-01", nil, nil, nil)

	asOf := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	slots, err := GetImmunizationSchedule(db, baby, asOf, time.UTC)
	if err != nil {
		t.Fatalf("GetImmunizationSchedule failed: %v", err)
	}
	if len(slots) == 0 {
		t.Fatal("expected non-empty schedule")
	}

	hb := findSlot(slots, "HB0", 1)
	if hb == nil || hb.Status != ImmunizationStatusDone {
		t.Errorf("expected HB0/1 done, got %+v", hb)
	}
}
