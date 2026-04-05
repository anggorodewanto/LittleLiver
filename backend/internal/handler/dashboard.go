package handler

import (
	"database/sql"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// summaryCardsResponse is the JSON response for dashboard summary cards.
type summaryCardsResponse struct {
	TotalFeeds      int      `json:"total_feeds"`
	TotalCalories   float64  `json:"total_calories"`
	TotalWetDiapers int      `json:"total_wet_diapers"`
	TotalStools     int      `json:"total_stools"`
	WorstStoolColor *int     `json:"worst_stool_color"`
	LastTemperature *float64 `json:"last_temperature"`
	LastWeight      *float64 `json:"last_weight"`
}

// stoolColorTrendEntry is the JSON response for a stool color trend entry.
type stoolColorTrendEntry struct {
	Date        string `json:"date"`
	Color       string `json:"color"`
	ColorRating int    `json:"color_rating"`
}

// upcomingMedResponse is the JSON response for a single upcoming medication.
type upcomingMedResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Dose          string   `json:"dose"`
	Frequency     string   `json:"frequency"`
	ScheduleTimes []string `json:"schedule_times"`
	Timezone      *string  `json:"timezone,omitempty"`
	IntervalDays  *int     `json:"interval_days,omitempty"`
	NextDoseAt    *string  `json:"next_dose_at,omitempty"`
}

// chartDataSeriesResponse holds all chart data series for the dashboard.
type chartDataSeriesResponse struct {
	FeedingDaily []store.FeedingDailyEntry        `json:"feeding_daily"`
	DiaperDaily  []store.DiaperDailyEntry         `json:"diaper_daily"`
	Temperature  []store.TemperatureSeriesEntry   `json:"temperature"`
	Weight       []store.WeightSeriesEntry        `json:"weight"`
	AbdomenGirth []store.AbdomenGirthEntry        `json:"abdomen_girth"`
	StoolColor            []store.StoolColorSeriesEntry            `json:"stool_color"`
	HeadCircumference     []store.HeadCircumferenceSeriesEntry     `json:"head_circumference"`
	UpperArmCircumference []store.UpperArmCircumferenceSeriesEntry `json:"upper_arm_circumference"`
	LabTrends             map[string][]store.LabTrendEntry         `json:"lab_trends"`
}

// dashboardResponseJSON is the full dashboard API response.
type dashboardResponseJSON struct {
	SummaryCards    summaryCardsResponse    `json:"summary_cards"`
	StoolColorTrend []stoolColorTrendEntry  `json:"stool_color_trend"`
	UpcomingMeds    []upcomingMedResponse   `json:"upcoming_meds"`
	ChartDataSeries chartDataSeriesResponse `json:"chart_data_series"`
	ActiveAlerts    []store.Alert           `json:"active_alerts"`
}

