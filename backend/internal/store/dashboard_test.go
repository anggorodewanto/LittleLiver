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
	summary, err := store.GetDashboardSummary(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetDashboardSummary failed: %v", err)
	}

	if summary.TotalFeeds != 0 {
		t.Errorf("expected 0 total_feeds, got %d", summary.TotalFeeds)
	}
	if summary.TotalCalories != 0 {
		t.Errorf("expected 0 total_calories, got %f", summary.TotalCalories)
	}
	if summary.TotalWetDiapers != 0 {
		t.Errorf("expected 0 wet_diapers, got %d", summary.TotalWetDiapers)
	}
	if summary.TotalStools != 0 {
		t.Errorf("expected 0 stools, got %d", summary.TotalStools)
	}
	if summary.WorstStoolColor != nil {
		t.Errorf("expected nil color_indicator, got %v", summary.WorstStoolColor)
	}
	if summary.LastTemperature != nil {
		t.Errorf("expected nil last_temp, got %v", summary.LastTemperature)
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
	store.CreateUrine(db, baby.ID, user.ID, ts, nil, nil, nil)
	store.CreateUrine(db, baby.ID, user.ID, ts2, nil, nil, nil)
	store.CreateUrine(db, baby.ID, user.ID, today+"T16:00:00Z", nil, nil, nil)

	// Insert 2 stools - most recent should be green
	yellow := "yellow"
	green := "green"
	store.CreateStool(db, baby.ID, user.ID, ts, 3, &yellow, nil, nil, nil, nil)
	store.CreateStool(db, baby.ID, user.ID, ts2, 5, &green, nil, nil, nil, nil)

	// Insert temperature and weight
	store.CreateTemperature(db, baby.ID, user.ID, ts, 37.2, "rectal", nil)
	store.CreateWeight(db, baby.ID, user.ID, ts, 4.5, nil, nil)

	summary, err := store.GetDashboardSummary(db, baby.ID, today, today, time.UTC)
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
	if summary.TotalWetDiapers != 3 {
		t.Errorf("expected 3 wet_diapers, got %d", summary.TotalWetDiapers)
	}
	if summary.TotalStools != 2 {
		t.Errorf("expected 2 stools, got %d", summary.TotalStools)
	}
	if summary.WorstStoolColor == nil || *summary.WorstStoolColor != 3 {
		t.Errorf("expected worst_stool_color=3, got %v", summary.WorstStoolColor)
	}
	if summary.LastTemperature == nil || *summary.LastTemperature != 37.2 {
		t.Errorf("expected last_temp=37.2, got %v", summary.LastTemperature)
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
	summary, err := store.GetDashboardSummary(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetDashboardSummary failed: %v", err)
	}

	if summary.TotalFeeds != 1 {
		t.Errorf("expected 1 total_feeds for today, got %d", summary.TotalFeeds)
	}
}

func TestGetDashboardSummary_LastTemperatureAndWeight_IgnoreDateRange(t *testing.T) {
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
	summary, err := store.GetDashboardSummary(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetDashboardSummary failed: %v", err)
	}

	if summary.LastTemperature == nil || *summary.LastTemperature != 38.5 {
		t.Errorf("expected last_temp=38.5 regardless of date range, got %v", summary.LastTemperature)
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
		store.CreateStool(db, baby.ID, user.ID, ts, 3, &label, nil, nil, nil, nil)
	}

	trend, err := store.GetStoolColorTrend(db, baby.ID, time.UTC)
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
	store.CreateStool(db, baby.ID, user.ID, ts, 5, &green, nil, nil, nil, nil)

	trend, err := store.GetStoolColorTrend(db, baby.ID, time.UTC)
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

func TestGetUpcomingMeds_SortedByNextScheduledDose(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	now := time.Now().UTC()

	// Med A: next dose is 3 hours from now
	schedA := `["` + now.Add(3*time.Hour).Format("15:04") + `"]`
	store.CreateMedication(db, baby.ID, user.ID, "MedA", "10mg", "once_daily", &schedA, &tz)

	// Med B: next dose is 1 hour from now (should appear first)
	schedB := `["` + now.Add(1*time.Hour).Format("15:04") + `"]`
	store.CreateMedication(db, baby.ID, user.ID, "MedB", "5mg", "once_daily", &schedB, &tz)

	// Med C: next dose is 2 hours from now
	schedC := `["` + now.Add(2*time.Hour).Format("15:04") + `"]`
	store.CreateMedication(db, baby.ID, user.ID, "MedC", "20mg", "once_daily", &schedC, &tz)

	meds, err := store.GetUpcomingMeds(db, baby.ID)
	if err != nil {
		t.Fatalf("GetUpcomingMeds failed: %v", err)
	}

	if len(meds) != 3 {
		t.Fatalf("expected 3 meds, got %d", len(meds))
	}
	if meds[0].Name != "MedB" {
		t.Errorf("expected first med to be MedB (soonest), got %s", meds[0].Name)
	}
	if meds[1].Name != "MedC" {
		t.Errorf("expected second med to be MedC, got %s", meds[1].Name)
	}
	if meds[2].Name != "MedA" {
		t.Errorf("expected third med to be MedA (latest), got %s", meds[2].Name)
	}
}

func TestParseDateRangeInLocation_NonUTC(t *testing.T) {
	t.Parallel()

	// For Asia/Tokyo (UTC+9), "2025-03-15" means 2025-03-15T00:00:00+09:00 = 2025-03-14T15:00:00Z
	// For UTC, "2025-03-15" means 2025-03-15T00:00:00Z
	// So the from times should differ.
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("load Tokyo location: %v", err)
	}

	fromUTC, _, err := store.ParseDateRange("2025-03-15", "2025-03-15")
	if err != nil {
		t.Fatalf("ParseDateRange failed: %v", err)
	}

	fromTZ, _, err := store.ParseDateRangeInLocation("2025-03-15", "2025-03-15", tokyo)
	if err != nil {
		t.Fatalf("ParseDateRangeInLocation failed: %v", err)
	}

	// UTC parsing: from = "2025-03-15T00:00:00Z"
	// Tokyo parsing: from = "2025-03-14T15:00:00Z" (midnight Tokyo = 15:00 UTC previous day)
	if fromUTC == fromTZ {
		t.Errorf("expected different from times for UTC vs Tokyo, both got %s", fromUTC)
	}
	if fromTZ != "2025-03-14T15:00:00Z" {
		t.Errorf("expected fromTZ=2025-03-14T15:00:00Z, got %s", fromTZ)
	}
}

func TestParseDateRangeInLocation_UTC_BackwardsCompatible(t *testing.T) {
	t.Parallel()

	fromOld, toOld, err := store.ParseDateRange("2025-03-15", "2025-03-16")
	if err != nil {
		t.Fatalf("ParseDateRange failed: %v", err)
	}

	fromNew, toNew, err := store.ParseDateRangeInLocation("2025-03-15", "2025-03-16", time.UTC)
	if err != nil {
		t.Fatalf("ParseDateRangeInLocation failed: %v", err)
	}

	if fromOld != fromNew {
		t.Errorf("expected same from for UTC, got %s vs %s", fromOld, fromNew)
	}
	if toOld != toNew {
		t.Errorf("expected same to for UTC, got %s vs %s", toOld, toNew)
	}
}

func TestGetDashboardSummary_WithTimezone(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Insert a feeding at 2025-03-15T01:00:00Z (UTC)
	// In Asia/Tokyo (UTC+9) this is 2025-03-15T10:00:00+09:00 — i.e., March 15th in Tokyo
	// In UTC this is March 15th too
	// But for date "2025-03-15" in Tokyo, the range is 2025-03-14T15:00:00Z to 2025-03-15T15:00:00Z
	// For date "2025-03-15" in UTC, the range is 2025-03-15T00:00:00Z to 2025-03-16T00:00:00Z
	//
	// Insert a feeding at 2025-03-14T16:00:00Z — this is March 15th in Tokyo but March 14th in UTC
	store.CreateFeeding(db, baby.ID, user.ID, "2025-03-14T16:00:00Z", "formula", nil, nil, nil, nil, 67.0)

	tokyo, _ := time.LoadLocation("Asia/Tokyo")

	// Query for March 15 with Tokyo timezone — should include the feeding
	summaryTZ, err := store.GetDashboardSummary(db, baby.ID, "2025-03-15", "2025-03-15", tokyo)
	if err != nil {
		t.Fatalf("GetDashboardSummary with tz failed: %v", err)
	}
	if summaryTZ.TotalFeeds != 1 {
		t.Errorf("expected 1 feed for March 15 in Tokyo, got %d", summaryTZ.TotalFeeds)
	}

	// Query for March 15 with UTC — should NOT include the feeding (it's March 14 in UTC)
	summaryUTC, err := store.GetDashboardSummary(db, baby.ID, "2025-03-15", "2025-03-15", time.UTC)
	if err != nil {
		t.Fatalf("GetDashboardSummary with UTC failed: %v", err)
	}
	if summaryUTC.TotalFeeds != 0 {
		t.Errorf("expected 0 feeds for March 15 in UTC, got %d", summaryUTC.TotalFeeds)
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
