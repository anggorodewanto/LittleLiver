package handler_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// dashboardResp is the expected shape of the dashboard API response.
type dashboardResp struct {
	SummaryCards struct {
		TotalFeeds     int      `json:"total_feeds"`
		TotalCalories  float64  `json:"total_calories"`
		WetDiapers     int      `json:"wet_diapers"`
		Stools         int      `json:"stools"`
		ColorIndicator *string  `json:"color_indicator"`
		LastTemp       *float64 `json:"last_temp"`
		LastWeight     *float64 `json:"last_weight"`
	} `json:"summary_cards"`
	StoolColorTrend []struct {
		Date        string `json:"date"`
		Color       string `json:"color"`
		ColorRating int    `json:"color_rating"`
	} `json:"stool_color_trend"`
	UpcomingMeds []struct {
		ID            string   `json:"id"`
		Name          string   `json:"name"`
		Dose          string   `json:"dose"`
		Frequency     string   `json:"frequency"`
		ScheduleTimes []string `json:"schedule_times"`
		Timezone      *string  `json:"timezone"`
		NextDoseAt    *string  `json:"next_dose_at"`
	} `json:"upcoming_meds"`
	ChartDataSeries *chartDataSeriesResp `json:"chart_data_series"`
}

type chartDataSeriesResp struct {
	FeedingDaily []struct {
		Date          string  `json:"date"`
		TotalVolumeMl float64 `json:"total_volume_ml"`
		TotalCalories float64 `json:"total_calories"`
		FeedCount     int     `json:"feed_count"`
		ByType        struct {
			BreastMilk int `json:"breast_milk"`
			Formula    int `json:"formula"`
			Solid      int `json:"solid"`
			Other      int `json:"other"`
		} `json:"by_type"`
	} `json:"feeding_daily"`
	DiaperDaily []struct {
		Date       string `json:"date"`
		WetCount   int    `json:"wet_count"`
		StoolCount int    `json:"stool_count"`
	} `json:"diaper_daily"`
	Temperature []struct {
		Timestamp string  `json:"timestamp"`
		Value     float64 `json:"value"`
		Method    string  `json:"method"`
	} `json:"temperature"`
	Weight []struct {
		Timestamp         string  `json:"timestamp"`
		WeightKg          float64 `json:"weight_kg"`
		MeasurementSource *string `json:"measurement_source"`
	} `json:"weight"`
	AbdomenGirth []struct {
		Timestamp string  `json:"timestamp"`
		GirthCm   float64 `json:"girth_cm"`
	} `json:"abdomen_girth"`
	StoolColor []struct {
		Timestamp  string `json:"timestamp"`
		ColorScore int    `json:"color_score"`
	} `json:"stool_color"`
	LabTrends map[string][]struct {
		Timestamp string  `json:"timestamp"`
		TestName  string  `json:"test_name"`
		Value     string  `json:"value"`
		Unit      *string `json:"unit"`
	} `json:"lab_trends"`
}

func doDashboardRequest(t *testing.T, db *sql.DB, userID, babyID, queryString string) (*httptest.ResponseRecorder, dashboardResp) {
	t.Helper()

	target := "/api/babies/" + babyID + "/dashboard"
	if queryString != "" {
		target += "?" + queryString
	}
	req := testutil.AuthenticatedRequest(t, db, userID, testCookieName, testSecret, http.MethodGet, target)

	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.DashboardHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var resp dashboardResp
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal dashboard response failed: %v", err)
		}
	}
	return rec, resp
}

func TestDashboardHandler_EmptyData_DefaultsToToday(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if resp.SummaryCards.TotalFeeds != 0 {
		t.Errorf("expected 0 total_feeds, got %d", resp.SummaryCards.TotalFeeds)
	}
	if resp.SummaryCards.TotalCalories != 0 {
		t.Errorf("expected 0 total_calories, got %f", resp.SummaryCards.TotalCalories)
	}
	if resp.SummaryCards.WetDiapers != 0 {
		t.Errorf("expected 0 wet_diapers, got %d", resp.SummaryCards.WetDiapers)
	}
	if resp.SummaryCards.Stools != 0 {
		t.Errorf("expected 0 stools, got %d", resp.SummaryCards.Stools)
	}
	if resp.SummaryCards.LastTemp != nil {
		t.Errorf("expected nil last_temp, got %v", resp.SummaryCards.LastTemp)
	}
	if resp.SummaryCards.LastWeight != nil {
		t.Errorf("expected nil last_weight, got %v", resp.SummaryCards.LastWeight)
	}

	// Verify arrays are not null in JSON
	raw := rec.Body.Bytes()
	var rawJSON map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawJSON); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if string(rawJSON["stool_color_trend"]) == "null" {
		t.Error("expected stool_color_trend to be empty array, not null")
	}
	if string(rawJSON["upcoming_meds"]) == "null" {
		t.Error("expected upcoming_meds to be empty array, not null")
	}
}

