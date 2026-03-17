package store_test

import (
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// --- Acholic stool alert tests ---

func TestGetActiveAlerts_AcholicStool_TriggersOnLowRating(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateStool(db, baby.ID, user.ID, ts, 2, strPtr("clay"), nil, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].AlertType != "acholic_stool" {
		t.Errorf("expected alert_type=acholic_stool, got %s", alerts[0].AlertType)
	}
	if v, ok := alerts[0].Value.(float64); !ok || int(v) != 2 {
		// Value might be int when returned from store directly (not JSON)
		if vi, ok2 := alerts[0].Value.(int); !ok2 || vi != 2 {
			t.Errorf("expected value=2, got %v (type %T)", alerts[0].Value, alerts[0].Value)
		}
	}
}

func TestGetActiveAlerts_AcholicStool_Rating3Triggers(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateStool(db, baby.ID, user.ID, ts, 3, strPtr("pale_yellow"), nil, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, a := range alerts {
		if a.AlertType == "acholic_stool" {
			found = true
		}
	}
	if !found {
		t.Error("expected acholic_stool alert for rating 3")
	}
}

func TestGetActiveAlerts_AcholicStool_ClearedByPigmented(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts1 := time.Now().UTC().Add(-1 * time.Hour).Format(model.DateTimeFormat)
	ts2 := time.Now().UTC().Format(model.DateTimeFormat)

	store.CreateStool(db, baby.ID, user.ID, ts1, 2, strPtr("clay"), nil, nil, nil)
	store.CreateStool(db, baby.ID, user.ID, ts2, 5, strPtr("light_green"), nil, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "acholic_stool" {
			t.Error("acholic_stool alert should be cleared by pigmented stool (rating 5)")
		}
	}
}

func TestGetActiveAlerts_AcholicStool_Rating4Clears(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts1 := time.Now().UTC().Add(-1 * time.Hour).Format(model.DateTimeFormat)
	ts2 := time.Now().UTC().Format(model.DateTimeFormat)

	store.CreateStool(db, baby.ID, user.ID, ts1, 1, strPtr("white"), nil, nil, nil)
	store.CreateStool(db, baby.ID, user.ID, ts2, 4, strPtr("yellow"), nil, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "acholic_stool" {
			t.Error("acholic_stool alert should be cleared by rating 4")
		}
	}
}

// --- Fever alert tests ---

func TestGetActiveAlerts_Fever_RectalThreshold(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateTemperature(db, baby.ID, user.ID, ts, 38.0, "rectal", nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, a := range alerts {
		if a.AlertType == "fever" {
			found = true
			if a.Method == nil || *a.Method != "rectal" {
				t.Errorf("expected method=rectal, got %v", a.Method)
			}
		}
	}
	if !found {
		t.Error("expected fever alert for rectal 38.0")
	}
}

func TestGetActiveAlerts_Fever_RectalSubThreshold(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateTemperature(db, baby.ID, user.ID, ts, 37.9, "rectal", nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "fever" {
			t.Error("should not have fever alert for rectal 37.9")
		}
	}
}

func TestGetActiveAlerts_Fever_AxillaryThreshold(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateTemperature(db, baby.ID, user.ID, ts, 37.5, "axillary", nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, a := range alerts {
		if a.AlertType == "fever" {
			found = true
			if a.Method == nil || *a.Method != "axillary" {
				t.Errorf("expected method=axillary, got %v", a.Method)
			}
		}
	}
	if !found {
		t.Error("expected fever alert for axillary 37.5")
	}
}

func TestGetActiveAlerts_Fever_EarThreshold(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateTemperature(db, baby.ID, user.ID, ts, 38.0, "ear", nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, a := range alerts {
		if a.AlertType == "fever" {
			found = true
		}
	}
	if !found {
		t.Error("expected fever alert for ear 38.0")
	}
}

func TestGetActiveAlerts_Fever_ForeheadThreshold(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateTemperature(db, baby.ID, user.ID, ts, 37.5, "forehead", nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, a := range alerts {
		if a.AlertType == "fever" {
			found = true
		}
	}
	if !found {
		t.Error("expected fever alert for forehead 37.5")
	}
}

func TestGetActiveAlerts_Fever_ClearedBySubThreshold(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts1 := time.Now().UTC().Add(-1 * time.Hour).Format(model.DateTimeFormat)
	ts2 := time.Now().UTC().Format(model.DateTimeFormat)

	store.CreateTemperature(db, baby.ID, user.ID, ts1, 38.5, "rectal", nil)
	store.CreateTemperature(db, baby.ID, user.ID, ts2, 37.2, "axillary", nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "fever" {
			t.Error("fever should be cleared; most recent temp (axillary 37.2) is sub-threshold")
		}
	}
}

func TestGetActiveAlerts_Fever_OnlyMostRecentMatters(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts1 := time.Now().UTC().Add(-2 * time.Hour).Format(model.DateTimeFormat)
	ts2 := time.Now().UTC().Format(model.DateTimeFormat)

	store.CreateTemperature(db, baby.ID, user.ID, ts1, 37.8, "axillary", nil)
	store.CreateTemperature(db, baby.ID, user.ID, ts2, 37.8, "rectal", nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "fever" {
			t.Error("no fever alert expected; most recent temp (rectal 37.8) is sub-threshold for rectal")
		}
	}
}

// --- Jaundice worsening alert tests ---

func TestGetActiveAlerts_Jaundice_SevereLevel(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateSkinObservation(db, baby.ID, user.ID, ts, strPtr("severe_limbs_and_trunk"), false, nil, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, a := range alerts {
		if a.AlertType == "jaundice_worsening" {
			found = true
		}
	}
	if !found {
		t.Error("expected jaundice_worsening alert for severe_limbs_and_trunk")
	}
}

func TestGetActiveAlerts_Jaundice_ScleralIcterus(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateSkinObservation(db, baby.ID, user.ID, ts, strPtr("none"), true, nil, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, a := range alerts {
		if a.AlertType == "jaundice_worsening" {
			found = true
		}
	}
	if !found {
		t.Error("expected jaundice_worsening alert for scleral_icterus")
	}
}

func TestGetActiveAlerts_Jaundice_BothSevereAndScleral(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateSkinObservation(db, baby.ID, user.ID, ts, strPtr("severe_limbs_and_trunk"), true, nil, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, a := range alerts {
		if a.AlertType == "jaundice_worsening" {
			found = true
			if v, ok := a.Value.(string); ok {
				if v != "severe_limbs_and_trunk+scleral_icterus" {
					t.Errorf("expected combined value, got %s", v)
				}
			}
		}
	}
	if !found {
		t.Error("expected jaundice_worsening alert")
	}
}

func TestGetActiveAlerts_Jaundice_ClearedByNormalObservation(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts1 := time.Now().UTC().Add(-1 * time.Hour).Format(model.DateTimeFormat)
	ts2 := time.Now().UTC().Format(model.DateTimeFormat)

	store.CreateSkinObservation(db, baby.ID, user.ID, ts1, strPtr("severe_limbs_and_trunk"), true, nil, nil, nil)
	store.CreateSkinObservation(db, baby.ID, user.ID, ts2, strPtr("none"), false, nil, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "jaundice_worsening" {
			t.Error("jaundice alert should be cleared by normal observation")
		}
	}
}

func TestGetActiveAlerts_Jaundice_MildDoesNotTrigger(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	ts := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateSkinObservation(db, baby.ID, user.ID, ts, strPtr("mild_face"), false, nil, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "jaundice_worsening" {
			t.Error("mild_face should not trigger jaundice_worsening")
		}
	}
}

// --- Missed medication alert tests ---

func TestGetActiveAlerts_MissedMedication_TriggersWhenNoDoseLogged(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	pastTime := time.Now().UTC().Add(-2 * time.Hour)
	schedTimeStr := pastTime.Format("15:04")
	sched := `["` + schedTimeStr + `"]`
	store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			found = true
		}
	}
	if !found {
		t.Error("expected missed_medication alert when no dose logged")
	}
}

func TestGetActiveAlerts_MissedMedication_NotTriggeredWhenDoseLogged(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	pastTime := time.Now().UTC().Add(-2 * time.Hour)
	schedTimeStr := pastTime.Format("15:04")
	sched := `["` + schedTimeStr + `"]`
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz)

	givenAt := pastTime.Format(model.DateTimeFormat)
	scheduledTime := pastTime.Format(model.DateTimeFormat)
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, &scheduledTime, &givenAt, false, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			t.Error("missed_medication alert should not fire when dose is logged")
		}
	}
}

func TestGetActiveAlerts_MissedMedication_NotTriggeredWithin30Min(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	pastTime := time.Now().UTC().Add(-20 * time.Minute)
	schedTimeStr := pastTime.Format("15:04")
	sched := `["` + schedTimeStr + `"]`
	store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			t.Error("missed_medication should not trigger for dose only 20 min past due")
		}
	}
}

func TestGetActiveAlerts_MissedMedication_InactiveMedSkipped(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	pastTime := time.Now().UTC().Add(-2 * time.Hour)
	schedTimeStr := pastTime.Format("15:04")
	sched := `["` + schedTimeStr + `"]`
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz)

	inactive := false
	store.UpdateMedication(db, baby.ID, med.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, &inactive)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			t.Error("inactive medication should not produce missed_medication alert")
		}
	}
}

