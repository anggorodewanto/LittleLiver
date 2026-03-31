package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// DashboardSummary holds the aggregated summary cards for the dashboard.
type DashboardSummary struct {
	TotalFeeds     int      `json:"total_feeds"`
	TotalCalories  float64  `json:"total_calories"`
	TotalWetDiapers int      `json:"total_wet_diapers"`
	TotalStools     int      `json:"total_stools"`
	WorstStoolColor *int     `json:"worst_stool_color"`
	LastTemperature *float64 `json:"last_temperature"`
	LastWeight     *float64 `json:"last_weight"`
}

// StoolColorEntry represents a single date+color entry in the stool color trend.
type StoolColorEntry struct {
	Date        string `json:"date"`
	Color       string `json:"color"`
	ColorRating int    `json:"color_rating"`
}

// UpcomingMed represents an active medication with schedule info.
type UpcomingMed struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Dose         string     `json:"dose"`
	Frequency    string     `json:"frequency"`
	Schedule     *string    `json:"schedule"`
	Timezone     *string    `json:"timezone"`
	IntervalDays *int       `json:"interval_days,omitempty"`
	StartsFrom   *string    `json:"starts_from,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	LastGivenAt  *time.Time `json:"last_given_at,omitempty"`
}

// GetDashboardSummary returns aggregated summary cards for a baby within the given date range.
// from and to are in YYYY-MM-DD format. loc specifies the timezone for date interpretation.
// last_temp and last_weight ignore the date range.
func GetDashboardSummary(db *sql.DB, babyID, from, to string, loc *time.Location) (*DashboardSummary, error) {
	fromTime, toTime, err := ParseDateRangeInLocation(from, to, loc)
	if err != nil {
		return nil, err
	}

	s := &DashboardSummary{}

	// Feeds count and calories sum in date range
	err = db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(calories), 0)
		 FROM feedings
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?`,
		babyID, fromTime, toTime,
	).Scan(&s.TotalFeeds, &s.TotalCalories)
	if err != nil {
		return nil, fmt.Errorf("query feedings summary: %w", err)
	}

	// Wet diapers count in date range
	err = db.QueryRow(
		`SELECT COUNT(*)
		 FROM urine
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?`,
		babyID, fromTime, toTime,
	).Scan(&s.TotalWetDiapers)
	if err != nil {
		return nil, fmt.Errorf("query urine summary: %w", err)
	}

	// Stools count in date range
	err = db.QueryRow(
		`SELECT COUNT(*)
		 FROM stools
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?`,
		babyID, fromTime, toTime,
	).Scan(&s.TotalStools)
	if err != nil {
		return nil, fmt.Errorf("query stools summary: %w", err)
	}

	// Worst (lowest) stool color rating in date range
	var worstColor sql.NullInt64
	err = db.QueryRow(
		`SELECT MIN(color_rating)
		 FROM stools
		 WHERE baby_id = ? AND timestamp >= ? AND timestamp < ?`,
		babyID, fromTime, toTime,
	).Scan(&worstColor)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query worst stool color: %w", err)
	}
	if worstColor.Valid {
		v := int(worstColor.Int64)
		s.WorstStoolColor = &v
	}

	// Last temperature (regardless of date range)
	var lastTemp sql.NullFloat64
	err = db.QueryRow(
		`SELECT value FROM temperatures
		 WHERE baby_id = ?
		 ORDER BY timestamp DESC LIMIT 1`,
		babyID,
	).Scan(&lastTemp)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query last temperature: %w", err)
	}
	s.LastTemperature = nullFloat(lastTemp)

	// Last weight (regardless of date range)
	var lastWeight sql.NullFloat64
	err = db.QueryRow(
		`SELECT weight_kg FROM weights
		 WHERE baby_id = ?
		 ORDER BY timestamp DESC LIMIT 1`,
		babyID,
	).Scan(&lastWeight)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query last weight: %w", err)
	}
	s.LastWeight = nullFloat(lastWeight)

	return s, nil
}

