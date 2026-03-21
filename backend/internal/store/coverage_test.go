package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// --- parseMetricTimes error path tests ---

func TestParseMetricTimes_ValidTimes(t *testing.T) {
	t.Parallel()
	ts, ca, ua, err := parseMetricTimes(
		"2025-07-01T10:00:00Z",
		"2025-07-01 10:00:00",
		"2025-07-01 10:00:00",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.IsZero() || ca.IsZero() || ua.IsZero() {
		t.Error("expected non-zero times")
	}
}

func TestParseMetricTimes_InvalidTimestamp(t *testing.T) {
	t.Parallel()
	_, _, _, err := parseMetricTimes("not-a-time", "2025-07-01 10:00:00", "2025-07-01 10:00:00")
	if err == nil {
		t.Fatal("expected error for invalid timestamp")
	}
}

func TestParseMetricTimes_InvalidCreatedAt(t *testing.T) {
	t.Parallel()
	_, _, _, err := parseMetricTimes("2025-07-01T10:00:00Z", "not-a-time", "2025-07-01 10:00:00")
	if err == nil {
		t.Fatal("expected error for invalid created_at")
	}
}

func TestParseMetricTimes_InvalidUpdatedAt(t *testing.T) {
	t.Parallel()
	_, _, _, err := parseMetricTimes("2025-07-01T10:00:00Z", "2025-07-01 10:00:00", "not-a-time")
	if err == nil {
		t.Fatal("expected error for invalid updated_at")
	}
}

// --- ParseDateRange error paths ---

func TestParseDateRange_ValidDates(t *testing.T) {
	t.Parallel()
	from, to, err := ParseDateRange("2025-07-01", "2025-07-02")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if from == "" || to == "" {
		t.Error("expected non-empty date strings")
	}
}

func TestParseDateRange_InvalidFrom(t *testing.T) {
	t.Parallel()
	_, _, err := ParseDateRange("bad-date", "2025-07-02")
	if err == nil {
		t.Fatal("expected error for invalid from date")
	}
}

func TestParseDateRange_InvalidTo(t *testing.T) {
	t.Parallel()
	_, _, err := ParseDateRange("2025-07-01", "bad-date")
	if err == nil {
		t.Fatal("expected error for invalid to date")
	}
}

// --- emptySliceIfNil ---

func TestEmptySliceIfNil_NilSlice(t *testing.T) {
	t.Parallel()
	var s []int
	result := emptySliceIfNil(s)
	if result == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(result) != 0 {
		t.Errorf("expected len 0, got %d", len(result))
	}
}

func TestEmptySliceIfNil_NonNilSlice(t *testing.T) {
	t.Parallel()
	s := []int{1, 2, 3}
	result := emptySliceIfNil(s)
	if len(result) != 3 {
		t.Errorf("expected len 3, got %d", len(result))
	}
}

// --- nullStr / nullFloat / nullInt ---

func TestNullStr_Valid(t *testing.T) {
	t.Parallel()
	ns := sql.NullString{String: "hello", Valid: true}
	result := nullStr(ns)
	if result == nil || *result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestNullStr_Invalid(t *testing.T) {
	t.Parallel()
	ns := sql.NullString{}
	result := nullStr(ns)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestNullFloat_Valid(t *testing.T) {
	t.Parallel()
	nf := sql.NullFloat64{Float64: 42.5, Valid: true}
	result := nullFloat(nf)
	if result == nil || *result != 42.5 {
		t.Errorf("expected 42.5, got %v", result)
	}
}

func TestNullFloat_Invalid(t *testing.T) {
	t.Parallel()
	nf := sql.NullFloat64{}
	result := nullFloat(nf)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestNullInt_Valid(t *testing.T) {
	t.Parallel()
	ni := sql.NullInt64{Int64: 99, Valid: true}
	result := nullInt(ni)
	if result == nil || *result != 99 {
		t.Errorf("expected 99, got %v", result)
	}
}

func TestNullInt_Invalid(t *testing.T) {
	t.Parallel()
	ni := sql.NullInt64{}
	result := nullInt(ni)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// --- checkRowsAffected ---

func TestCheckRowsAffected_ZeroRows(t *testing.T) {
	t.Parallel()
	res := &mockResult{rowsAffected: 0}
	err := checkRowsAffected(res, "test-op")
	if err == nil {
		t.Fatal("expected error for 0 rows affected")
	}
}

func TestCheckRowsAffected_OneRow(t *testing.T) {
	t.Parallel()
	res := &mockResult{rowsAffected: 1}
	err := checkRowsAffected(res, "test-op")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GetDashboardSummary with invalid date range ---

func TestGetDashboardSummary_InvalidDateRange(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetDashboardSummary(db, "baby-id", "invalid-date", "2025-07-01", time.UTC)
	if err == nil {
		t.Fatal("expected error for invalid date range")
	}
}

// --- GetStoolColorTrend with NULL color_label ---

func TestGetStoolColorTrend_NullColorLabel(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov1", "cov1@test.com", "Cov1")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	// Create stool with nil color_label
	today := time.Now().UTC().Format("2006-01-02")
	ts := today + "T12:00:00Z"
	_, err = CreateStool(db, baby.ID, user.ID, ts, 5, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateStool: %v", err)
	}

	trend, err := GetStoolColorTrend(db, baby.ID)
	if err != nil {
		t.Fatalf("GetStoolColorTrend: %v", err)
	}
	if len(trend) == 0 {
		t.Fatal("expected at least 1 trend entry")
	}
	// With NULL color_label, Color should be empty string
	if trend[0].Color != "" {
		t.Errorf("expected empty color for null color_label, got %q", trend[0].Color)
	}
}

func TestGetStoolColorTrend_EmptyData(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov2", "cov2@test.com", "Cov2")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby2", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	trend, err := GetStoolColorTrend(db, baby.ID)
	if err != nil {
		t.Fatalf("GetStoolColorTrend: %v", err)
	}
	if trend == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(trend) != 0 {
		t.Errorf("expected 0 entries, got %d", len(trend))
	}
}

// --- GetUpcomingMeds with no schedule/timezone ---

func TestGetUpcomingMeds_NoScheduleOrTimezone(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov3", "cov3@test.com", "Cov3")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby3", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	// Create medication without schedule or timezone
	_, err = CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	meds, err := GetUpcomingMeds(db, baby.ID)
	if err != nil {
		t.Fatalf("GetUpcomingMeds: %v", err)
	}
	if len(meds) != 1 {
		t.Fatalf("expected 1 med, got %d", len(meds))
	}
	if meds[0].Schedule != nil {
		t.Errorf("expected nil schedule, got %v", meds[0].Schedule)
	}
	if meds[0].Timezone != nil {
		t.Errorf("expected nil timezone, got %v", meds[0].Timezone)
	}
}

func TestGetUpcomingMeds_EmptyData(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov4", "cov4@test.com", "Cov4")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby4", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	meds, err := GetUpcomingMeds(db, baby.ID)
	if err != nil {
		t.Fatalf("GetUpcomingMeds: %v", err)
	}
	if meds == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(meds) != 0 {
		t.Errorf("expected 0 meds, got %d", len(meds))
	}
}

// --- RecalculateFeedingCalories with zero affected rows ---

func TestRecalculateFeedingCalories_NoAffectedRows(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov5", "cov5@test.com", "Cov5")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby5", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	// Create only formula feedings (used_default_cal=false)
	vol := 120.0
	calDen := 20.0
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:00:00Z", "formula", &vol, &calDen, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding: %v", err)
	}

	count, err := RecalculateFeedingCalories(db, baby.ID, 80.0)
	if err != nil {
		t.Fatalf("RecalculateFeedingCalories: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 affected rows, got %d", count)
	}
}

func TestRecalculateFeedingCalories_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := RecalculateFeedingCalories(db, "baby-id", 80.0)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

// --- listMetricWithTZ error paths ---

func TestListFeedingsWithTZ_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov6", "cov6@test.com", "Cov6")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby6", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	badDate := "not-a-date"
	_, err = ListFeedingsWithTZ(db, baby.ID, &badDate, nil, nil, 50, time.UTC)
	if err == nil {
		t.Fatal("expected error for invalid from date")
	}
}

func TestListFeedingsWithTZ_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov7", "cov7@test.com", "Cov7")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby7", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	badDate := "not-a-date"
	_, err = ListFeedingsWithTZ(db, baby.ID, nil, &badDate, nil, 50, time.UTC)
	if err == nil {
		t.Fatal("expected error for invalid to date")
	}
}

// --- Stool scan with all nullable fields populated ---

func TestCreateStool_AllFieldsPopulated(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov8", "cov8@test.com", "Cov8")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby8", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	colorLabel := "green"
	consistency := "soft"
	volumeEstimate := "medium"
	notes := "some notes"
	stool, err := CreateStool(db, baby.ID, user.ID, "2025-07-01T10:00:00Z", 5, &colorLabel, &consistency, &volumeEstimate, &notes)
	if err != nil {
		t.Fatalf("CreateStool: %v", err)
	}

	if stool.ColorLabel == nil || *stool.ColorLabel != "green" {
		t.Errorf("expected color_label=green, got %v", stool.ColorLabel)
	}
	if stool.Consistency == nil || *stool.Consistency != "soft" {
		t.Errorf("expected consistency=soft, got %v", stool.Consistency)
	}
	if stool.VolumeEstimate == nil || *stool.VolumeEstimate != "medium" {
		t.Errorf("expected volume_estimate=medium, got %v", stool.VolumeEstimate)
	}
	if stool.Notes == nil || *stool.Notes != "some notes" {
		t.Errorf("expected notes='some notes', got %v", stool.Notes)
	}

	// Update to exercise updatedBy branch
	updatedStool, err := UpdateStool(db, baby.ID, stool.ID, user.ID, "2025-07-01T10:00:00Z", 4, &colorLabel, &consistency, &volumeEstimate, &notes)
	if err != nil {
		t.Fatalf("UpdateStool: %v", err)
	}
	if updatedStool.UpdatedBy == nil || *updatedStool.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%s, got %v", user.ID, updatedStool.UpdatedBy)
	}
}

// --- SkinObservation scan with all nullable fields ---

func TestCreateSkinObservation_AllFieldsPopulated(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov9", "cov9@test.com", "Cov9")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby9", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	jaundice := "mild_face"
	rashes := "diaper rash"
	bruising := "minor"
	notes := "observation notes"
	obs, err := CreateSkinObservation(db, baby.ID, user.ID, "2025-07-01T10:00:00Z", &jaundice, false, &rashes, &bruising, &notes)
	if err != nil {
		t.Fatalf("CreateSkinObservation: %v", err)
	}

	if obs.JaundiceLevel == nil || *obs.JaundiceLevel != "mild_face" {
		t.Errorf("expected jaundice_level=mild_face, got %v", obs.JaundiceLevel)
	}
	if obs.Rashes == nil || *obs.Rashes != "diaper rash" {
		t.Errorf("expected rashes='diaper rash', got %v", obs.Rashes)
	}
	if obs.Bruising == nil || *obs.Bruising != "minor" {
		t.Errorf("expected bruising=minor, got %v", obs.Bruising)
	}
	if obs.Notes == nil || *obs.Notes != "observation notes" {
		t.Errorf("expected notes, got %v", obs.Notes)
	}
}

// --- Feeding with all optional fields ---

func TestCreateFeeding_AllOptionalFieldsPopulated(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov10", "cov10@test.com", "Cov10")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby10", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	vol := 120.0
	calDen := 24.0
	durMin := 15
	notes := "feeding notes"
	feeding, err := CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:00:00Z", "formula", &vol, &calDen, &durMin, &notes, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding: %v", err)
	}

	if feeding.VolumeMl == nil || *feeding.VolumeMl != 120.0 {
		t.Errorf("expected volume_ml=120, got %v", feeding.VolumeMl)
	}
	if feeding.CalDensity == nil || *feeding.CalDensity != 24.0 {
		t.Errorf("expected cal_density=24, got %v", feeding.CalDensity)
	}
	if feeding.DurationMin == nil || *feeding.DurationMin != 15 {
		t.Errorf("expected duration_min=15, got %v", feeding.DurationMin)
	}
	if feeding.Notes == nil || *feeding.Notes != "feeding notes" {
		t.Errorf("expected notes, got %v", feeding.Notes)
	}
	if feeding.Calories == nil {
		t.Error("expected non-nil calories")
	}
}