func TestDashboardHandler_WithData(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	ts := today + "T10:00:00Z"
	ts2 := today + "T14:00:00Z"

	// Seed data
	vol := 120.0
	calDen := 20.0
	store.CreateFeeding(db, baby.ID, user.ID, ts, "formula", &vol, &calDen, nil, nil, 67.0)
	store.CreateFeeding(db, baby.ID, user.ID, ts2, "breast_milk", nil, nil, nil, nil, 67.0)
	store.CreateUrine(db, baby.ID, user.ID, ts, nil, nil)
	store.CreateUrine(db, baby.ID, user.ID, ts2, nil, nil)

	green := "green"
	store.CreateStool(db, baby.ID, user.ID, ts2, 5, &green, nil, nil, nil)

	store.CreateTemperature(db, baby.ID, user.ID, ts, 37.5, "rectal", nil)
	store.CreateWeight(db, baby.ID, user.ID, ts, 4.8, nil, nil)

	tz := "America/New_York"
	sched := `["08:00","20:00"]`
	store.CreateMedication(db, baby.ID, user.ID, "Ursodiol", "50mg", "twice_daily", &sched, &tz)

	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if resp.SummaryCards.TotalFeeds != 2 {
		t.Errorf("expected 2 total_feeds, got %d", resp.SummaryCards.TotalFeeds)
	}
	if resp.SummaryCards.WetDiapers != 2 {
		t.Errorf("expected 2 wet_diapers, got %d", resp.SummaryCards.WetDiapers)
	}
	if resp.SummaryCards.Stools != 1 {
		t.Errorf("expected 1 stools, got %d", resp.SummaryCards.Stools)
	}
	if resp.SummaryCards.ColorIndicator == nil || *resp.SummaryCards.ColorIndicator != "green" {
		t.Errorf("expected color_indicator=green, got %v", resp.SummaryCards.ColorIndicator)
	}
	if resp.SummaryCards.LastTemp == nil || *resp.SummaryCards.LastTemp != 37.5 {
		t.Errorf("expected last_temp=37.5, got %v", resp.SummaryCards.LastTemp)
	}
	if resp.SummaryCards.LastWeight == nil || *resp.SummaryCards.LastWeight != 4.8 {
		t.Errorf("expected last_weight=4.8, got %v", resp.SummaryCards.LastWeight)
	}

	// Check upcoming_meds
	if len(resp.UpcomingMeds) != 1 {
		t.Fatalf("expected 1 upcoming med, got %d", len(resp.UpcomingMeds))
	}
	if resp.UpcomingMeds[0].Name != "Ursodiol" {
		t.Errorf("expected med name=Ursodiol, got %s", resp.UpcomingMeds[0].Name)
	}
	if len(resp.UpcomingMeds[0].ScheduleTimes) != 2 {
		t.Errorf("expected 2 schedule_times, got %d", len(resp.UpcomingMeds[0].ScheduleTimes))
	}
	if resp.UpcomingMeds[0].Timezone == nil || *resp.UpcomingMeds[0].Timezone != "America/New_York" {
		t.Errorf("expected timezone=America/New_York, got %v", resp.UpcomingMeds[0].Timezone)
	}
}

func TestDashboardHandler_StoolColorTrend_Always7Days(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	now := time.Now().UTC()
	for i := 0; i < 10; i++ {
		day := now.AddDate(0, 0, -i)
		ts := day.Format("2006-01-02") + "T12:00:00Z"
		label := "yellow"
		store.CreateStool(db, baby.ID, user.ID, ts, 3, &label, nil, nil, nil)
	}

	// Even when querying a narrow date range, stool_color_trend is always 7 days
	fromTo := "from=" + now.Format("2006-01-02") + "&to=" + now.Format("2006-01-02")
	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, fromTo)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if len(resp.StoolColorTrend) != 7 {
		t.Errorf("expected 7 stool_color_trend entries regardless of from/to, got %d", len(resp.StoolColorTrend))
	}
}

