package store_test

import (
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestGetFeedingDaily_AggregatesCorrectly(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

	// Two feedings today: formula 120ml + breast_milk (no vol)
	vol := 120.0
	calDen := 20.0
	store.CreateFeeding(db, baby.ID, user.ID, today+"T10:00:00Z", "formula", &vol, &calDen, nil, nil, 67.0)
	store.CreateFeeding(db, baby.ID, user.ID, today+"T14:00:00Z", "breast_milk", nil, nil, nil, nil, 67.0)

	// One feeding yesterday
	vol2 := 100.0
	calDen2 := 20.0
	store.CreateFeeding(db, baby.ID, user.ID, yesterday+"T08:00:00Z", "formula", &vol2, &calDen2, nil, nil, 67.0)

	series, err := store.GetFeedingDaily(db, baby.ID, yesterday, today, time.UTC)
	if err != nil {
		t.Fatalf("GetFeedingDaily failed: %v", err)
	}

	if len(series) != 2 {
		t.Fatalf("expected 2 daily entries, got %d", len(series))
	}

	// Find today's entry
	var todayEntry, yesterdayEntry *store.FeedingDailyEntry
	for i := range series {
		if series[i].Date == today {
			todayEntry = &series[i]
		}
		if series[i].Date == yesterday {
			yesterdayEntry = &series[i]
		}
	}

	if todayEntry == nil {
		t.Fatal("expected today's entry")
	}
	if todayEntry.FeedCount != 2 {
		t.Errorf("expected feed_count=2 for today, got %d", todayEntry.FeedCount)
	}
	if todayEntry.TotalVolumeMl < 119 || todayEntry.TotalVolumeMl > 121 {
		t.Errorf("expected total_volume_ml~120 for today, got %.1f", todayEntry.TotalVolumeMl)
	}
	if todayEntry.ByType.Formula != 1 {
		t.Errorf("expected formula=1 for today, got %d", todayEntry.ByType.Formula)
	}
	if todayEntry.ByType.BreastMilk != 1 {
		t.Errorf("expected breast_milk=1 for today, got %d", todayEntry.ByType.BreastMilk)
	}

	if yesterdayEntry == nil {
		t.Fatal("expected yesterday's entry")
	}
	if yesterdayEntry.FeedCount != 1 {
		t.Errorf("expected feed_count=1 for yesterday, got %d", yesterdayEntry.FeedCount)
	}
}

func TestGetFeedingDaily_EmptyData(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	series, err := store.GetFeedingDaily(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetFeedingDaily failed: %v", err)
	}
	if len(series) != 0 {
		t.Errorf("expected 0 entries, got %d", len(series))
	}
}

func TestGetDiaperDaily_CombinesStoolAndUrine(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	ts1 := today + "T10:00:00Z"
	ts2 := today + "T14:00:00Z"

	store.CreateUrine(db, baby.ID, user.ID, ts1, nil, nil, nil)
	store.CreateUrine(db, baby.ID, user.ID, ts2, nil, nil, nil)
	store.CreateUrine(db, baby.ID, user.ID, today+"T16:00:00Z", nil, nil, nil)

	yellow := "yellow"
	store.CreateStool(db, baby.ID, user.ID, ts1, 3, &yellow, nil, nil, nil, nil)
	store.CreateStool(db, baby.ID, user.ID, ts2, 5, &yellow, nil, nil, nil, nil)

	series, err := store.GetDiaperDaily(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetDiaperDaily failed: %v", err)
	}

	if len(series) != 1 {
		t.Fatalf("expected 1 daily entry, got %d", len(series))
	}

	if series[0].Date != today {
		t.Errorf("expected date=%s, got %s", today, series[0].Date)
	}
	if series[0].WetCount != 3 {
		t.Errorf("expected wet_count=3, got %d", series[0].WetCount)
	}
	if series[0].StoolCount != 2 {
		t.Errorf("expected stool_count=2, got %d", series[0].StoolCount)
	}
}

func TestGetTemperatureSeries_IndividualReadings(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	store.CreateTemperature(db, baby.ID, user.ID, today+"T08:00:00Z", 37.2, "rectal", nil)
	store.CreateTemperature(db, baby.ID, user.ID, today+"T14:00:00Z", 37.8, "axillary", nil)

	series, err := store.GetTemperatureSeries(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetTemperatureSeries failed: %v", err)
	}

	if len(series) != 2 {
		t.Fatalf("expected 2 readings, got %d", len(series))
	}

	// Ordered by timestamp ascending
	if series[0].Value != 37.2 {
		t.Errorf("expected first value=37.2, got %.1f", series[0].Value)
	}
	if series[0].Method != "rectal" {
		t.Errorf("expected first method=rectal, got %s", series[0].Method)
	}
	if series[1].Value != 37.8 {
		t.Errorf("expected second value=37.8, got %.1f", series[1].Value)
	}
}

func TestGetWeightSeries_IndividualReadings(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	src := "clinic"
	store.CreateWeight(db, baby.ID, user.ID, today+"T10:00:00Z", 4.5, &src, nil)
	store.CreateWeight(db, baby.ID, user.ID, today+"T15:00:00Z", 4.55, nil, nil)

	series, err := store.GetWeightSeries(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetWeightSeries failed: %v", err)
	}

	if len(series) != 2 {
		t.Fatalf("expected 2 readings, got %d", len(series))
	}

	if series[0].WeightKg != 4.5 {
		t.Errorf("expected first weight_kg=4.5, got %.2f", series[0].WeightKg)
	}
	if series[0].MeasurementSource == nil || *series[0].MeasurementSource != "clinic" {
		t.Errorf("expected first measurement_source=clinic, got %v", series[0].MeasurementSource)
	}
	if series[1].MeasurementSource != nil {
		t.Errorf("expected second measurement_source=nil, got %v", series[1].MeasurementSource)
	}
}

func TestGetAbdomenGirthSeries_IndividualReadings(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")

	girth1 := 35.0
	girth2 := 36.5
	store.CreateAbdomen(db, baby.ID, user.ID, today+"T10:00:00Z", "soft", false, &girth1, nil)
	store.CreateAbdomen(db, baby.ID, user.ID, today+"T16:00:00Z", "firm", true, &girth2, nil)
	// One without girth - should be excluded
	store.CreateAbdomen(db, baby.ID, user.ID, today+"T18:00:00Z", "soft", false, nil, nil)

	series, err := store.GetAbdomenGirthSeries(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetAbdomenGirthSeries failed: %v", err)
	}

	if len(series) != 2 {
		t.Fatalf("expected 2 readings (excluding nil girth), got %d", len(series))
	}

	if series[0].GirthCm != 35.0 {
		t.Errorf("expected first girth_cm=35.0, got %.1f", series[0].GirthCm)
	}
	if series[1].GirthCm != 36.5 {
		t.Errorf("expected second girth_cm=36.5, got %.1f", series[1].GirthCm)
	}
}

func TestGetStoolColorSeries_ColorCodedData(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	green := "green"
	yellow := "yellow"
	store.CreateStool(db, baby.ID, user.ID, today+"T10:00:00Z", 5, &green, nil, nil, nil, nil)
	store.CreateStool(db, baby.ID, user.ID, today+"T14:00:00Z", 3, &yellow, nil, nil, nil, nil)

	series, err := store.GetStoolColorSeries(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetStoolColorSeries failed: %v", err)
	}

	if len(series) != 2 {
		t.Fatalf("expected 2 readings, got %d", len(series))
	}

	if series[0].ColorScore != 5 {
		t.Errorf("expected first color_score=5, got %d", series[0].ColorScore)
	}
	if series[1].ColorScore != 3 {
		t.Errorf("expected second color_score=3, got %d", series[1].ColorScore)
	}
}

func TestGetLabTrends_GroupsByTestName(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	unit := "mg/dL"

	store.CreateLabResult(db, baby.ID, user.ID, today+"T10:00:00Z", "bilirubin", "5.2", &unit, nil, nil)
	store.CreateLabResult(db, baby.ID, user.ID, today+"T14:00:00Z", "bilirubin", "4.8", &unit, nil, nil)
	store.CreateLabResult(db, baby.ID, user.ID, today+"T10:00:00Z", "ALT", "45", &unit, nil, nil)

	trends, err := store.GetLabTrends(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetLabTrends failed: %v", err)
	}

	if len(trends) != 2 {
		t.Fatalf("expected 2 test groups, got %d", len(trends))
	}

	bilirubinEntries, ok := trends["bilirubin"]
	if !ok {
		t.Fatal("expected bilirubin in trends")
	}
	if len(bilirubinEntries) != 2 {
		t.Errorf("expected 2 bilirubin entries, got %d", len(bilirubinEntries))
	}

	altEntries, ok := trends["ALT"]
	if !ok {
		t.Fatal("expected ALT in trends")
	}
	if len(altEntries) != 1 {
		t.Errorf("expected 1 ALT entry, got %d", len(altEntries))
	}
	if altEntries[0].Value != "45" {
		t.Errorf("expected ALT value=45, got %s", altEntries[0].Value)
	}
}

func TestGetLabTrends_EmptyData(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	trends, err := store.GetLabTrends(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetLabTrends failed: %v", err)
	}
	if len(trends) != 0 {
		t.Errorf("expected 0 groups, got %d", len(trends))
	}
}

func TestGetFeedingDaily_InvalidDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	_, err := store.GetFeedingDaily(db, baby.ID, "not-a-date", "2024-01-01", time.UTC)
	if err == nil {
		t.Error("expected error for invalid from date")
	}

	_, err = store.GetFeedingDaily(db, baby.ID, "2024-01-01", "not-a-date", time.UTC)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}