// --- MedLog with all optional fields ---

func TestCreateMedLog_AllFieldsPopulated(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov11", "cov11@test.com", "Cov11")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby11", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	tz := "UTC"
	sched := `["08:00"]`
	med, err := CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	scheduledTime := "2025-07-01T08:00:00Z"
	givenAt := "2025-07-01T08:15:00Z"
	notes := "med log notes"
	medLog, err := CreateMedLog(db, baby.ID, med.ID, user.ID, &scheduledTime, &givenAt, false, nil, &notes)
	if err != nil {
		t.Fatalf("CreateMedLog: %v", err)
	}

	if medLog.ScheduledTime == nil {
		t.Error("expected non-nil scheduled_time")
	}
	if medLog.GivenAt == nil {
		t.Error("expected non-nil given_at")
	}
	if medLog.Notes == nil || *medLog.Notes != "med log notes" {
		t.Errorf("expected notes, got %v", medLog.Notes)
	}
}

func TestCreateMedLog_SkippedWithReason(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov12", "cov12@test.com", "Cov12")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby12", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	tz := "UTC"
	sched := `["08:00"]`
	med, err := CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	scheduledTime := "2025-07-01T08:00:00Z"
	skipReason := "baby asleep"
	medLog, err := CreateMedLog(db, baby.ID, med.ID, user.ID, &scheduledTime, nil, true, &skipReason, nil)
	if err != nil {
		t.Fatalf("CreateMedLog: %v", err)
	}

	if !medLog.Skipped {
		t.Error("expected skipped=true")
	}
	if medLog.SkipReason == nil || *medLog.SkipReason != "baby asleep" {
		t.Errorf("expected skip_reason='baby asleep', got %v", medLog.SkipReason)
	}
	if medLog.GivenAt != nil {
		t.Errorf("expected nil given_at for skipped dose, got %v", medLog.GivenAt)
	}
}

