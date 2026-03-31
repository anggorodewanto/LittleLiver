package integration_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// dashboardResp mirrors the full dashboard JSON response for integration testing.
type dashboardResp struct {
	SummaryCards struct {
		TotalFeeds      int      `json:"total_feeds"`
		TotalCalories   float64  `json:"total_calories"`
		TotalWetDiapers int      `json:"total_wet_diapers"`
		TotalStools     int      `json:"total_stools"`
		WorstStoolColor *int     `json:"worst_stool_color"`
		LastTemperature *float64 `json:"last_temperature"`
		LastWeight      *float64 `json:"last_weight"`
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
	ChartDataSeries struct {
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
	} `json:"chart_data_series"`
	ActiveAlerts []struct {
		EntryID   string  `json:"entry_id"`
		AlertType string  `json:"alert_type"`
		Method    *string `json:"method,omitempty"`
		Value     any     `json:"value"`
		Timestamp string  `json:"timestamp"`
	} `json:"active_alerts"`
}

// getDashboardResp fetches the dashboard endpoint via the integration server and decodes the response.
func getDashboardResp(t *testing.T, client *http.Client, baseURL, babyID, query string) (int, dashboardResp) {
	t.Helper()
	target := baseURL + "/api/babies/" + babyID + "/dashboard"
	if query != "" {
		target += "?" + query
	}
	resp, err := client.Get(target)
	if err != nil {
		t.Fatalf("dashboard request failed: %v", err)
	}
	defer resp.Body.Close()

	var dr dashboardResp
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &dr); err != nil {
			t.Fatalf("unmarshal dashboard: %v\nbody: %s", err, string(body))
		}
	}
	return resp.StatusCode, dr
}

