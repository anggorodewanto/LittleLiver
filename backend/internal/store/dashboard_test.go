package store_test

import (
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestGetDashboardSummary_EmptyData(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	summary, err := store.GetDashboardSummary(db, baby.ID, today, today)
	if err != nil {
		t.Fatalf("GetDashboardSummary failed: %v", err)
	}

	if summary.TotalFeeds != 0 {
		t.Errorf("expected 0 total_feeds, got %d", summary.TotalFeeds)
	}
	if summary.TotalCalories != 0 {
		t.Errorf("expected 0 total_calories, got %f", summary.TotalCalories)
	}
	if summary.WetDiapers != 0 {
		t.Errorf("expected 0 wet_diapers, got %d", summary.WetDiapers)
	}
	if summary.Stools != 0 {
		t.Errorf("expected 0 stools, got %d", summary.Stools)
	}
	if summary.ColorIndicator != nil {
		t.Errorf("expected nil color_indicator, got %v", summary.ColorIndicator)
	}
	if summary.LastTemp != nil {
		t.Errorf("expected nil last_temp, got %v", summary.LastTemp)
	}
	if summary.LastWeight != nil {
		t.Errorf("expected nil last_weight, got %v", summary.LastWeight)
	}
}

func TestGetDashboardSummary_WithData(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	ts := today + "T10:00:00Z"
	ts2 := today + "T14:00:00Z"

	// Insert 2 feedings with calories
	vol := 120.0
	calDen := 20.0 // 20 kcal/oz
	store.CreateFeeding(db, baby.ID, user.ID, ts, "formula", &vol, &calDen, nil, nil, 67.0)
	store.CreateFeeding(db, baby.ID, user.ID, ts2, "breast_milk", nil, nil, nil, nil, 67.0)

	// Insert 3 urine entries
	store.CreateUrine(db, baby.ID, user.ID, ts, nil, nil)
	store.CreateUrine(db, baby.ID, user.ID, ts2, nil, nil)
	store.CreateUrine(db, baby.ID, user.ID, today+"T16:00:00Z", nil, nil)

	// Insert 2 stools - most recent should be green
	yellow := "yellow"
	green := "green"
	store.CreateStool(db, baby.ID, user.ID, ts, 3, &yellow, nil, nil, nil)
	store.CreateStool(db, baby.ID, user.ID, ts2, 5, &green, nil, nil, nil)

	// Insert temperature and weight
	store.CreateTemperature(db, baby.ID, user.ID, ts, 37.2, "rectal", nil)
	store.CreateWeight(db, baby.ID, user.ID, ts, 4.5, nil, nil)

	summary, err := store.GetDashboardSummary(db, baby.ID, today, today)
	if err != nil {
		t.Fatalf("GetDashboardSummary failed: %v", err)
	}

	if summary.TotalFeeds != 2 {
		t.Errorf("expected 2 total_feeds, got %d", summary.TotalFeeds)
	}
	// formula: 120ml * (20 kcal/oz / 29.5735 ml/oz) ~= 81.13, breast_milk default: 67.0
	expectedCal := 120.0*(20.0/29.5735) + 67.0
	if summary.TotalCalories < expectedCal-1.0 || summary.TotalCalories > expectedCal+1.0 {
		t.Errorf("expected total_calories ~%.1f, got %.1f", expectedCal, summary.TotalCalories)
	}
	if summary.WetDiapers != 3 {
		t.Errorf("expected 3 wet_diapers, got %d", summary.WetDiapers)
	}
	if summary.Stools != 2 {
		t.Errorf("expected 2 stools, got %d", summary.Stools)
	}
	if summary.ColorIndicator == nil || *summary.ColorIndicator != 3 {
		t.Errorf("expected worst_stool_color=3, got %v", summary.ColorIndicator)
	}
	if summary.LastTemp == nil || *summary.LastTemp != 37.2 {
		t.Errorf("expected last_temp=37.2, got %v", summary.LastTemp)
	}
	if summary.LastWeight == nil || *summary.LastWeight != 4.5 {
		t.Errorf("expected last_weight=4.5, got %v", summary.LastWeight)
	}
}

func TestGetDashboardSummary_DateFiltering(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Feeding yesterday - should not appear when querying today
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	today := time.Now().UTC().Format("2006-01-02")
	tsYesterday := yesterday + "T10:00:00Z"
	tsToday := today + "T10:00:00Z"

	vol := 100.0
	calDen := 0.67
	store.CreateFeeding(db, baby.ID, user.ID, tsYesterday, "formula", &vol, &calDen, nil, nil, 67.0)
	store.CreateFeeding(db, baby.ID, user.ID, tsToday, "formula", &vol, &calDen, nil, nil, 67.0)

	// Query today only
	summary, err := store.GetDashboardSummary(db, baby.ID, today, today)
	if err != nil {
		t.Fatalf("GetDashboardSummary failed: %v", err)
	}

	if summary.TotalFeeds != 1 {
		t.Errorf("expected 1 total_feeds for today, got %d", summary.TotalFeeds)
	}
}

func TestGetDashboardSummary_LastTempAndWeight_IgnoreDateRange(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Insert temp and weight from a week ago
	weekAgo := time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02")
	tsWeekAgo := weekAgo + "T10:00:00Z"
	today := time.Now().UTC().Format("2006-01-02")

	store.CreateTemperature(db, baby.ID, user.ID, tsWeekAgo, 38.5, "rectal", nil)
	store.CreateWeight(db, baby.ID, user.ID, tsWeekAgo, 5.2, nil, nil)

	// Query today - should still return last temp and weight
	summary, err := store.GetDashboardSummary(db, baby.ID, today, today)
	if err != nil {
		t.Fatalf("GetDashboardSummary failed: %v", err)
	}

	if summary.LastTemp == nil || *summary.LastTemp != 38.5 {
		t.Errorf("expected last_temp=38.5 regardless of date range, got %v", summary.LastTemp)
	}
	if summary.LastWeight == nil || *summary.LastWeight != 5.2 {
		t.Errorf("expected last_weight=5.2 regardless of date range, got %v", summary.LastWeight)
	}
}

func TestGetStoolColorTrend_Always7Days(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Insert stools over the last 10 days
	now := time.Now().UTC()
	for i := 0; i < 10; i++ {
		day := now.AddDate(0, 0, -i)
		ts := day.Format("2006-01-02") + "T12:00:00Z"
		label := "yellow"
		store.CreateStool(db, baby.ID, user.ID, ts, 3, &label, nil, nil, nil)
	}

	trend, err := store.GetStoolColorTrend(db, baby.ID)
	if err != nil {
		t.Fatalf("GetStoolColorTrend failed: %v", err)
	}

	// Should only get entries from last 7 days
	if len(trend) > 7 {
		t.Errorf("expected at most 7 trend entries, got %d", len(trend))
	}
	if len(trend) < 7 {
		t.Errorf("expected 7 trend entries, got %d", len(trend))
	}
}

func TestGetStoolColorTrend_ReturnsDateAndColor(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	ts := today + "T12:00:00Z"
	green := "green"
	store.CreateStool(db, baby.ID, user.ID, ts, 5, &green, nil, nil, nil)

	trend, err := store.GetStoolColorTrend(db, baby.ID)
	if err != nil {
		t.Fatalf("GetStoolColorTrend failed: %v", err)
	}

	if len(trend) == 0 {
		t.Fatal("expected at least 1 trend entry")
	}

	found := false
	for _, entry := range trend {
		if entry.Date == today {
			found = true
			if entry.Color != "green" {
				t.Errorf("expected color=green, got %q", entry.Color)
			}
			if entry.ColorRating != 5 {
				t.Errorf("expected color_rating=5, got %d", entry.ColorRating)
			}
		}
	}
	if !found {
		t.Errorf("expected to find entry for today %s", today)
	}
}

func TestGetUpcomingMeds_ExcludesInactive(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "America/New_York"
	sched := `["08:00","20:00"]`

	// Create active medication
	store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", &sched, &tz)
	// Create inactive medication
	med2, _ := store.CreateMedication(db, baby.ID, user.ID, "VitD", "400IU", "once_daily", &sched, &tz)
	inactive := false
	store.UpdateMedication(db, baby.ID, med2.ID, user.ID, "VitD", "400IU", "once_daily", &sched, &tz, &inactive)

	meds, err := store.GetUpcomingMeds(db, baby.ID)
	if err != nil {
		t.Fatalf("GetUpcomingMeds failed: %v", err)
	}

	if len(meds) != 1 {
		t.Fatalf("expected 1 active med, got %d", len(meds))
	}
	if meds[0].Name != "Ursodiol" {
		t.Errorf("expected Ursodiol, got %s", meds[0].Name)
	}
}

func TestGetUpcomingMeds_ReturnsScheduleAndTimezone(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "America/New_York"
	sched := `["08:00","20:00"]`
	store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", &sched, &tz)

	meds, err := store.GetUpcomingMeds(db, baby.ID)
	if err != nil {
		t.Fatalf("GetUpcomingMeds failed: %v", err)
	}

	if len(meds) != 1 {
		t.Fatal("expected 1 med")
	}
	if meds[0].Schedule == nil || *meds[0].Schedule != `["08:00","20:00"]` {
		t.Errorf("expected schedule, got %v", meds[0].Schedule)
	}
	if meds[0].Timezone == nil || *meds[0].Timezone != "America/New_York" {
		t.Errorf("expected timezone=America/New_York, got %v", meds[0].Timezone)
	}
}