// --- DeleteAccount with real metric table anonymization ---

func TestDeleteAccount_AnonymizesFeedingsTable(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov13", "cov13@test.com", "Cov13")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby13", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	// Create another user who also parents this baby so baby is not deleted
	user2, err := UpsertUser(db, "google-cov13b", "cov13b@test.com", "Cov13b")
	if err != nil {
		t.Fatalf("UpsertUser2: %v", err)
	}
	_, err = db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)", baby.ID, user2.ID)
	if err != nil {
		t.Fatalf("insert bp: %v", err)
	}

	// Create a feeding logged by user
	_, err = CreateFeeding(db, baby.ID, user.ID, "2025-07-01T10:00:00Z", "formula", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding: %v", err)
	}

	// Delete account with feedings in anonymization list
	err = DeleteAccount(db, user.ID, []string{"feedings"})
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	// Verify the feeding's logged_by was anonymized
	var loggedBy string
	err = db.QueryRow("SELECT logged_by FROM feedings LIMIT 1").Scan(&loggedBy)
	if err != nil {
		t.Fatalf("query feeding: %v", err)
	}
	if loggedBy != "deleted_user" {
		t.Errorf("expected logged_by='deleted_user', got %q", loggedBy)
	}
}

// --- Missed medication: no schedule, invalid timezone, invalid JSON schedule ---