func TestDashboardHandler_UpcomingMeds_ExcludesDeactivated(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	tz := "UTC"
	sched := `["09:00"]`

	store.CreateMedication(db, baby.ID, user.ID, "Active Med", "10mg", "once_daily", &sched, &tz)
	med2, _ := store.CreateMedication(db, baby.ID, user.ID, "Inactive Med", "5mg", "once_daily", &sched, &tz)
	inactive := false
	store.UpdateMedication(db, baby.ID, med2.ID, user.ID, "Inactive Med", "5mg", "once_daily", &sched, &tz, &inactive)

	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if len(resp.UpcomingMeds) != 1 {
		t.Fatalf("expected 1 upcoming med, got %d", len(resp.UpcomingMeds))
	}
	if resp.UpcomingMeds[0].Name != "Active Med" {
		t.Errorf("expected Active Med, got %s", resp.UpcomingMeds[0].Name)
	}
}

func TestDashboardHandler_DateRangeParams(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	today := time.Now().UTC().Format("2006-01-02")

	vol := 100.0
	calDen := 20.0
	store.CreateFeeding(db, baby.ID, user.ID, yesterday+"T10:00:00Z", "formula", &vol, &calDen, nil, nil, 67.0)
	store.CreateFeeding(db, baby.ID, user.ID, today+"T10:00:00Z", "formula", &vol, &calDen, nil, nil, 67.0)

	// Query only today
	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, "from="+today+"&to="+today)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if resp.SummaryCards.TotalFeeds != 1 {
		t.Errorf("expected 1 feed for today, got %d", resp.SummaryCards.TotalFeeds)
	}
}

func TestDashboardHandler_ChartDataSeries(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	ts1 := today + "T10:00:00Z"
	ts2 := today + "T14:00:00Z"

	// Seed feeding data
	vol := 120.0
	calDen := 20.0
	store.CreateFeeding(db, baby.ID, user.ID, ts1, "formula", &vol, &calDen, nil, nil, 67.0)
	store.CreateFeeding(db, baby.ID, user.ID, ts2, "breast_milk", nil, nil, nil, nil, 67.0)

	// Seed diaper data
	store.CreateUrine(db, baby.ID, user.ID, ts1, nil, nil)
	store.CreateUrine(db, baby.ID, user.ID, ts2, nil, nil)
	green := "green"
	store.CreateStool(db, baby.ID, user.ID, ts1, 5, &green, nil, nil, nil)

	// Seed vitals
	store.CreateTemperature(db, baby.ID, user.ID, ts1, 37.2, "rectal", nil)
	store.CreateWeight(db, baby.ID, user.ID, ts1, 4.5, nil, nil)
	girth := 35.0
	store.CreateAbdomen(db, baby.ID, user.ID, ts1, "soft", false, &girth, nil)

	// Seed lab result
	unit := "mg/dL"
	store.CreateLabResult(db, baby.ID, user.ID, ts1, "bilirubin", "5.2", &unit, nil, nil)

	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, "from="+today+"&to="+today)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	cds := resp.ChartDataSeries
	if cds == nil {
		t.Fatal("expected chart_data_series to be present")
	}

	// feeding_daily
	if len(cds.FeedingDaily) != 1 {
		t.Fatalf("expected 1 feeding_daily entry, got %d", len(cds.FeedingDaily))
	}
	if cds.FeedingDaily[0].FeedCount != 2 {
		t.Errorf("expected feed_count=2, got %d", cds.FeedingDaily[0].FeedCount)
	}
	if cds.FeedingDaily[0].ByType.Formula != 1 {
		t.Errorf("expected formula=1, got %d", cds.FeedingDaily[0].ByType.Formula)
	}
	if cds.FeedingDaily[0].ByType.BreastMilk != 1 {
		t.Errorf("expected breast_milk=1, got %d", cds.FeedingDaily[0].ByType.BreastMilk)
	}

	// diaper_daily
	if len(cds.DiaperDaily) != 1 {
		t.Fatalf("expected 1 diaper_daily entry, got %d", len(cds.DiaperDaily))
	}
	if cds.DiaperDaily[0].WetCount != 2 {
		t.Errorf("expected wet_count=2, got %d", cds.DiaperDaily[0].WetCount)
	}
	if cds.DiaperDaily[0].StoolCount != 1 {
		t.Errorf("expected stool_count=1, got %d", cds.DiaperDaily[0].StoolCount)
	}

	// temperature
	if len(cds.Temperature) != 1 {
		t.Fatalf("expected 1 temperature reading, got %d", len(cds.Temperature))
	}
	if cds.Temperature[0].Value != 37.2 {
		t.Errorf("expected temp=37.2, got %.1f", cds.Temperature[0].Value)
	}

	// weight
	if len(cds.Weight) != 1 {
		t.Fatalf("expected 1 weight reading, got %d", len(cds.Weight))
	}
	if cds.Weight[0].WeightKg != 4.5 {
		t.Errorf("expected weight=4.5, got %.2f", cds.Weight[0].WeightKg)
	}

	// abdomen_girth
	if len(cds.AbdomenGirth) != 1 {
		t.Fatalf("expected 1 abdomen_girth reading, got %d", len(cds.AbdomenGirth))
	}
	if cds.AbdomenGirth[0].GirthCm != 35.0 {
		t.Errorf("expected girth_cm=35.0, got %.1f", cds.AbdomenGirth[0].GirthCm)
	}

	// stool_color
	if len(cds.StoolColor) != 1 {
		t.Fatalf("expected 1 stool_color reading, got %d", len(cds.StoolColor))
	}
	if cds.StoolColor[0].ColorScore != 5 {
		t.Errorf("expected color_score=5, got %d", cds.StoolColor[0].ColorScore)
	}

	// lab_trends
	if len(cds.LabTrends) != 1 {
		t.Fatalf("expected 1 lab trend group, got %d", len(cds.LabTrends))
	}
	biliEntries, ok := cds.LabTrends["bilirubin"]
	if !ok {
		t.Fatal("expected bilirubin in lab_trends")
	}
	if len(biliEntries) != 1 {
		t.Errorf("expected 1 bilirubin entry, got %d", len(biliEntries))
	}
	if biliEntries[0].Value != "5.2" {
		t.Errorf("expected bilirubin value=5.2, got %s", biliEntries[0].Value)
	}
}