func TestGetActiveAlerts_MissedMedication_GivenDoseCoverage(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	// Schedule 2 hours ago (>30 min past due)
	pastTime := time.Now().UTC().Add(-2 * time.Hour)
	schedTimeStr := pastTime.Format("15:04")
	sched := `["` + schedTimeStr + `"]`
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz)

	// Log a given dose with given_at within +/-30 min of the scheduled time
	givenAt := pastTime.Add(10 * time.Minute).Format(model.DateTimeFormat)
	scheduledTime := pastTime.Format(model.DateTimeFormat)
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, &scheduledTime, &givenAt, false, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			t.Error("given dose with given_at within +/-30 min of scheduled time should suppress missed_medication alert")
		}
	}
}

// --- IsDoseCovered shared utility tests ---

func TestIsDoseCovered_GivenDoseWithinWindow(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	sched := `["08:00"]`
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz)

	scheduledUTC := time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC)
	givenAt := time.Date(2026, 3, 18, 8, 15, 0, 0, time.UTC).Format(model.DateTimeFormat)
	schedStr := scheduledUTC.Format(model.DateTimeFormat)
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, &schedStr, &givenAt, false, nil, nil)

	covered, err := store.IsDoseCovered(db, med.ID, scheduledUTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !covered {
		t.Error("expected dose to be covered (given within +15 min)")
	}
}