func TestGetActiveAlerts_MissedMedication_NoSchedule(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov14", "cov14@test.com", "Cov14")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby14", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	// Medication without schedule should not produce missed alerts
	_, err = CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	alerts, err := GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("GetActiveAlerts: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			t.Error("medication without schedule should not produce missed_medication alert")
		}
	}
}

func TestGetActiveAlerts_MissedMedication_InvalidTimezone(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov15", "cov15@test.com", "Cov15")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby15", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	invalidTz := "Invalid/Timezone"
	pastTime := time.Now().UTC().Add(-2 * time.Hour)
	schedTimeStr := pastTime.Format("15:04")
	sched := `["` + schedTimeStr + `"]`

	_, err = CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &invalidTz)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	// Should not error - just skip the medication with invalid timezone
	alerts, err := GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("GetActiveAlerts: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			t.Error("medication with invalid timezone should be skipped")
		}
	}
}

func TestGetActiveAlerts_MissedMedication_InvalidScheduleJSON(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov16", "cov16@test.com", "Cov16")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby16", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	tz := "UTC"
	badSched := `not-json`

	_, err = CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &badSched, &tz)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	// Should not error - just skip the medication with invalid JSON schedule
	alerts, err := GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("GetActiveAlerts: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			t.Error("medication with invalid schedule JSON should be skipped")
		}
	}
}