// GetStoolColorTrend returns stool color entries for the last 7 days, regardless of any params.
// Returns one entry per stool (multiple per day possible), ordered by date descending.
// loc specifies the user's timezone for computing the 7-day window and grouping by date.
func GetStoolColorTrend(db *sql.DB, babyID string, loc *time.Location) ([]StoolColorEntry, error) {
	now := time.Now().In(loc)
	sevenDaysAgo := time.Date(now.Year(), now.Month(), now.Day()-6, 0, 0, 0, 0, loc)
	sevenDaysAgoUTC := sevenDaysAgo.UTC().Format(model.DateTimeFormat)
	offsetSec := tzOffsetSeconds(sevenDaysAgo.Format(model.DateFormat), loc)

	rows, err := db.Query(
		`SELECT DATE(datetime(timestamp, ? || ' seconds')) as date, color_label, color_rating
		 FROM stools
		 WHERE baby_id = ? AND timestamp >= ?
		 ORDER BY timestamp DESC`,
		offsetSec, babyID, sevenDaysAgoUTC,
	)
	if err != nil {
		return nil, fmt.Errorf("query stool color trend: %w", err)
	}
	defer rows.Close()

	var entries []StoolColorEntry
	for rows.Next() {
		var e StoolColorEntry
		var colorLabel sql.NullString
		if err := rows.Scan(&e.Date, &colorLabel, &e.ColorRating); err != nil {
			return nil, fmt.Errorf("scan stool color trend: %w", err)
		}
		if colorLabel.Valid {
			e.Color = colorLabel.String
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if entries == nil {
		entries = make([]StoolColorEntry, 0)
	}
	return entries, nil
}

// GetUpcomingMeds returns active medications for a baby, sorted by next scheduled dose time.
func GetUpcomingMeds(db *sql.DB, babyID string) ([]UpcomingMed, error) {
	rows, err := db.Query(
		`SELECT m.id, m.name, m.dose, m.frequency, m.schedule, m.timezone,
		        m.interval_days, m.starts_from, m.created_at,
		        (SELECT MAX(ml.given_at) FROM med_logs ml WHERE ml.medication_id = m.id AND ml.skipped = 0) as last_given_at
		 FROM medications m
		 WHERE m.baby_id = ? AND m.active = 1`,
		babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("query upcoming meds: %w", err)
	}
	defer rows.Close()

	var meds []UpcomingMed
	for rows.Next() {
		var m UpcomingMed
		var schedule, timezone, startsFrom sql.NullString
		var intervalDays sql.NullInt64
		var createdAtStr string
		var lastGivenAtStr sql.NullString
		if err := rows.Scan(&m.ID, &m.Name, &m.Dose, &m.Frequency, &schedule, &timezone,
			&intervalDays, &startsFrom, &createdAtStr, &lastGivenAtStr); err != nil {
			return nil, fmt.Errorf("scan upcoming med: %w", err)
		}
		m.Schedule = nullStr(schedule)
		m.Timezone = nullStr(timezone)
		m.StartsFrom = nullStr(startsFrom)
		if intervalDays.Valid {
			v := int(intervalDays.Int64)
			m.IntervalDays = &v
		}
		ca, err := ParseTime(createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("parse created_at for med %s: %w", m.ID, err)
		}
		m.CreatedAt = ca
		if lastGivenAtStr.Valid {
			lga, err := ParseTime(lastGivenAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("parse last_given_at for med %s: %w", m.ID, err)
			}
			m.LastGivenAt = &lga
		}
		meds = append(meds, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if meds == nil {
		meds = make([]UpcomingMed, 0)
	}

	sort.Slice(meds, func(i, j int) bool {
		ti := nextDoseTime(meds[i])
		tj := nextDoseTime(meds[j])
		return ti.Before(tj)
	})

	return meds, nil
}

// nextDoseTime computes the next scheduled dose time for a medication.
// If no schedule/timezone is set, returns a far-future sentinel so it sorts last.
func nextDoseTime(m UpcomingMed) time.Time {
	farFuture := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

	// Handle every_x_days frequency: due date is a whole calendar day.
	if m.Frequency == "every_x_days" {
		return nextDoseTimeInterval(m, farFuture)
	}

	if m.Schedule == nil || m.Timezone == nil {
		return farFuture
	}

	loc, err := time.LoadLocation(*m.Timezone)
	if err != nil {
		return farFuture
	}

	var times []string
	if err := json.Unmarshal([]byte(*m.Schedule), &times); err != nil {
		return farFuture
	}

	now := time.Now().In(loc)
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")

	var earliest time.Time
	for _, st := range times {
		// Check today first
		t, err := time.ParseInLocation("2006-01-02 15:04", today+" "+st, loc)
		if err != nil {
			continue
		}
		if t.After(now) {
			if earliest.IsZero() || t.Before(earliest) {
				earliest = t
			}
			continue
		}
		// Already passed today, use tomorrow
		t, err = time.ParseInLocation("2006-01-02 15:04", tomorrow+" "+st, loc)
		if err != nil {
			continue
		}
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
	}

	if earliest.IsZero() {
		return farFuture
	}
	return earliest
}

// nextDoseTimeInterval computes the next dose time for an every_x_days medication.
// Returns start-of-day (midnight) in the medication's timezone for the due date.
func nextDoseTimeInterval(m UpcomingMed, farFuture time.Time) time.Time {
	if m.IntervalDays == nil || m.Timezone == nil {
		return farFuture
	}

	loc, err := time.LoadLocation(*m.Timezone)
	if err != nil {
		return farFuture
	}

	// Anchor is the last given_at, or starts_from date, or created_at if no doses.
	anchor := m.CreatedAt
	if m.StartsFrom != nil {
		if sf, err := time.ParseInLocation("2006-01-02", *m.StartsFrom, loc); err == nil {
			anchor = sf
		}
	}
	if m.LastGivenAt != nil {
		anchor = *m.LastGivenAt
	}

	// Compute next due date: anchor date + interval_days, at midnight in med's timezone.
	anchorLocal := anchor.In(loc)
	anchorDate := time.Date(anchorLocal.Year(), anchorLocal.Month(), anchorLocal.Day(), 0, 0, 0, 0, loc)
	dueDate := anchorDate.AddDate(0, 0, *m.IntervalDays)

	return dueDate
}