func TestIsDoseCovered_SkippedDoseWithinWindow(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	sched := `["08:00"]`
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz)

	// Use "now" as the scheduled time so created_at (set by SQLite to NOW) falls within +/-30 min
	scheduledUTC := time.Now().UTC()
	schedStr := scheduledUTC.Format(model.DateTimeFormat)
	reason := "baby asleep"
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, &schedStr, nil, true, &reason, nil)

	covered, err := store.IsDoseCovered(db, med.ID, scheduledUTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !covered {
		t.Error("expected dose to be covered (skipped log created within window)")
	}
}

func TestIsDoseCovered_NoDoseLogged(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	sched := `["08:00"]`
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz)
	_ = baby // suppress unused

	scheduledUTC := time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC)

	covered, err := store.IsDoseCovered(db, med.ID, scheduledUTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if covered {
		t.Error("expected dose to NOT be covered when no log exists")
	}
}

func TestIsDoseCovered_DoseOutsideWindow(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	sched := `["08:00"]`
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz)

	scheduledUTC := time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC)
	givenAt := time.Date(2026, 3, 18, 8, 45, 0, 0, time.UTC).Format(model.DateTimeFormat)
	schedStr := scheduledUTC.Format(model.DateTimeFormat)
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, &schedStr, &givenAt, false, nil, nil)

	covered, err := store.IsDoseCovered(db, med.ID, scheduledUTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if covered {
		t.Error("expected dose to NOT be covered when given_at is outside +/-30 min window")
	}
}

// --- No data = no alerts ---

func TestGetActiveAlerts_NoData_EmptyArray(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alerts == nil {
		t.Error("expected empty array, got nil")
	}
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alerts))
	}
}

// --- FeverThreshold tests ---

func TestFeverThreshold(t *testing.T) {
	t.Parallel()
	tests := []struct {
		method    string
		threshold float64
	}{
		{"rectal", 38.0},
		{"axillary", 37.5},
		{"ear", 38.0},
		{"forehead", 37.5},
		{"unknown", 38.0},
	}
	for _, tt := range tests {
		if got := store.FeverThreshold(tt.method); got != tt.threshold {
			t.Errorf("FeverThreshold(%s) = %f, want %f", tt.method, got, tt.threshold)
		}
	}
}