// --- LabResult with unit set ---

func TestGetLabTrends_WithUnit(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov17", "cov17@test.com", "Cov17")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby17", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	today := time.Now().UTC().Format("2006-01-02")

	// Lab result with unit
	unit := "mg/dL"
	_, err = CreateLabResult(db, baby.ID, user.ID, today+"T10:00:00Z", "bilirubin", "5.2", &unit, nil, nil)
	if err != nil {
		t.Fatalf("CreateLabResult: %v", err)
	}

	// Lab result without unit
	_, err = CreateLabResult(db, baby.ID, user.ID, today+"T14:00:00Z", "bilirubin", "4.8", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateLabResult: %v", err)
	}

	trends, err := GetLabTrends(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetLabTrends: %v", err)
	}

	entries := trends["bilirubin"]
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// First has unit
	if entries[0].Unit == nil || *entries[0].Unit != "mg/dL" {
		t.Errorf("expected unit=mg/dL, got %v", entries[0].Unit)
	}
	// Second has nil unit
	if entries[1].Unit != nil {
		t.Errorf("expected nil unit, got %v", entries[1].Unit)
	}
}

// --- Feeding types for chart_data FeedingByType branches ---

func TestGetFeedingDaily_AllFeedTypes(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov18", "cov18@test.com", "Cov18")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby18", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	today := time.Now().UTC().Format("2006-01-02")

	// Create feedings of each type
	vol := 100.0
	_, err = CreateFeeding(db, baby.ID, user.ID, today+"T08:00:00Z", "breast_milk", &vol, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding breast_milk: %v", err)
	}
	_, err = CreateFeeding(db, baby.ID, user.ID, today+"T10:00:00Z", "formula", &vol, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding formula: %v", err)
	}
	_, err = CreateFeeding(db, baby.ID, user.ID, today+"T12:00:00Z", "solid", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding solid: %v", err)
	}
	_, err = CreateFeeding(db, baby.ID, user.ID, today+"T14:00:00Z", "other", nil, nil, nil, nil, model.DefaultCalPerFeed)
	if err != nil {
		t.Fatalf("CreateFeeding other: %v", err)
	}

	series, err := GetFeedingDaily(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetFeedingDaily: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(series))
	}

	entry := series[0]
	if entry.FeedCount != 4 {
		t.Errorf("expected feed_count=4, got %d", entry.FeedCount)
	}
	if entry.ByType.BreastMilk != 1 {
		t.Errorf("expected breast_milk=1, got %d", entry.ByType.BreastMilk)
	}
	if entry.ByType.Formula != 1 {
		t.Errorf("expected formula=1, got %d", entry.ByType.Formula)
	}
	if entry.ByType.Solid != 1 {
		t.Errorf("expected solid=1, got %d", entry.ByType.Solid)
	}
	if entry.ByType.Other != 1 {
		t.Errorf("expected other=1, got %d", entry.ByType.Other)
	}
}

