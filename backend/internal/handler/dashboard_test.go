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