func TestDashboardHandler_ChartDataSeries_EmptyArraysNotNull(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	rec, _ := doDashboardRequest(t, db, user.ID, baby.ID, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	raw := rec.Body.Bytes()
	var rawJSON map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawJSON); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	cdsRaw, ok := rawJSON["chart_data_series"]
	if !ok {
		t.Fatal("expected chart_data_series key in response")
	}

	var cds map[string]json.RawMessage
	if err := json.Unmarshal(cdsRaw, &cds); err != nil {
		t.Fatalf("unmarshal chart_data_series: %v", err)
	}

	for _, key := range []string{"feeding_daily", "diaper_daily", "temperature", "weight", "abdomen_girth", "stool_color"} {
		if string(cds[key]) == "null" {
			t.Errorf("expected %s to be empty array, not null", key)
		}
	}
	// lab_trends should be empty object, not null
	if string(cds["lab_trends"]) == "null" {
		t.Error("expected lab_trends to be empty object, not null")
	}
}

func TestDashboardHandler_ChartDataSeries_DateRangeFiltering(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	twoDaysAgo := time.Now().UTC().AddDate(0, 0, -2).Format("2006-01-02")

	// Seed data across multiple days
	vol := 100.0
	calDen := 20.0
	store.CreateFeeding(db, baby.ID, user.ID, twoDaysAgo+"T10:00:00Z", "formula", &vol, &calDen, nil, nil, 67.0)
	store.CreateFeeding(db, baby.ID, user.ID, yesterday+"T10:00:00Z", "formula", &vol, &calDen, nil, nil, 67.0)
	store.CreateFeeding(db, baby.ID, user.ID, today+"T10:00:00Z", "formula", &vol, &calDen, nil, nil, 67.0)

	store.CreateTemperature(db, baby.ID, user.ID, twoDaysAgo+"T10:00:00Z", 37.0, "rectal", nil)
	store.CreateTemperature(db, baby.ID, user.ID, yesterday+"T10:00:00Z", 37.5, "rectal", nil)

	// Query only yesterday
	rec, resp := doDashboardRequest(t, db, user.ID, baby.ID, "from="+yesterday+"&to="+yesterday)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	cds := resp.ChartDataSeries
	if cds == nil {
		t.Fatal("expected chart_data_series to be present")
	}

	if len(cds.FeedingDaily) != 1 {
		t.Fatalf("expected 1 feeding_daily entry for yesterday, got %d", len(cds.FeedingDaily))
	}
	if cds.FeedingDaily[0].Date != yesterday {
		t.Errorf("expected date=%s, got %s", yesterday, cds.FeedingDaily[0].Date)
	}

	if len(cds.Temperature) != 1 {
		t.Fatalf("expected 1 temperature reading for yesterday, got %d", len(cds.Temperature))
	}
	if cds.Temperature[0].Value != 37.5 {
		t.Errorf("expected temp=37.5, got %.1f", cds.Temperature[0].Value)
	}
}

func TestDashboardHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := http.HandlerFunc(handler.DashboardHandler(db))
	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/dashboard", h)

	req := httptest.NewRequest(http.MethodGet, "/api/babies/some-id/dashboard", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestDashboardHandler_Forbidden(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	otherUser := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, otherUser.ID)

	rec, _ := doDashboardRequest(t, db, user.ID, baby.ID, "")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