// --- WeightSeries with measurement_source null and non-null ---

func TestGetWeightSeries_MixedMeasurementSource(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov19", "cov19@test.com", "Cov19")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby19", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	src := "home_scale"
	_, err = CreateWeight(db, baby.ID, user.ID, today+"T08:00:00Z", 4.5, &src, nil)
	if err != nil {
		t.Fatalf("CreateWeight with source: %v", err)
	}
	_, err = CreateWeight(db, baby.ID, user.ID, today+"T16:00:00Z", 4.55, nil, nil)
	if err != nil {
		t.Fatalf("CreateWeight without source: %v", err)
	}

	series, err := GetWeightSeries(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetWeightSeries: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected 2, got %d", len(series))
	}
	if series[0].MeasurementSource == nil || *series[0].MeasurementSource != "home_scale" {
		t.Errorf("expected measurement_source=home_scale, got %v", series[0].MeasurementSource)
	}
	if series[1].MeasurementSource != nil {
		t.Errorf("expected nil measurement_source, got %v", series[1].MeasurementSource)
	}
}

// --- StoolWithPhotos ---

func TestCreateStoolWithPhotos_PhotoKeysSet(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov20", "cov20@test.com", "Cov20")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby20", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	colorLabel := "brown"
	photoKeys := "photo1.jpg,photo2.jpg"
	stool, err := CreateStoolWithPhotos(db, baby.ID, user.ID, "2025-07-01T10:00:00Z", 5, &colorLabel, nil, nil, &photoKeys, nil)
	if err != nil {
		t.Fatalf("CreateStoolWithPhotos: %v", err)
	}

	if stool.PhotoKeys == nil || *stool.PhotoKeys != "photo1.jpg,photo2.jpg" {
		t.Errorf("expected photo_keys='photo1.jpg,photo2.jpg', got %v", stool.PhotoKeys)
	}
}

// --- Medication with schedule and timezone ---

func TestCreateMedication_WithScheduleAndTimezone(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov21", "cov21@test.com", "Cov21")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby21", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	tz := "America/New_York"
	sched := `["08:00","20:00"]`
	med, err := CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", &sched, &tz)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	if med.Schedule == nil || *med.Schedule != `["08:00","20:00"]` {
		t.Errorf("expected schedule, got %v", med.Schedule)
	}
	if med.Timezone == nil || *med.Timezone != "America/New_York" {
		t.Errorf("expected timezone, got %v", med.Timezone)
	}
	if !med.Active {
		t.Error("expected active=true by default")
	}
}

func TestCreateMedication_WithoutScheduleOrTimezone(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov22", "cov22@test.com", "Cov22")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby22", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	med, err := CreateMedication(db, baby.ID, user.ID, "VitD", "400IU", "once_daily", nil, nil)
	if err != nil {
		t.Fatalf("CreateMedication: %v", err)
	}

	if med.Schedule != nil {
		t.Errorf("expected nil schedule, got %v", med.Schedule)
	}
	if med.Timezone != nil {
		t.Errorf("expected nil timezone, got %v", med.Timezone)
	}
}

// --- DiaperDaily with empty date range ---

