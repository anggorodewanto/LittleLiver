package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// Alert type constants.
const (
	AlertTypeAcholicStool      = "acholic_stool"
	AlertTypeFever             = "fever"
	AlertTypeJaundiceWorsening = "jaundice_worsening"
	AlertTypeMissedMedication  = "missed_medication"
)

// Alert represents a single active alert for the dashboard.
type Alert struct {
	EntryID        string  `json:"entry_id"`
	AlertType      string  `json:"alert_type"`
	Method         *string `json:"method,omitempty"`
	Value          any     `json:"value"`
	Timestamp      string  `json:"timestamp"`
	MedicationID   string  `json:"medication_id,omitempty"`
	MedicationName string  `json:"medication_name,omitempty"`
}

// FeverThreshold returns the fever threshold for the given temperature method.
func FeverThreshold(method string) float64 {
	switch method {
	case "rectal":
		return 38.0
	case "axillary":
		return 37.5
	case "ear":
		return 38.0
	case "forehead":
		return 37.5
	default:
		return 38.0
	}
}

// IsDoseCovered checks whether a scheduled dose (as a UTC time) has a corresponding
// med_log entry. A dose is covered if:
//  1. A med_log has a scheduled_time matching the dose slot (minute precision), OR
//  2. A given dose (no scheduled_time) has given_at within [windowStart, scheduled+6h], OR
//  3. A skipped dose (no scheduled_time) was created within [windowStart, scheduled+6h].
//
// windowStartUTC defines how far back to look for matching doses. Pass zero time
// to use a default of 30 minutes before the scheduled time.
func IsDoseCovered(db *sql.DB, medicationID string, scheduledUTC time.Time, windowStartUTC ...time.Time) (bool, error) {
	ws := scheduledUTC.Add(-30 * time.Minute)
	if len(windowStartUTC) > 0 && !windowStartUTC[0].IsZero() {
		ws = windowStartUTC[0]
	}
	windowStart := ws.Format(model.DateTimeFormat)
	windowEnd := scheduledUTC.Add(6 * time.Hour).Format(model.DateTimeFormat)
	scheduledStr := scheduledUTC.Format(model.DateTimeFormat)

	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM med_logs
		WHERE medication_id = ?
		AND (
			(scheduled_time IS NOT NULL AND strftime('%Y-%m-%dT%H:%M', scheduled_time) = strftime('%Y-%m-%dT%H:%M', ?))
			OR
			(scheduled_time IS NULL AND skipped = 0 AND datetime(given_at) >= datetime(?) AND datetime(given_at) <= datetime(?))
			OR
			(scheduled_time IS NULL AND skipped = 1 AND datetime(created_at) >= datetime(?) AND datetime(created_at) <= datetime(?))
		)`,
		medicationID, scheduledStr, windowStart, windowEnd, windowStart, windowEnd,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check dose covered: %w", err)
	}
	return count > 0, nil
}

// GetActiveAlerts computes all active alerts for a baby. Alerts are global (ignore from/to).
func GetActiveAlerts(db *sql.DB, babyID string) ([]Alert, error) {
	var alerts []Alert

	// 1. Acholic stool alert: most recent stool with color_rating <= 3
	acholicAlerts, err := getAcholicStoolAlert(db, babyID)
	if err != nil {
		return nil, err
	}
	alerts = append(alerts, acholicAlerts...)

	// 2. Fever alert: most recent temperature exceeding method-specific threshold
	feverAlert, err := getFeverAlert(db, babyID)
	if err != nil {
		return nil, err
	}
	if feverAlert != nil {
		alerts = append(alerts, *feverAlert)
	}

	// 3. Jaundice worsening: most recent skin observation with severe or scleral icterus
	jaundiceAlert, err := getJaundiceAlert(db, babyID)
	if err != nil {
		return nil, err
	}
	if jaundiceAlert != nil {
		alerts = append(alerts, *jaundiceAlert)
	}

	// 4. Missed medication: scheduled doses >30 min past due without med_log
	missedAlerts, err := getMissedMedicationAlerts(db, babyID)
	if err != nil {
		return nil, err
	}
	alerts = append(alerts, missedAlerts...)

	return emptySliceIfNil(alerts), nil
}

func getAcholicStoolAlert(db *sql.DB, babyID string) ([]Alert, error) {
	var id string
	var colorRating int
	var tsStr string

	err := db.QueryRow(`
		SELECT id, color_rating, timestamp FROM stools
		WHERE baby_id = ?
		ORDER BY timestamp DESC, id DESC LIMIT 1`,
		babyID,
	).Scan(&id, &colorRating, &tsStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("acholic stool alert: %w", err)
	}

	if colorRating >= 4 {
		return nil, nil
	}

	return []Alert{{
		EntryID:   id,
		AlertType: AlertTypeAcholicStool,
		Value:     colorRating,
		Timestamp: tsStr,
	}}, nil
}

func getFeverAlert(db *sql.DB, babyID string) (*Alert, error) {
	var id string
	var value float64
	var method string
	var tsStr string

	err := db.QueryRow(`
		SELECT id, value, method, timestamp FROM temperatures
		WHERE baby_id = ?
		ORDER BY timestamp DESC, id DESC LIMIT 1`,
		babyID,
	).Scan(&id, &value, &method, &tsStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fever alert: %w", err)
	}

	threshold := FeverThreshold(method)
	if value < threshold {
		return nil, nil
	}

	return &Alert{
		EntryID:   id,
		AlertType: AlertTypeFever,
		Method:    &method,
		Value:     value,
		Timestamp: tsStr,
	}, nil
}

func getJaundiceAlert(db *sql.DB, babyID string) (*Alert, error) {
	var id string
	var jaundiceLevel sql.NullString
	var scleralIcterus bool
	var tsStr string

	err := db.QueryRow(`
		SELECT id, jaundice_level, scleral_icterus, timestamp FROM skin_observations
		WHERE baby_id = ?
		ORDER BY timestamp DESC, id DESC LIMIT 1`,
		babyID,
	).Scan(&id, &jaundiceLevel, &scleralIcterus, &tsStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("jaundice alert: %w", err)
	}

	isSevere := jaundiceLevel.Valid && jaundiceLevel.String == "severe_limbs_and_trunk"
	if !isSevere && !scleralIcterus {
		return nil, nil
	}

	var value string
	if isSevere && scleralIcterus {
		value = "severe_limbs_and_trunk+scleral_icterus"
	} else if isSevere {
		value = "severe_limbs_and_trunk"
	} else {
		value = "scleral_icterus"
	}

	return &Alert{
		EntryID:   id,
		AlertType: AlertTypeJaundiceWorsening,
		Value:     value,
		Timestamp: tsStr,
	}, nil
}

// medScheduleInfo holds info needed to check missed medication doses.
type medScheduleInfo struct {
	ID           string
	Name         string
	Frequency    string
	Schedule     string
	Timezone     string
	IntervalDays *int
	CreatedAt    time.Time
}

func getMissedMedicationAlerts(db *sql.DB, babyID string) ([]Alert, error) {
	// Get all active medications for this baby that have a timezone.
	// Schedule-based meds also need schedule IS NOT NULL; every_x_days meds need interval_days.
	// Collect results first to avoid holding open rows cursor during IsDoseCovered queries.
	rows, err := db.Query(`
		SELECT id, name, frequency, schedule, timezone, interval_days, created_at FROM medications
		WHERE baby_id = ? AND active = 1 AND timezone IS NOT NULL
		AND (schedule IS NOT NULL OR frequency = 'every_x_days')`,
		babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("missed medication query: %w", err)
	}

	var meds []medScheduleInfo
	for rows.Next() {
		var m medScheduleInfo
		var schedule sql.NullString
		var intervalDays sql.NullInt64
		var createdAtStr string
		if err := rows.Scan(&m.ID, &m.Name, &m.Frequency, &schedule, &m.Timezone, &intervalDays, &createdAtStr); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan medication: %w", err)
		}
		if schedule.Valid {
			m.Schedule = schedule.String
		}
		if intervalDays.Valid {
			v := int(intervalDays.Int64)
			m.IntervalDays = &v
		}
		ca, err := time.Parse(model.DateTimeFormat, createdAtStr)
		if err != nil {
			// Try alternate SQLite format
			ca, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("parse created_at for medication %s: %w", m.ID, err)
			}
		}
		m.CreatedAt = ca
		meds = append(meds, m)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	rows.Close()

	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour)

	var alerts []Alert

	for _, m := range meds {
		loc, err := time.LoadLocation(m.Timezone)
		if err != nil {
			log.Printf("invalid timezone %s for medication %s: %v", m.Timezone, m.ID, err)
			continue
		}

		// Handle every_x_days medications separately.
		if m.Frequency == "every_x_days" {
			intervalAlerts, err := checkEveryXDaysMissed(db, m, loc, now)
			if err != nil {
				return nil, err
			}
			alerts = append(alerts, intervalAlerts...)
			continue
		}

		var scheduleTimes []string
		if err := json.Unmarshal([]byte(m.Schedule), &scheduleTimes); err != nil {
			log.Printf("invalid schedule JSON for medication %s: %v", m.ID, err)
			continue
		}
		sort.Strings(scheduleTimes)

		// Expand schedule into concrete UTC datetimes for the last 24 hours.
		// Check today and yesterday in the medication's timezone.
		nowLocal := now.In(loc)
		todayLocal := nowLocal.Format(model.DateFormat)
		yesterdayLocal := nowLocal.AddDate(0, 0, -1).Format(model.DateFormat)

		for _, day := range []string{yesterdayLocal, todayLocal} {
			for i, st := range scheduleTimes {
				t, err := time.ParseInLocation(model.DateFormat+" 15:04", day+" "+st, loc)
				if err != nil {
					continue
				}
				scheduledUTC := t.UTC()

				// Must be within last 24 hours
				if scheduledUTC.Before(cutoff) {
					continue
				}

				// Skip doses scheduled before the medication was created
				if scheduledUTC.Before(m.CreatedAt) {
					continue
				}

				// Must be >30 min past due
				if now.Sub(scheduledUTC) <= 30*time.Minute {
					continue
				}

				coverStart := SlotCoverageStart(scheduleTimes, i, day, loc).UTC()
				covered, err := IsDoseCovered(db, m.ID, scheduledUTC, coverStart)
				if err != nil {
					return nil, err
				}
				if covered {
					continue
				}

				alerts = append(alerts, Alert{
					EntryID:        fmt.Sprintf("%s_%s", m.ID, scheduledUTC.Format("20060102T150405Z")),
					AlertType:      AlertTypeMissedMedication,
					Value:          st,
					Timestamp:      scheduledUTC.Format(model.DateTimeFormat),
					MedicationID:   m.ID,
					MedicationName: m.Name,
				})
			}
		}
	}

	return alerts, nil
}

// checkEveryXDaysMissed checks if an every_x_days medication has a missed dose.
// A dose is "missed" only after the due date has fully passed (now > end of due day)
// and no med_log entry exists with given_at on that calendar day.
func checkEveryXDaysMissed(db *sql.DB, m medScheduleInfo, loc *time.Location, now time.Time) ([]Alert, error) {
	if m.IntervalDays == nil {
		return nil, nil
	}

	// Find last given_at for this medication
	var lastGivenAtStr sql.NullString
	err := db.QueryRow(
		`SELECT MAX(given_at) FROM med_logs WHERE medication_id = ? AND skipped = 0`,
		m.ID,
	).Scan(&lastGivenAtStr)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query last given_at for %s: %w", m.ID, err)
	}

	anchor := m.CreatedAt
	if lastGivenAtStr.Valid {
		lga, err := time.Parse(model.DateTimeFormat, lastGivenAtStr.String)
		if err != nil {
			lga, err = time.Parse("2006-01-02 15:04:05", lastGivenAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("parse last_given_at for %s: %w", m.ID, err)
			}
		}
		anchor = lga
	}

	// Compute due date
	anchorLocal := anchor.In(loc)
	anchorDate := time.Date(anchorLocal.Year(), anchorLocal.Month(), anchorLocal.Day(), 0, 0, 0, 0, loc)
	dueDate := anchorDate.AddDate(0, 0, *m.IntervalDays)

	// Due date must have fully passed (now is past end of due day)
	endOfDueDay := dueDate.AddDate(0, 0, 1) // midnight of the day after
	nowLocal := now.In(loc)
	if nowLocal.Before(endOfDueDay) {
		return nil, nil
	}

	// Check if any dose was logged on the due date
	covered, err := isDayCovered(db, m.ID, dueDate, loc)
	if err != nil {
		return nil, err
	}
	if covered {
		return nil, nil
	}

	dueDateStr := dueDate.Format(model.DateFormat)
	return []Alert{{
		EntryID:        fmt.Sprintf("%s_interval_%s", m.ID, dueDateStr),
		AlertType:      AlertTypeMissedMedication,
		Value:          dueDateStr,
		Timestamp:      dueDate.UTC().Format(model.DateTimeFormat),
		MedicationID:   m.ID,
		MedicationName: m.Name,
	}}, nil
}

// isDayCovered checks if any med_log entry (given or skipped) exists for a medication
// on the given calendar day in the given timezone.
func isDayCovered(db *sql.DB, medicationID string, dayStart time.Time, loc *time.Location) (bool, error) {
	dayEnd := dayStart.AddDate(0, 0, 1)
	startUTC := dayStart.UTC().Format(model.DateTimeFormat)
	endUTC := dayEnd.UTC().Format(model.DateTimeFormat)

	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM med_logs
		WHERE medication_id = ?
		AND (
			(skipped = 0 AND datetime(given_at) >= datetime(?) AND datetime(given_at) < datetime(?))
			OR
			(skipped = 1 AND datetime(created_at) >= datetime(?) AND datetime(created_at) < datetime(?))
		)`,
		medicationID, startUTC, endUTC, startUTC, endUTC,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check day covered: %w", err)
	}
	return count > 0, nil
}