// loginAndGetAuthClient performs the OAuth flow and returns an authenticated client plus the user ID.
func loginAndGetAuthClient(t *testing.T, srv *httptest.Server) (*http.Client, string) {
	t.Helper()
	client := newClientWithCookies(t)

	resp, err := client.Get(srv.URL + "/auth/google/login")
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	u, _ := url.Parse(location)
	state := u.Query().Get("state")

	resp, err = client.Get(srv.URL + "/auth/google/callback?code=test-code&state=" + state)
	if err != nil {
		t.Fatalf("callback failed: %v", err)
	}
	resp.Body.Close()

	resp, err = client.Get(srv.URL + "/api/me")
	if err != nil {
		t.Fatalf("/api/me failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var meResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&meResp)
	user := meResp["user"].(map[string]interface{})
	userID := user["id"].(string)

	return client, userID
}

// createBabyForIntegration inserts a baby and links it to the user. Returns the baby ID.
func createBabyForIntegration(t *testing.T, db *sql.DB, userID string) string {
	t.Helper()
	babyID := model.NewULID()
	_, err := db.Exec(
		"INSERT INTO babies (id, name, sex, date_of_birth) VALUES (?, ?, 'female', '2025-01-01')",
		babyID, "Dashboard Test Baby",
	)
	if err != nil {
		t.Fatalf("insert baby: %v", err)
	}
	_, err = db.Exec(
		"INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)",
		babyID, userID,
	)
	if err != nil {
		t.Fatalf("insert baby_parents: %v", err)
	}
	return babyID
}

func TestDashboardAggregation_Integration(t *testing.T) {
	t.Parallel()

	srv, db, cleanup := setupOAuthIntegrationServer(t)
	defer cleanup()

	client, userID := loginAndGetAuthClient(t, srv)
	babyID := createBabyForIntegration(t, db, userID)

	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	threeDaysAgo := now.AddDate(0, 0, -3).Format("2006-01-02")
	sevenDaysAgo := now.AddDate(0, 0, -7).Format("2006-01-02")

	// --- Seed diverse data across multiple days ---

	// Feedings: formula (today), breast_milk (today), solid (yesterday), formula (3 days ago)
	vol120 := 120.0
	vol80 := 80.0
	calDen20 := 0.676
	calDen24 := 0.811
	f1, _ := store.CreateFeeding(db, babyID, userID, today+"T08:00:00Z", "formula", &vol120, &calDen20, nil, nil, 67.0)
	f2, _ := store.CreateFeeding(db, babyID, userID, today+"T12:00:00Z", "breast_milk", nil, nil, nil, nil, 67.0)
	store.CreateFeeding(db, babyID, userID, yesterday+"T09:00:00Z", "solid", &vol80, &calDen24, nil, nil, 67.0)
	store.CreateFeeding(db, babyID, userID, threeDaysAgo+"T10:00:00Z", "formula", &vol120, &calDen20, nil, nil, 67.0)

	// Compute expected calories for today: formula + breast_milk
	todayExpectedCal := 0.0
	if f1 != nil && f1.Calories != nil {
		todayExpectedCal += *f1.Calories
	}
	if f2 != nil && f2.Calories != nil {
		todayExpectedCal += *f2.Calories
	}

	// Stools: acholic today (rating 2), pigmented yesterday (rating 5),
	//         acholic 3 days ago (rating 1), pigmented 7 days ago (rating 6)
	clay := "clay"
	green := "green"
	white := "white"
	brown := "brown"
	store.CreateStool(db, babyID, userID, today+"T09:00:00Z", 2, &clay, nil, nil, nil, nil)
	store.CreateStool(db, babyID, userID, yesterday+"T11:00:00Z", 5, &green, nil, nil, nil, nil)
	store.CreateStool(db, babyID, userID, threeDaysAgo+"T10:00:00Z", 1, &white, nil, nil, nil, nil)
	store.CreateStool(db, babyID, userID, sevenDaysAgo+"T10:00:00Z", 6, &brown, nil, nil, nil, nil)

	// Urine: 2 today, 1 yesterday
	store.CreateUrine(db, babyID, userID, today+"T07:00:00Z", nil, nil, nil)
	store.CreateUrine(db, babyID, userID, today+"T15:00:00Z", nil, nil, nil)
	store.CreateUrine(db, babyID, userID, yesterday+"T08:00:00Z", nil, nil, nil)

	// Temperatures: normal today, fever today (38.5 rectal), normal yesterday
	store.CreateTemperature(db, babyID, userID, today+"T07:30:00Z", 37.0, "rectal", nil)
	store.CreateTemperature(db, babyID, userID, today+"T14:00:00Z", 38.5, "rectal", nil)
	store.CreateTemperature(db, babyID, userID, yesterday+"T10:00:00Z", 37.2, "axillary", nil)

	// Weights: with and without measurement_source
	homeSource := "home_scale"
	store.CreateWeight(db, babyID, userID, today+"T08:00:00Z", 4.8, &homeSource, nil)
	store.CreateWeight(db, babyID, userID, threeDaysAgo+"T09:00:00Z", 4.6, nil, nil)

	// Abdomen girth
	girth35 := 35.0
	girth34 := 34.0
	store.CreateAbdomen(db, babyID, userID, today+"T08:30:00Z", "soft", false, &girth35, nil)
	store.CreateAbdomen(db, babyID, userID, yesterday+"T09:00:00Z", "firm", false, &girth34, nil)

	// Lab results
	unit := "mg/dL"
	store.CreateLabResult(db, babyID, userID, today+"T10:00:00Z", "bilirubin", "3.5", &unit, nil, nil)
	store.CreateLabResult(db, babyID, userID, threeDaysAgo+"T10:00:00Z", "bilirubin", "4.2", &unit, nil, nil)
	unitUL := "U/L"
	store.CreateLabResult(db, babyID, userID, today+"T10:00:00Z", "GGT", "120", &unitUL, nil, nil)

	// Medications: one active with schedule, one active without schedule
	tz := "UTC"
	sched := `["08:00","20:00"]`
	store.CreateMedication(db, babyID, userID, "Ursodiol", "50mg", "twice_daily", &sched, &tz, nil, nil)
	store.CreateMedication(db, babyID, userID, "Vitamin D", "400IU", "once_daily", nil, nil, nil, nil)

	// Deactivate a third medication
	med3, _ := store.CreateMedication(db, babyID, userID, "Stopped Med", "10mg", "once_daily", nil, nil, nil, nil)
	inactive := false
	store.UpdateMedication(db, babyID, med3.ID, userID, "Stopped Med", "10mg", "once_daily", nil, nil, &inactive, nil, nil)

	// Get Ursodiol ID for med_logs
	var ursodiolID string
	err := db.QueryRow(`SELECT id FROM medications WHERE baby_id = ? AND name = 'Ursodiol'`, babyID).Scan(&ursodiolID)
	if err != nil {
		t.Fatalf("query ursodiol ID: %v", err)
	}

	// Med logs: given dose and skipped dose
	givenAt := today + "T08:15:00Z"
	schedTime := today + "T08:00:00Z"
	store.CreateMedLog(db, babyID, ursodiolID, userID, &schedTime, &givenAt, false, nil, nil)

	skipReason := "vomited"
	schedTime2 := yesterday + "T08:00:00Z"
	store.CreateMedLog(db, babyID, ursodiolID, userID, &schedTime2, nil, true, &skipReason, nil)

	// Skin observation: mild (no jaundice alert)
	mild := "mild_face_only"
	store.CreateSkinObservation(db, babyID, userID, today+"T09:00:00Z", &mild, false, nil, nil, nil)

	// =====================================================
	// Test 1: Today-only query
	// =====================================================
	t.Run("today_only", func(t *testing.T) {
		code, resp := getDashboardResp(t, client, srv.URL, babyID, "from="+today+"&to="+today)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}

		sc := resp.SummaryCards

		// 2 feeds today (formula + breast_milk)
		if sc.TotalFeeds != 2 {
			t.Errorf("total_feeds: expected 2, got %d", sc.TotalFeeds)
		}
		if sc.TotalCalories != todayExpectedCal {
			t.Errorf("total_calories: expected %.4f, got %.4f", todayExpectedCal, sc.TotalCalories)
		}
		if sc.TotalWetDiapers != 2 {
			t.Errorf("wet_diapers: expected 2, got %d", sc.TotalWetDiapers)
		}
		if sc.TotalStools != 1 {
			t.Errorf("stools: expected 1, got %d", sc.TotalStools)
		}
		if sc.WorstStoolColor == nil || *sc.WorstStoolColor != 2 {
			t.Errorf("worst_stool_color: expected 2, got %v", sc.WorstStoolColor)
		}
		// last_temp / last_weight are global (most recent overall)
		if sc.LastTemperature == nil || *sc.LastTemperature != 38.5 {
			t.Errorf("last_temp: expected 38.5, got %v", sc.LastTemperature)
		}
		if sc.LastWeight == nil || *sc.LastWeight != 4.8 {
			t.Errorf("last_weight: expected 4.8, got %v", sc.LastWeight)
		}

		// --- Chart data series ---
		cds := resp.ChartDataSeries

		// feeding_daily: 1 day (today) with 2 feeds
		if len(cds.FeedingDaily) != 1 {
			t.Fatalf("feeding_daily: expected 1 entry, got %d", len(cds.FeedingDaily))
		}
		if cds.FeedingDaily[0].Date != today {
			t.Errorf("feeding_daily date: expected %s, got %s", today, cds.FeedingDaily[0].Date)
		}
		if cds.FeedingDaily[0].FeedCount != 2 {
			t.Errorf("feed_count: expected 2, got %d", cds.FeedingDaily[0].FeedCount)
		}
		if cds.FeedingDaily[0].ByType.Formula != 1 {
			t.Errorf("by_type.formula: expected 1, got %d", cds.FeedingDaily[0].ByType.Formula)
		}
		if cds.FeedingDaily[0].ByType.BreastMilk != 1 {
			t.Errorf("by_type.breast_milk: expected 1, got %d", cds.FeedingDaily[0].ByType.BreastMilk)
		}
		if cds.FeedingDaily[0].ByType.Solid != 0 {
			t.Errorf("by_type.solid: expected 0, got %d", cds.FeedingDaily[0].ByType.Solid)
		}

		// diaper_daily: 1 day (today) with 2 wet, 1 stool
		if len(cds.DiaperDaily) != 1 {
			t.Fatalf("diaper_daily: expected 1 entry, got %d", len(cds.DiaperDaily))
		}
		if cds.DiaperDaily[0].WetCount != 2 {
			t.Errorf("wet_count: expected 2, got %d", cds.DiaperDaily[0].WetCount)
		}
		if cds.DiaperDaily[0].StoolCount != 1 {
			t.Errorf("stool_count: expected 1, got %d", cds.DiaperDaily[0].StoolCount)
		}

		// temperature: 2 readings today, ordered by timestamp ASC
		if len(cds.Temperature) != 2 {
			t.Fatalf("temperature: expected 2 entries, got %d", len(cds.Temperature))
		}
		if cds.Temperature[0].Value != 37.0 {
			t.Errorf("temp[0] value: expected 37.0, got %.1f", cds.Temperature[0].Value)
		}
		if cds.Temperature[1].Value != 38.5 {
			t.Errorf("temp[1] value: expected 38.5, got %.1f", cds.Temperature[1].Value)
		}
		if cds.Temperature[1].Method != "rectal" {
			t.Errorf("temp[1] method: expected rectal, got %s", cds.Temperature[1].Method)
		}

		// weight: 1 reading today
		if len(cds.Weight) != 1 {
			t.Fatalf("weight: expected 1 entry, got %d", len(cds.Weight))
		}
		if cds.Weight[0].WeightKg != 4.8 {
			t.Errorf("weight_kg: expected 4.8, got %.2f", cds.Weight[0].WeightKg)
		}
		if cds.Weight[0].MeasurementSource == nil || *cds.Weight[0].MeasurementSource != "home_scale" {
			t.Errorf("measurement_source: expected home_scale, got %v", cds.Weight[0].MeasurementSource)
		}

		// abdomen_girth: 1 reading today
		if len(cds.AbdomenGirth) != 1 {
			t.Fatalf("abdomen_girth: expected 1 entry, got %d", len(cds.AbdomenGirth))
		}
		if cds.AbdomenGirth[0].GirthCm != 35.0 {
			t.Errorf("girth_cm: expected 35.0, got %.1f", cds.AbdomenGirth[0].GirthCm)
		}

		// stool_color: 1 reading today (color_rating 2)
		if len(cds.StoolColor) != 1 {
			t.Fatalf("stool_color: expected 1 entry, got %d", len(cds.StoolColor))
		}
		if cds.StoolColor[0].ColorScore != 2 {
			t.Errorf("color_score: expected 2, got %d", cds.StoolColor[0].ColorScore)
		}

		// lab_trends: bilirubin (1 today) + GGT (1 today)
		if len(cds.LabTrends) != 2 {
			t.Fatalf("lab_trends: expected 2 groups, got %d", len(cds.LabTrends))
		}
		if biliEntries := cds.LabTrends["bilirubin"]; len(biliEntries) != 1 || biliEntries[0].Value != "3.5" {
			t.Errorf("bilirubin: expected 1 entry with value 3.5, got %v", biliEntries)
		}
		if ggtEntries := cds.LabTrends["GGT"]; len(ggtEntries) != 1 || ggtEntries[0].Value != "120" {
			t.Errorf("GGT: expected 1 entry with value 120, got %v", ggtEntries)
		}

		// --- stool_color_trend: always 7 days ---
		// We have stools on: today (rating 2), yesterday (rating 5), 3 days ago (rating 1)
		// 7 days ago stool might or might not be in the 7-day window depending on exact timing
		if len(resp.StoolColorTrend) < 3 {
			t.Errorf("stool_color_trend: expected at least 3 entries, got %d", len(resp.StoolColorTrend))
		}
		// Most recent (DESC order) should be today's acholic stool
		if len(resp.StoolColorTrend) > 0 && resp.StoolColorTrend[0].ColorRating != 2 {
			t.Errorf("stool_color_trend[0] rating: expected 2, got %d", resp.StoolColorTrend[0].ColorRating)
		}

		// --- upcoming_meds: 2 active (Ursodiol + Vitamin D) ---
		if len(resp.UpcomingMeds) != 2 {
			t.Fatalf("upcoming_meds: expected 2, got %d", len(resp.UpcomingMeds))
		}
		if resp.UpcomingMeds[0].Name != "Ursodiol" {
			t.Errorf("upcoming_meds[0] name: expected Ursodiol, got %s", resp.UpcomingMeds[0].Name)
		}
		if resp.UpcomingMeds[0].Dose != "50mg" {
			t.Errorf("upcoming_meds[0] dose: expected 50mg, got %s", resp.UpcomingMeds[0].Dose)
		}
		if len(resp.UpcomingMeds[0].ScheduleTimes) != 2 {
			t.Errorf("upcoming_meds[0] schedule_times: expected 2, got %d", len(resp.UpcomingMeds[0].ScheduleTimes))
		}
		if resp.UpcomingMeds[0].Timezone == nil || *resp.UpcomingMeds[0].Timezone != "UTC" {
			t.Errorf("upcoming_meds[0] timezone: expected UTC, got %v", resp.UpcomingMeds[0].Timezone)
		}
		if resp.UpcomingMeds[1].Name != "Vitamin D" {
			t.Errorf("upcoming_meds[1] name: expected Vitamin D, got %s", resp.UpcomingMeds[1].Name)
		}
		// Vitamin D has no schedule, so schedule_times should be empty and next_dose_at nil
		if len(resp.UpcomingMeds[1].ScheduleTimes) != 0 {
			t.Errorf("upcoming_meds[1] schedule_times: expected 0, got %d", len(resp.UpcomingMeds[1].ScheduleTimes))
		}
		if resp.UpcomingMeds[1].NextDoseAt != nil {
			t.Errorf("upcoming_meds[1] next_dose_at: expected nil, got %v", resp.UpcomingMeds[1].NextDoseAt)
		}

		// --- active_alerts ---
		// Most recent stool: acholic (rating 2) -> acholic_stool alert
		// Most recent temp: 38.5 rectal (>= 38.0) -> fever alert
		// Most recent skin obs: mild (not severe) -> no jaundice alert
		foundAcholic := false
		foundFever := false
		for _, a := range resp.ActiveAlerts {
			switch a.AlertType {
			case "acholic_stool":
				foundAcholic = true
				if fmt.Sprintf("%v", a.Value) != "2" {
					t.Errorf("acholic_stool value: expected 2, got %v", a.Value)
				}
			case "fever":
				foundFever = true
				if a.Method == nil || *a.Method != "rectal" {
					t.Errorf("fever method: expected rectal, got %v", a.Method)
				}
				if fmt.Sprintf("%v", a.Value) != "38.5" {
					t.Errorf("fever value: expected 38.5, got %v", a.Value)
				}
			case "jaundice_worsening":
				t.Error("unexpected jaundice_worsening alert (skin obs is mild)")
			}
		}
		if !foundAcholic {
			t.Error("expected acholic_stool alert")
		}
		if !foundFever {
			t.Error("expected fever alert")
		}
	})

	// =====================================================
	// Test 2: 7-day range query
	// =====================================================
	t.Run("seven_day_range", func(t *testing.T) {
		from7 := now.AddDate(0, 0, -6).Format("2006-01-02")
		code, resp := getDashboardResp(t, client, srv.URL, babyID, "from="+from7+"&to="+today)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}

		sc := resp.SummaryCards

		// Feedings in range (6 days ago to today): 2 today + 1 yesterday + 1 three_days_ago = 4
		if sc.TotalFeeds != 4 {
			t.Errorf("total_feeds (7d): expected 4, got %d", sc.TotalFeeds)
		}
		if sc.TotalWetDiapers != 3 {
			t.Errorf("wet_diapers (7d): expected 3, got %d", sc.TotalWetDiapers)
		}
		// Stools in range: today + yesterday + 3 days ago = 3
		if sc.TotalStools != 3 {
			t.Errorf("stools (7d): expected 3, got %d", sc.TotalStools)
		}
		if sc.WorstStoolColor == nil || *sc.WorstStoolColor != 1 {
			t.Errorf("worst_stool_color (7d): expected 1, got %v", sc.WorstStoolColor)
		}

		cds := resp.ChartDataSeries

		// feeding_daily: 3 day entries
		if len(cds.FeedingDaily) != 3 {
			t.Errorf("feeding_daily (7d): expected 3 entries, got %d", len(cds.FeedingDaily))
		}

		// diaper_daily: today (2 wet + 1 stool), yesterday (1 wet + 1 stool), 3 days ago (0 wet + 1 stool)
		if len(cds.DiaperDaily) != 3 {
			t.Errorf("diaper_daily (7d): expected 3 entries, got %d", len(cds.DiaperDaily))
		}

		// temperature: 2 today + 1 yesterday = 3
		if len(cds.Temperature) != 3 {
			t.Errorf("temperature (7d): expected 3 entries, got %d", len(cds.Temperature))
		}

		// weight: 1 today + 1 three_days_ago = 2
		if len(cds.Weight) != 2 {
			t.Errorf("weight (7d): expected 2 entries, got %d", len(cds.Weight))
		}
		// Ordered by timestamp ASC: three_days_ago first, then today
		if len(cds.Weight) == 2 {
			if cds.Weight[0].WeightKg != 4.6 {
				t.Errorf("weight[0] (3d ago): expected 4.6, got %.2f", cds.Weight[0].WeightKg)
			}
			if cds.Weight[0].MeasurementSource != nil {
				t.Errorf("weight[0] measurement_source: expected nil, got %v", cds.Weight[0].MeasurementSource)
			}
			if cds.Weight[1].WeightKg != 4.8 {
				t.Errorf("weight[1] (today): expected 4.8, got %.2f", cds.Weight[1].WeightKg)
			}
			if cds.Weight[1].MeasurementSource == nil || *cds.Weight[1].MeasurementSource != "home_scale" {
				t.Errorf("weight[1] measurement_source: expected home_scale, got %v", cds.Weight[1].MeasurementSource)
			}
		}

		// abdomen_girth: 1 today + 1 yesterday = 2
		if len(cds.AbdomenGirth) != 2 {
			t.Errorf("abdomen_girth (7d): expected 2 entries, got %d", len(cds.AbdomenGirth))
		}

		// stool_color: 1 today + 1 yesterday + 1 three_days_ago = 3
		if len(cds.StoolColor) != 3 {
			t.Errorf("stool_color (7d): expected 3 entries, got %d", len(cds.StoolColor))
		}

		// lab_trends: bilirubin (2 entries: 3 days ago + today), GGT (1 entry: today)
		if len(cds.LabTrends) != 2 {
			t.Errorf("lab_trends (7d): expected 2 groups, got %d", len(cds.LabTrends))
		}
		if bili := cds.LabTrends["bilirubin"]; len(bili) != 2 {
			t.Errorf("bilirubin (7d): expected 2 entries, got %d", len(bili))
		}

		// stool_color_trend always 7 days
		if len(resp.StoolColorTrend) < 3 {
			t.Errorf("stool_color_trend (7d): expected at least 3, got %d", len(resp.StoolColorTrend))
		}

		// Active alerts are global
		foundAcholic := false
		foundFever := false
		for _, a := range resp.ActiveAlerts {
			if a.AlertType == "acholic_stool" {
				foundAcholic = true
			}
			if a.AlertType == "fever" {
				foundFever = true
			}
		}
		if !foundAcholic {
			t.Error("expected acholic_stool alert (7d)")
		}
		if !foundFever {
			t.Error("expected fever alert (7d)")
		}
	})

	// =====================================================
	// Test 3: Verify stool_color_trend is always 7 days
	// =====================================================
	t.Run("stool_color_trend_always_7_days", func(t *testing.T) {
		code, resp := getDashboardResp(t, client, srv.URL, babyID, "from="+threeDaysAgo+"&to="+threeDaysAgo)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}

		// Even with a narrow date range, stool_color_trend covers 7 days
		if len(resp.StoolColorTrend) < 3 {
			t.Errorf("stool_color_trend (narrow query): expected at least 3 entries, got %d", len(resp.StoolColorTrend))
		}

		// Verify the summary reflects only the 3-days-ago date
		if resp.SummaryCards.TotalStools != 1 {
			t.Errorf("stools (3d ago only): expected 1, got %d", resp.SummaryCards.TotalStools)
		}
	})

	// =====================================================
	// Test 4: Verify alerts reflect most recent entries globally
	// =====================================================
	t.Run("alerts_reflect_most_recent_state", func(t *testing.T) {
		// The most recent stool is today's acholic (rating 2) -> alert present
		// The most recent temp is today's 38.5 rectal -> fever alert present
		// The most recent skin obs is mild -> no jaundice alert

		code, resp := getDashboardResp(t, client, srv.URL, babyID, "")
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}

		alertTypes := make(map[string]bool)
		for _, a := range resp.ActiveAlerts {
			alertTypes[a.AlertType] = true
		}

		if !alertTypes["acholic_stool"] {
			t.Error("expected acholic_stool alert")
		}
		if !alertTypes["fever"] {
			t.Error("expected fever alert")
		}
		if alertTypes["jaundice_worsening"] {
			t.Error("unexpected jaundice_worsening alert")
		}
	})
}