func TestGetDiaperDaily_EmptyData(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov23", "cov23@test.com", "Cov23")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby23", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	series, err := GetDiaperDaily(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetDiaperDaily: %v", err)
	}
	if series == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(series) != 0 {
		t.Errorf("expected 0 entries, got %d", len(series))
	}
}

// --- Temperature series empty ---

func TestGetTemperatureSeries_EmptyData(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov24", "cov24@test.com", "Cov24")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby24", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	series, err := GetTemperatureSeries(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetTemperatureSeries: %v", err)
	}
	if series == nil {
		t.Error("expected non-nil empty slice")
	}
}

// --- AbdomenGirth empty ---

func TestGetAbdomenGirthSeries_EmptyData(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov25", "cov25@test.com", "Cov25")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby25", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	series, err := GetAbdomenGirthSeries(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetAbdomenGirthSeries: %v", err)
	}
	if series == nil {
		t.Error("expected non-nil empty slice")
	}
}

// --- StoolColorSeries empty ---

func TestGetStoolColorSeries_EmptyData(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google-cov26", "cov26@test.com", "Cov26")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "CovBaby26", "female", "2025-01-01", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby: %v", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	series, err := GetStoolColorSeries(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetStoolColorSeries: %v", err)
	}
	if series == nil {
		t.Error("expected non-nil empty slice")
	}
}

// --- Closed DB error path tests for chart_data functions ---

func TestGetFeedingDaily_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetFeedingDaily(db, "b1", "2025-07-01", "2025-07-02", time.UTC)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestGetDiaperDaily_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetDiaperDaily(db, "b1", "2025-07-01", "2025-07-02", time.UTC)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestGetTemperatureSeries_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetTemperatureSeries(db, "b1", "2025-07-01", "2025-07-02", time.UTC)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestGetWeightSeries_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetWeightSeries(db, "b1", "2025-07-01", "2025-07-02", time.UTC)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestGetAbdomenGirthSeries_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetAbdomenGirthSeries(db, "b1", "2025-07-01", "2025-07-02", time.UTC)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestGetStoolColorSeries_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetStoolColorSeries(db, "b1", "2025-07-01", "2025-07-02", time.UTC)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestGetLabTrends_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetLabTrends(db, "b1", "2025-07-01", "2025-07-02", time.UTC)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

// --- Closed DB error paths for dashboard functions ---

func TestGetDashboardSummary_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetDashboardSummary(db, "b1", "2025-07-01", "2025-07-02", time.UTC)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestGetStoolColorTrend_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetStoolColorTrend(db, "b1")
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestGetUpcomingMeds_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetUpcomingMeds(db, "b1")
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestGetActiveAlerts_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetActiveAlerts(db, "b1")
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestIsDoseCovered_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := IsDoseCovered(db, "med-id", time.Now())
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

// --- Closed DB for other low-coverage functions ---

func TestCreateGeneralNote_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := CreateGeneralNote(db, "b1", "u1", "2025-07-01T10:00:00Z", "content", nil, nil)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestDeletePushSubscription_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	err := DeletePushSubscription(db, "u1", "sub-id")
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestGetPushSubscriptionsByUserID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := GetPushSubscriptionsByUserID(db, "u1")
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestCreatePhotoUpload_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := CreatePhotoUpload(db, "b1", "r2key", "thumb")
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestCreateMedication_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := CreateMedication(db, "b1", "u1", "Med", "10mg", "once_daily", nil, nil)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestListMedications_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := ListMedications(db, "b1")
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestCreateMedLog_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := CreateMedLog(db, "b1", "m1", "u1", nil, nil, false, nil, nil)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestListMedLogs_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()
	_, err := ListMedLogs(db, "b1", nil, nil, nil, nil, 50, time.UTC)
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

// --- helpers ---

type mockResult struct {
	rowsAffected int64
}

func (m *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (m *mockResult) RowsAffected() (int64, error) { return m.rowsAffected, nil }

