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
	store.CreateStool(db, baby.ID, user.ID, ts, 2, strPtr("clay"), nil, nil, nil, nil)

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
	store.CreateStool(db, baby.ID, user.ID, ts, 3, strPtr("pale_yellow"), nil, nil, nil, nil)

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

	store.CreateStool(db, baby.ID, user.ID, ts1, 2, strPtr("clay"), nil, nil, nil, nil)
	store.CreateStool(db, baby.ID, user.ID, ts2, 5, strPtr("light_green"), nil, nil, nil, nil)

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

	store.CreateStool(db, baby.ID, user.ID, ts1, 1, strPtr("white"), nil, nil, nil, nil)
	store.CreateStool(db, baby.ID, user.ID, ts2, 4, strPtr("yellow"), nil, nil, nil, nil)

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
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, nil, nil)

	// Backdate created_at so the scheduled dose falls after creation
	createdBefore := pastTime.Add(-1 * time.Hour).Format(model.DateTimeFormat)
	db.Exec("UPDATE medications SET created_at = ? WHERE id = ?", createdBefore, med.ID)

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
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, nil, nil)

	// Backdate created_at so the scheduled dose falls after creation
	createdBefore := pastTime.Add(-1 * time.Hour).Format(model.DateTimeFormat)
	db.Exec("UPDATE medications SET created_at = ? WHERE id = ?", createdBefore, med.ID)

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
	store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, nil, nil)

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
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, nil, nil)

	// Backdate created_at so the scheduled dose falls after creation
	createdBefore := pastTime.Add(-1 * time.Hour).Format(model.DateTimeFormat)
	db.Exec("UPDATE medications SET created_at = ? WHERE id = ?", createdBefore, med.ID)

	inactive := false
	store.UpdateMedication(db, baby.ID, med.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, &inactive, nil, nil)

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
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, nil, nil)

	// Backdate created_at so the scheduled dose falls after creation
	createdBefore := pastTime.Add(-1 * time.Hour).Format(model.DateTimeFormat)
	db.Exec("UPDATE medications SET created_at = ? WHERE id = ?", createdBefore, med.ID)

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

func TestGetActiveAlerts_MissedMedication_NotTriggeredForDosesBeforeCreation(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	// Schedule a twice-daily med with times that are already past.
	// E.g. if it's 15:00 now, schedule at 08:00 and 12:00.
	// Both are >30 min past due, but the medication was JUST created,
	// so neither should trigger a missed dose alert.
	t1 := time.Now().UTC().Add(-7 * time.Hour)
	t2 := time.Now().UTC().Add(-3 * time.Hour)
	sched := `["` + t1.Format("15:04") + `","` + t2.Format("15:04") + `"]`
	store.CreateMedication(db, baby.ID, user.ID, "NewMed", "5mg", "twice_daily", &sched, &tz, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			t.Error("newly created medication should not trigger missed_medication alert for doses scheduled before creation")
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
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, nil, nil)

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
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, nil, nil)

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
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, nil, nil)
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

func TestIsDoseCovered_DoseOutsideWindowNoScheduledTime(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	sched := `["08:00"]`
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, nil, nil)

	scheduledUTC := time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC)
	givenAt := time.Date(2026, 3, 18, 8, 45, 0, 0, time.UTC).Format(model.DateTimeFormat)
	// Ad-hoc dose with no scheduled_time — only given_at is set
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, &givenAt, false, nil, nil)

	covered, err := store.IsDoseCovered(db, med.ID, scheduledUTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if covered {
		t.Error("expected dose to NOT be covered when no scheduled_time and given_at is outside window")
	}
}

func TestIsDoseCovered_LateDoseWithMatchingScheduledTime(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	sched := `["08:00"]`
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, nil, nil)

	scheduledUTC := time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC)
	// Parent logs dose 2 hours late, but scheduled_time matches the dose slot
	givenAt := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC).Format(model.DateTimeFormat)
	schedStr := scheduledUTC.Format(model.DateTimeFormat)
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, &schedStr, &givenAt, false, nil, nil)

	covered, err := store.IsDoseCovered(db, med.ID, scheduledUTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !covered {
		t.Error("expected dose to be covered when scheduled_time matches even if given_at is late")
	}
}

