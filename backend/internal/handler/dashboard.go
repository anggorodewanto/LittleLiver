package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// upcomingMedResponse is the JSON response for a single upcoming medication.
type upcomingMedResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Dose          string   `json:"dose"`
	Frequency     string   `json:"frequency"`
	ScheduleTimes []string `json:"schedule_times"`
	Timezone      *string  `json:"timezone,omitempty"`
	NextDoseAt    *string  `json:"next_dose_at,omitempty"`
}

// dashboardResponseJSON is the full dashboard API response.
type dashboardResponseJSON struct {
	SummaryCards    *store.DashboardSummary `json:"summary_cards"`
	StoolColorTrend []store.StoolColorEntry `json:"stool_color_trend"`
	UpcomingMeds    []upcomingMedResponse   `json:"upcoming_meds"`
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

		// Parse from/to, default to today
		today := time.Now().UTC().Format("2006-01-02")
		from := r.URL.Query().Get("from")
		if from == "" {
			from = today
		}
		to := r.URL.Query().Get("to")
		if to == "" {
			to = today
		}

		summary, err := store.GetDashboardSummary(db, baby.ID, from, to)
		if err != nil {
			log.Printf("dashboard summary: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		trend, err := store.GetStoolColorTrend(db, baby.ID)
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

		upcomingMeds := make([]upcomingMedResponse, 0, len(meds))
		for _, m := range meds {
			resp := upcomingMedResponse{
				ID:        m.ID,
				Name:      m.Name,
				Dose:      m.Dose,
				Frequency: m.Frequency,
				Timezone:  m.Timezone,
			}

			// Parse schedule_times from JSON
			if m.Schedule != nil && *m.Schedule != "" {
				var times []string
				if err := json.Unmarshal([]byte(*m.Schedule), &times); err != nil {
					log.Printf("unmarshal schedule for med %s: %v", m.ID, err)
				} else {
					resp.ScheduleTimes = times
				}
			}
			if resp.ScheduleTimes == nil {
				resp.ScheduleTimes = []string{}
			}

			// Compute next_dose_at if we have schedule and timezone
			resp.NextDoseAt = computeNextDoseAt(resp.ScheduleTimes, m.Timezone)

			upcomingMeds = append(upcomingMeds, resp)
		}

		result := dashboardResponseJSON{
			SummaryCards:    summary,
			StoolColorTrend: trend,
			UpcomingMeds:    upcomingMeds,
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// computeNextDoseAt calculates the next dose time based on schedule_times and timezone.
// Returns nil if no schedule or timezone is available.
func computeNextDoseAt(scheduleTimes []string, tz *string) *string {
	if len(scheduleTimes) == 0 || tz == nil {
		return nil
	}

	loc, err := time.LoadLocation(*tz)
	if err != nil {
		return nil
	}

	now := time.Now().In(loc)
	todayStr := now.Format("2006-01-02")

	var earliest time.Time
	found := false

	// Check today's remaining schedule times
	for _, st := range scheduleTimes {
		t, err := time.ParseInLocation("2006-01-02 15:04", todayStr+" "+st, loc)
		if err != nil {
			continue
		}
		if t.After(now) && (!found || t.Before(earliest)) {
			earliest = t
			found = true
		}
	}

	// If no remaining times today, use tomorrow's first time
	if !found {
		tomorrowStr := now.AddDate(0, 0, 1).Format("2006-01-02")
		for _, st := range scheduleTimes {
			t, err := time.ParseInLocation("2006-01-02 15:04", tomorrowStr+" "+st, loc)
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

	s := earliest.UTC().Format("2006-01-02T15:04:05Z")
	return &s
}