// DashboardHandler handles GET /api/babies/{id}/dashboard.
func DashboardHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		// Extract timezone from X-Timezone header
		loc := time.UTC
		if tz := optionalTimezone(r); tz != nil {
			if parsed, err := time.LoadLocation(*tz); err == nil {
				loc = parsed
			}
		}

		// Parse from/to, default to today
		today := time.Now().In(loc).Format(model.DateFormat)
		from := r.URL.Query().Get("from")
		if from == "" {
			from = today
		}
		to := r.URL.Query().Get("to")
		if to == "" {
			to = today
		}

		summary, err := store.GetDashboardSummary(db, baby.ID, from, to, loc)
		if err != nil {
			log.Printf("dashboard summary: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		trend, err := store.GetStoolColorTrend(db, baby.ID, loc)
		if err != nil {
			log.Printf("stool color trend: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		meds, err := store.GetUpcomingMeds(db, baby.ID)
		if err != nil {
			log.Printf("upcoming meds: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Fetch chart data series
		feedingDaily, err := store.GetFeedingDaily(db, baby.ID, from, to, loc)
		if err != nil {
			log.Printf("feeding daily: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		diaperDaily, err := store.GetDiaperDaily(db, baby.ID, from, to, loc)
		if err != nil {
			log.Printf("diaper daily: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		tempSeries, err := store.GetTemperatureSeries(db, baby.ID, from, to, loc)
		if err != nil {
			log.Printf("temperature series: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		weightSeries, err := store.GetWeightSeries(db, baby.ID, from, to, loc)
		if err != nil {
			log.Printf("weight series: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		abdomenSeries, err := store.GetAbdomenGirthSeries(db, baby.ID, from, to, loc)
		if err != nil {
			log.Printf("abdomen girth series: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		stoolColorSeries, err := store.GetStoolColorSeries(db, baby.ID, from, to, loc)
		if err != nil {
			log.Printf("stool color series: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		labTrends, err := store.GetLabTrends(db, baby.ID, from, to, loc)
		if err != nil {
			log.Printf("lab trends: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		headCircSeries, err := store.GetHeadCircumferenceSeries(db, baby.ID, from, to, loc)
		if err != nil {
			log.Printf("head circumference series: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		upperArmCircSeries, err := store.GetUpperArmCircumferenceSeries(db, baby.ID, from, to, loc)
		if err != nil {
			log.Printf("upper arm circumference series: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Fetch active alerts (global, ignores from/to)
		storeAlerts, err := store.GetActiveAlerts(db, baby.ID)
		if err != nil {
			log.Printf("active alerts: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Map store types to response types
		summaryResp := summaryCardsResponse{
			TotalFeeds:      summary.TotalFeeds,
			TotalCalories:   summary.TotalCalories,
			TotalWetDiapers: summary.TotalWetDiapers,
			TotalStools:     summary.TotalStools,
			WorstStoolColor: summary.WorstStoolColor,
			LastTemperature: summary.LastTemperature,
			LastWeight:      summary.LastWeight,
		}

		trendResp := make([]stoolColorTrendEntry, 0, len(trend))
		for _, e := range trend {
			trendResp = append(trendResp, stoolColorTrendEntry{
				Date:        e.Date,
				Color:       e.Color,
				ColorRating: e.ColorRating,
			})
		}

		upcomingMeds := make([]upcomingMedResponse, 0, len(meds))
		for _, m := range meds {
			resp := upcomingMedResponse{
				ID:            m.ID,
				Name:          m.Name,
				Dose:          m.Dose,
				Frequency:     m.Frequency,
				ScheduleTimes: parseScheduleTimes(m.Schedule),
				Timezone:      m.Timezone,
				IntervalDays:  m.IntervalDays,
			}
			if m.Frequency == "every_x_days" {
				resp.NextDoseAt = computeNextDoseAtInterval(m)
			} else {
				resp.NextDoseAt = computeNextDoseAt(resp.ScheduleTimes, m.Timezone, m.LastGivenAt)
			}
			upcomingMeds = append(upcomingMeds, resp)
		}

		result := dashboardResponseJSON{
			SummaryCards:    summaryResp,
			StoolColorTrend: trendResp,
			UpcomingMeds:    upcomingMeds,
			ChartDataSeries: chartDataSeriesResponse{
				FeedingDaily: feedingDaily,
				DiaperDaily:  diaperDaily,
				Temperature:  tempSeries,
				Weight:       weightSeries,
				AbdomenGirth: abdomenSeries,
				StoolColor:            stoolColorSeries,
				HeadCircumference:     headCircSeries,
				UpperArmCircumference: upperArmCircSeries,
				LabTrends:             labTrends,
			},
			ActiveAlerts: storeAlerts,
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// computeNextDoseAtInterval calculates the next dose time for every_x_days medications.
// Delegates to store.NextDoseTimeInterval for the scheduling logic, then formats as UTC ISO string.
func computeNextDoseAtInterval(m store.UpcomingMed) *string {
	farFuture := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	dueDate := store.NextDoseTimeInterval(m, farFuture)
	if dueDate.Equal(farFuture) {
		return nil
	}
	s := dueDate.UTC().Format(model.DateTimeFormat)
	return &s
}

// computeNextDoseAt calculates the next dose time based on schedule_times and timezone.
// It skips schedule slots that are already covered by lastGivenAt (dose logged after that time).
// Returns nil if no schedule or timezone is available.
func computeNextDoseAt(scheduleTimes []string, tz *string, lastGivenAt *time.Time) *string {
	if len(scheduleTimes) == 0 || tz == nil {
		return nil
	}

	loc, err := time.LoadLocation(*tz)
	if err != nil {
		return nil
	}

	now := time.Now().In(loc)
	todayStr := now.Format(model.DateFormat)
	overdueGrace := 60 * time.Minute

	var earliest time.Time
	found := false

	// Check today's schedule times that are either upcoming or recently
	// passed (within the overdue grace window so the frontend can still
	// show them as "due now"). Skip times already covered by a logged dose.
	sort.Strings(scheduleTimes)
	for i, st := range scheduleTimes {
		t, err := time.ParseInLocation(model.DateFormat+" 15:04", todayStr+" "+st, loc)
		if err != nil {
			continue
		}
		// Skip this slot if a dose was logged after the midpoint between
		// this slot and the previous one (covers early dosing).
		coverStart := store.SlotCoverageStart(scheduleTimes, i, todayStr, loc)
		if lastGivenAt != nil && !lastGivenAt.In(loc).Before(coverStart) {
			continue
		}
		if now.Sub(t) <= overdueGrace && (!found || t.Before(earliest)) {
			earliest = t
			found = true
		}
	}

	// If no remaining times today, use tomorrow's first time
	if !found {
		tomorrowStr := now.AddDate(0, 0, 1).Format(model.DateFormat)
		for _, st := range scheduleTimes {
			t, err := time.ParseInLocation(model.DateFormat+" 15:04", tomorrowStr+" "+st, loc)
			if err != nil {
				continue
			}
			if !found || t.Before(earliest) {
				earliest = t
				found = true
			}
		}
	}

	if !found {
		return nil
	}

	s := earliest.UTC().Format(model.DateTimeFormat)
	return &s
}