func TestIsDoseCovered_SkippedDoseWithMatchingScheduledTime(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	sched := `["08:00"]`
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "TestMed", "10mg", "once_daily", &sched, &tz, nil, nil)

	scheduledUTC := time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC)
	schedStr := scheduledUTC.Format(model.DateTimeFormat)
	reason := "vomited"
	// Skipped dose logged hours later, but scheduled_time matches
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, &schedStr, nil, true, &reason, nil)

	covered, err := store.IsDoseCovered(db, med.ID, scheduledUTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !covered {
		t.Error("expected dose to be covered when skipped log has matching scheduled_time")
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

// --- Every X Days missed medication tests ---

func TestMissedMedAlert_EveryXDays_NotYetDue(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	intervalDays := 5
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "VitA", "5000IU", "every_x_days", nil, &tz, &intervalDays, nil)

	// Log a dose today — not due again for 5 days
	now := time.Now().UTC().Format(model.DateTimeFormat)
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, &now, false, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("GetActiveAlerts failed: %v", err)
	}

	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			t.Errorf("should not have missed_medication alert when dose was just given")
		}
	}
}

func TestMissedMedAlert_EveryXDays_DueToday_NotMissedYet(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	intervalDays := 3
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "VitA", "5000IU", "every_x_days", nil, &tz, &intervalDays, nil)

	// Log a dose exactly 3 days ago — due today, but day hasn't ended, so not missed
	threeDaysAgo := time.Now().UTC().AddDate(0, 0, -3).Format(model.DateTimeFormat)
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, &threeDaysAgo, false, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("GetActiveAlerts failed: %v", err)
	}

	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			t.Errorf("should not have missed_medication alert for med due today (day not over)")
		}
	}
}

func TestMissedMedAlert_EveryXDays_Overdue_NoLog(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	intervalDays := 2
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "VitA", "5000IU", "every_x_days", nil, &tz, &intervalDays, nil)

	// Log a dose 4 days ago, interval is 2 — was due 2 days ago (full day passed), no log for that day
	fourDaysAgo := time.Now().UTC().AddDate(0, 0, -4).Format(model.DateTimeFormat)
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, &fourDaysAgo, false, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("GetActiveAlerts failed: %v", err)
	}

	hasMissed := false
	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			hasMissed = true
		}
	}
	if !hasMissed {
		t.Error("expected missed_medication alert for overdue every_x_days med with no log")
	}
}

func TestMissedMedAlert_EveryXDays_Overdue_WithLog(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	intervalDays := 2
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "VitA", "5000IU", "every_x_days", nil, &tz, &intervalDays, nil)

	// Log a dose 4 days ago
	fourDaysAgo := time.Now().UTC().AddDate(0, 0, -4).Format(model.DateTimeFormat)
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, &fourDaysAgo, false, nil, nil)

	// Also log a dose on the due day (2 days ago) — should suppress alert
	twoDaysAgo := time.Now().UTC().AddDate(0, 0, -2).Format(model.DateTimeFormat)
	store.CreateMedLog(db, baby.ID, med.ID, user.ID, nil, &twoDaysAgo, false, nil, nil)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("GetActiveAlerts failed: %v", err)
	}

	for _, a := range alerts {
		if a.AlertType == "missed_medication" {
			t.Errorf("should NOT have missed_medication alert when dose was logged on due day")
		}
	}
}

// --- Missed medication alert includes medication info ---

func TestGetActiveAlerts_MissedMedication_IncludesMedicationInfo(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	pastTime := time.Now().UTC().Add(-2 * time.Hour)
	schedTimeStr := pastTime.Format("15:04")
	sched := `["` + schedTimeStr + `"]`
	med, _ := store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "once_daily", &sched, &tz, nil, nil)

	// Backdate created_at so the scheduled dose falls after creation
	createdBefore := pastTime.Add(-1 * time.Hour).Format(model.DateTimeFormat)
	db.Exec("UPDATE medications SET created_at = ? WHERE id = ?", createdBefore, med.ID)

	alerts, err := store.GetActiveAlerts(db, baby.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var missedAlert *store.Alert
	for i, a := range alerts {
		if a.AlertType == "missed_medication" {
			missedAlert = &alerts[i]
			break
		}
	}
	if missedAlert == nil {
		t.Fatal("expected missed_medication alert")
	}
	if missedAlert.MedicationID != med.ID {
		t.Errorf("MedicationID = %q, want %q", missedAlert.MedicationID, med.ID)
	}
	if missedAlert.MedicationName != "Ursodiol" {
		t.Errorf("MedicationName = %q, want %q", missedAlert.MedicationName, "Ursodiol")
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
