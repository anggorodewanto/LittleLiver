package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

func TestCreateCarePlan_PersistsRowAndAssignsULID(t *testing.T) {
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

	notes := "rotating monthly antibiotics for 12mo"
	plan, err := CreateCarePlan(db, baby.ID, user.ID, "Antibiotic Rotation", &notes, "America/New_York")
	if err != nil {
		t.Fatalf("CreateCarePlan failed: %v", err)
	}
	if plan.ID == "" {
		t.Error("expected plan.ID to be assigned")
	}
	if plan.BabyID != baby.ID {
		t.Errorf("expected baby_id=%s, got %s", baby.ID, plan.BabyID)
	}
	if plan.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%s, got %s", user.ID, plan.LoggedBy)
	}
	if plan.Name != "Antibiotic Rotation" {
		t.Errorf("expected name=Antibiotic Rotation, got %q", plan.Name)
	}
	if plan.Notes == nil || *plan.Notes != notes {
		t.Errorf("expected notes=%q, got %v", notes, plan.Notes)
	}
	if plan.Timezone != "America/New_York" {
		t.Errorf("expected timezone=America/New_York, got %q", plan.Timezone)
	}
	if !plan.Active {
		t.Error("expected active=true on creation")
	}
	if plan.UpdatedBy != nil {
		t.Errorf("expected updated_by=nil on creation, got %v", *plan.UpdatedBy)
	}
}

func TestGetCarePlanByID_ReturnsNotFoundWhenWrongBaby(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	babyA, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby A failed: %v", err)
	}
	babyB, err := CreateBaby(db, user.ID, "Mira", "female", "2025-08-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby B failed: %v", err)
	}

	plan, err := CreateCarePlan(db, babyA.ID, user.ID, "Antibiotic Rotation", nil, "America/New_York")
	if err != nil {
		t.Fatalf("CreateCarePlan failed: %v", err)
	}

	// Querying with the wrong baby id should not return the plan.
	_, err = GetCarePlanByID(db, babyB.ID, plan.ID)
	if err == nil {
		t.Error("expected error fetching plan under wrong baby_id")
	}

	// Sanity: the right baby still finds it.
	got, err := GetCarePlanByID(db, babyA.ID, plan.ID)
	if err != nil {
		t.Fatalf("GetCarePlanByID under correct baby failed: %v", err)
	}
	if got.ID != plan.ID {
		t.Errorf("expected id=%s, got %s", plan.ID, got.ID)
	}
}

func TestCarePlansSchema_TableExists(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	assertTableExists(t, db, "care_plans")
	assertTableExists(t, db, "care_plan_phases")
	assertTableExists(t, db, "care_plan_phase_notifications")
}