func TestGetDiaperDaily_InvalidDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	_, err := store.GetDiaperDaily(db, baby.ID, "bad", "2024-01-01", time.UTC)
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestGetTemperatureSeries_InvalidDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	_, err := store.GetTemperatureSeries(db, baby.ID, "bad", "2024-01-01", time.UTC)
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestGetWeightSeries_InvalidDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	_, err := store.GetWeightSeries(db, baby.ID, "bad", "2024-01-01", time.UTC)
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestGetAbdomenGirthSeries_InvalidDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	_, err := store.GetAbdomenGirthSeries(db, baby.ID, "bad", "2024-01-01", time.UTC)
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestGetStoolColorSeries_InvalidDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	_, err := store.GetStoolColorSeries(db, baby.ID, "bad", "2024-01-01", time.UTC)
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestGetLabTrends_InvalidDate(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	_, err := store.GetLabTrends(db, baby.ID, "bad", "2024-01-01", time.UTC)
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestGetFeedingDaily_DateFiltering(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

	vol := 100.0
	calDen := 20.0
	store.CreateFeeding(db, baby.ID, user.ID, yesterday+"T10:00:00Z", "formula", &vol, &calDen, nil, nil, 67.0)
	store.CreateFeeding(db, baby.ID, user.ID, today+"T10:00:00Z", "formula", &vol, &calDen, nil, nil, 67.0)

	// Query only today
	series, err := store.GetFeedingDaily(db, baby.ID, today, today, time.UTC)
	if err != nil {
		t.Fatalf("GetFeedingDaily failed: %v", err)
	}

	if len(series) != 1 {
		t.Fatalf("expected 1 daily entry for today only, got %d", len(series))
	}
	if series[0].Date != today {
		t.Errorf("expected date=%s, got %s", today, series[0].Date)
	}
}