func TestListCarePlans_OrdersByCreatedAtDesc(t *testing.T) {
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

	first, err := CreateCarePlan(db, baby.ID, user.ID, "First", nil, "UTC")
	if err != nil {
		t.Fatalf("CreateCarePlan first: %v", err)
	}
	// Bump created_at on first so sort by created_at DESC is well-defined even
	// when two inserts land in the same second; ULIDs in id are also strictly
	// monotonic, but care_plans is sorted by created_at to match medications.
	if _, err := db.Exec("UPDATE care_plans SET created_at = datetime(created_at, '-1 minute') WHERE id = ?", first.ID); err != nil {
		t.Fatalf("backdate first: %v", err)
	}
	second, err := CreateCarePlan(db, baby.ID, user.ID, "Second", nil, "UTC")
	if err != nil {
		t.Fatalf("CreateCarePlan second: %v", err)
	}

	plans, err := ListCarePlans(db, baby.ID)
	if err != nil {
		t.Fatalf("ListCarePlans failed: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(plans))
	}
	if plans[0].ID != second.ID {
		t.Errorf("expected newest first, got %q before %q", plans[0].Name, plans[1].Name)
	}
	if plans[1].ID != first.ID {
		t.Errorf("expected oldest last, got %q at index 1", plans[1].Name)
	}
}

func TestListCarePlans_EmptyReturnsZero(t *testing.T) {
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

	plans, err := ListCarePlans(db, baby.ID)
	if err != nil {
		t.Fatalf("ListCarePlans failed: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(plans))
	}
}

func TestUpdateCarePlan_PreservesTimezoneWhenEmpty(t *testing.T) {
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

	plan, err := CreateCarePlan(db, baby.ID, user.ID, "Plan", nil, "America/New_York")
	if err != nil {
		t.Fatalf("CreateCarePlan: %v", err)
	}

	// Empty timezone string preserves existing.
	updated, err := UpdateCarePlan(db, baby.ID, plan.ID, user.ID, "Plan v2", nil, "", nil)
	if err != nil {
		t.Fatalf("UpdateCarePlan: %v", err)
	}
	if updated.Timezone != "America/New_York" {
		t.Errorf("expected timezone preserved, got %q", updated.Timezone)
	}
	if updated.Name != "Plan v2" {
		t.Errorf("expected name updated, got %q", updated.Name)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%s, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateCarePlan_DeactivateAndChangeTZ(t *testing.T) {
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

	plan, err := CreateCarePlan(db, baby.ID, user.ID, "Plan", nil, "America/New_York")
	if err != nil {
		t.Fatalf("CreateCarePlan: %v", err)
	}

	inactive := false
	updated, err := UpdateCarePlan(db, baby.ID, plan.ID, user.ID, "Plan", nil, "America/Los_Angeles", &inactive)
	if err != nil {
		t.Fatalf("UpdateCarePlan: %v", err)
	}
	if updated.Active {
		t.Error("expected active=false after deactivation")
	}
	if updated.Timezone != "America/Los_Angeles" {
		t.Errorf("expected timezone=Los_Angeles, got %q", updated.Timezone)
	}
}

func TestDeleteCarePlan_CascadesPhasesAndAudit(t *testing.T) {
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

	plan, err := CreateCarePlan(db, baby.ID, user.ID, "Plan", nil, "UTC")
	if err != nil {
		t.Fatalf("CreateCarePlan: %v", err)
	}

	phases := []model.CarePlanPhase{
		{Seq: 1, Label: "A", StartDate: "2026-05-01"},
		{Seq: 2, Label: "B", StartDate: "2026-06-01"},
	}
	saved, err := ReplaceCarePlanPhases(db, plan.ID, phases)
	if err != nil {
		t.Fatalf("ReplaceCarePlanPhases: %v", err)
	}

	// Seed an audit row so cascade behaviour is observable.
	if _, err := db.Exec(
		`INSERT INTO care_plan_phase_notifications (phase_id, kind) VALUES (?, ?)`,
		saved[0].ID, "switch",
	); err != nil {
		t.Fatalf("seed notification: %v", err)
	}

	if err := DeleteCarePlan(db, baby.ID, plan.ID); err != nil {
		t.Fatalf("DeleteCarePlan: %v", err)
	}

	var phaseCount, notifCount, planCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM care_plan_phases WHERE care_plan_id = ?", plan.ID).Scan(&phaseCount); err != nil {
		t.Fatalf("count phases: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM care_plan_phase_notifications WHERE phase_id = ?", saved[0].ID).Scan(&notifCount); err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM care_plans WHERE id = ?", plan.ID).Scan(&planCount); err != nil {
		t.Fatalf("count plans: %v", err)
	}

	if planCount != 0 {
		t.Errorf("expected plan deleted, got %d rows", planCount)
	}
	if phaseCount != 0 {
		t.Errorf("expected phases cascaded, got %d rows", phaseCount)
	}
	if notifCount != 0 {
		t.Errorf("expected notifications cascaded, got %d rows", notifCount)
	}
}

func TestReplaceCarePlanPhases_RejectsNonContiguousSeq(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	plan := mustCreatePlan(t, db, "UTC")
	bad := []model.CarePlanPhase{
		{Seq: 1, Label: "A", StartDate: "2026-05-01"},
		{Seq: 3, Label: "C", StartDate: "2026-06-01"},
	}
	if _, err := ReplaceCarePlanPhases(db, plan.ID, bad); err == nil {
		t.Error("expected error for non-contiguous seq")
	}
}

func TestReplaceCarePlanPhases_RejectsNonMonotonicDates(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	plan := mustCreatePlan(t, db, "UTC")
	bad := []model.CarePlanPhase{
		{Seq: 1, Label: "A", StartDate: "2026-06-01"},
		{Seq: 2, Label: "B", StartDate: "2026-05-01"}, // earlier than phase 1
	}
	if _, err := ReplaceCarePlanPhases(db, plan.ID, bad); err == nil {
		t.Error("expected error for non-monotonic start_date")
	}
}

func TestReplaceCarePlanPhases_RejectsBadDateFormat(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	plan := mustCreatePlan(t, db, "UTC")
	bad := []model.CarePlanPhase{
		{Seq: 1, Label: "A", StartDate: "May 1 2026"},
	}
	if _, err := ReplaceCarePlanPhases(db, plan.ID, bad); err == nil {
		t.Error("expected error for malformed date")
	}
}

func TestReplaceCarePlanPhases_RejectsEmpty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	plan := mustCreatePlan(t, db, "UTC")
	if _, err := ReplaceCarePlanPhases(db, plan.ID, nil); err == nil {
		t.Error("expected error for empty phase list")
	}
}

func TestReplaceCarePlanPhases_ReplacesExistingAtomically(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	plan := mustCreatePlan(t, db, "UTC")
	first := []model.CarePlanPhase{
		{Seq: 1, Label: "A", StartDate: "2026-05-01"},
		{Seq: 2, Label: "B", StartDate: "2026-06-01"},
	}
	if _, err := ReplaceCarePlanPhases(db, plan.ID, first); err != nil {
		t.Fatalf("first replace: %v", err)
	}

	second := []model.CarePlanPhase{
		{Seq: 1, Label: "X", StartDate: "2027-01-01"},
	}
	saved, err := ReplaceCarePlanPhases(db, plan.ID, second)
	if err != nil {
		t.Fatalf("second replace: %v", err)
	}
	if len(saved) != 1 || saved[0].Label != "X" {
		t.Errorf("expected single 'X' phase, got %+v", saved)
	}

	listed, err := ListCarePlanPhases(db, plan.ID)
	if err != nil {
		t.Fatalf("ListCarePlanPhases: %v", err)
	}
	if len(listed) != 1 {
		t.Errorf("expected 1 phase after replace, got %d", len(listed))
	}
}

func TestGetCurrentCarePlanPhase_ReturnsPhaseStraddlingNow(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	plan := mustCreatePlan(t, db, "America/New_York")
	_, err := ReplaceCarePlanPhases(db, plan.ID, []model.CarePlanPhase{
		{Seq: 1, Label: "Cefixime", StartDate: "2026-05-01"},
		{Seq: 2, Label: "Amoxicillin", StartDate: "2026-06-01"},
		{Seq: 3, Label: "Cotrimoxazole", StartDate: "2026-07-01"},
	})
	if err != nil {
		t.Fatalf("ReplaceCarePlanPhases: %v", err)
	}

	loc, _ := time.LoadLocation("America/New_York")
	asOf := time.Date(2026, 6, 15, 12, 0, 0, 0, loc).UTC()

	got, err := GetCurrentCarePlanPhase(db, plan.ID, asOf, loc)
	if err != nil {
		t.Fatalf("GetCurrentCarePlanPhase: %v", err)
	}
	if got == nil {
		t.Fatal("expected current phase, got nil")
	}
	if got.Label != "Amoxicillin" {
		t.Errorf("expected Amoxicillin (phase 2), got %q", got.Label)
	}
}

func TestGetCurrentCarePlanPhase_ReturnsNilBeforeFirstPhase(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	plan := mustCreatePlan(t, db, "UTC")
	_, err := ReplaceCarePlanPhases(db, plan.ID, []model.CarePlanPhase{
		{Seq: 1, Label: "A", StartDate: "2026-05-01"},
		{Seq: 2, Label: "B", StartDate: "2026-06-01"},
	})
	if err != nil {
		t.Fatalf("ReplaceCarePlanPhases: %v", err)
	}

	loc, _ := time.LoadLocation("UTC")
	asOf := time.Date(2026, 4, 30, 23, 0, 0, 0, loc)

	got, err := GetCurrentCarePlanPhase(db, plan.ID, asOf, loc)
	if err != nil {
		t.Fatalf("GetCurrentCarePlanPhase: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil before first phase, got %+v", got)
	}
}

func TestGetCurrentCarePlanPhase_LastPhaseOpenEnded(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	plan := mustCreatePlan(t, db, "UTC")
	_, err := ReplaceCarePlanPhases(db, plan.ID, []model.CarePlanPhase{
		{Seq: 1, Label: "A", StartDate: "2026-05-01"},
		{Seq: 2, Label: "B", StartDate: "2026-06-01"},
	})
	if err != nil {
		t.Fatalf("ReplaceCarePlanPhases: %v", err)
	}

	loc, _ := time.LoadLocation("UTC")
	asOf := time.Date(2030, 1, 1, 0, 0, 0, 0, loc)

	got, err := GetCurrentCarePlanPhase(db, plan.ID, asOf, loc)
	if err != nil {
		t.Fatalf("GetCurrentCarePlanPhase: %v", err)
	}
	if got == nil || got.Label != "B" {
		t.Errorf("expected last phase 'B' open-ended, got %+v", got)
	}
}

func TestGetCurrentCarePlanPhase_LastPhaseEndedClosesPlan(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	plan := mustCreatePlan(t, db, "UTC")
	endsOn := "2026-06-30"
	_, err := ReplaceCarePlanPhases(db, plan.ID, []model.CarePlanPhase{
		{Seq: 1, Label: "A", StartDate: "2026-05-01"},
		{Seq: 2, Label: "B", StartDate: "2026-06-01", EndsOn: &endsOn},
	})
	if err != nil {
		t.Fatalf("ReplaceCarePlanPhases: %v", err)
	}

	loc, _ := time.LoadLocation("UTC")
	asOf := time.Date(2026, 7, 1, 0, 0, 0, 0, loc)

	got, err := GetCurrentCarePlanPhase(db, plan.ID, asOf, loc)
	if err != nil {
		t.Fatalf("GetCurrentCarePlanPhase: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after closed last phase, got %+v", got)
	}
}

func TestRecordPhaseNotification_OnlyOnceForSameKind(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	plan := mustCreatePlan(t, db, "UTC")
	saved, err := ReplaceCarePlanPhases(db, plan.ID, []model.CarePlanPhase{
		{Seq: 1, Label: "A", StartDate: "2026-05-01"},
	})
	if err != nil {
		t.Fatalf("ReplaceCarePlanPhases: %v", err)
	}
	phaseID := saved[0].ID

	first, err := RecordPhaseNotification(db, phaseID, "switch")
	if err != nil {
		t.Fatalf("first RecordPhaseNotification: %v", err)
	}
	if !first {
		t.Error("expected first record to insert (sent=true)")
	}

	second, err := RecordPhaseNotification(db, phaseID, "switch")
	if err != nil {
		t.Fatalf("second RecordPhaseNotification: %v", err)
	}
	if second {
		t.Error("expected second record to be a no-op (sent=false)")
	}

	// Different kind on same phase is independent.
	other, err := RecordPhaseNotification(db, phaseID, "heads_up")
	if err != nil {
		t.Fatalf("heads_up RecordPhaseNotification: %v", err)
	}
	if !other {
		t.Error("expected heads_up to insert independently of switch")
	}
}

func mustCreatePlan(t *testing.T, db *sql.DB, tz string) *model.CarePlan {
	t.Helper()
	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}
	plan, err := CreateCarePlan(db, baby.ID, user.ID, "Plan", nil, tz)
	if err != nil {
		t.Fatalf("CreateCarePlan: %v", err)
	}
	return plan
}

func TestCarePlansSchema_Columns(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	assertColumns(t, db, "care_plans", []string{
		"id", "baby_id", "logged_by", "updated_by",
		"name", "notes", "timezone", "active", "created_at", "updated_at",
	})
	assertColumns(t, db, "care_plan_phases", []string{
		"id", "care_plan_id", "seq", "label", "start_date", "ends_on", "notes", "created_at", "updated_at",
	})
	assertColumns(t, db, "care_plan_phase_notifications", []string{
		"phase_id", "kind", "sent_at",
	})
}
