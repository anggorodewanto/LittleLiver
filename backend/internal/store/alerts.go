package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// Alert represents a single active alert for the dashboard.
type Alert struct {
	EntryID   string  `json:"entry_id"`
	AlertType string  `json:"alert_type"`
	Method    *string `json:"method,omitempty"`
	Value     any     `json:"value"`
	Timestamp string  `json:"timestamp"`
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
// med_log entry (given or skipped) within +/-30 min of the scheduled time.
// This is the shared suppression utility reusable by the scheduler (Phase 34).
func IsDoseCovered(db *sql.DB, medicationID string, scheduledUTC time.Time) (bool, error) {
	windowStart := scheduledUTC.Add(-30 * time.Minute).Format(model.DateTimeFormat)
	windowEnd := scheduledUTC.Add(30 * time.Minute).Format(model.DateTimeFormat)

	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM med_logs
		WHERE medication_id = ?
		AND (
			(skipped = 0 AND datetime(given_at) >= datetime(?) AND datetime(given_at) <= datetime(?))
			OR
			(skipped = 1 AND datetime(created_at) >= datetime(?) AND datetime(created_at) <= datetime(?))
		)`,
		medicationID, windowStart, windowEnd, windowStart, windowEnd,
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

	if alerts == nil {
		alerts = make([]Alert, 0)
	}
	return alerts, nil
}

func getAcholicStoolAlert(db *sql.DB, babyID string) ([]Alert, error) {
	var id string
	var colorRating int
	var tsStr string

	err := db.QueryRow(`
		SELECT id, color_rating, timestamp FROM stools
		WHERE baby_id = ?
		ORDER BY timestamp DESC LIMIT 1`,
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
		AlertType: "acholic_stool",
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
		ORDER BY timestamp DESC LIMIT 1`,
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
		AlertType: "fever",
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
		ORDER BY timestamp DESC LIMIT 1`,
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
		AlertType: "jaundice_worsening",
		Value:     value,
		Timestamp: tsStr,
	}, nil
}

// medScheduleInfo holds info needed to check missed medication doses.
type medScheduleInfo struct {
	ID       string
	Schedule string
	Timezone string
}

func getMissedMedicationAlerts(db *sql.DB, babyID string) ([]Alert, error) {
	// Get all active medications for this baby that have a schedule and timezone.
	// Collect results first to avoid holding open rows cursor during IsDoseCovered queries.
	rows, err := db.Query(`
		SELECT id, schedule, timezone FROM medications
		WHERE baby_id = ? AND active = 1 AND schedule IS NOT NULL AND timezone IS NOT NULL`,
		babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("missed medication query: %w", err)
	}

	var meds []medScheduleInfo
	for rows.Next() {
		var m medScheduleInfo
		if err := rows.Scan(&m.ID, &m.Schedule, &m.Timezone); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan medication: %w", err)
		}
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

		var scheduleTimes []string
		if err := json.Unmarshal([]byte(m.Schedule), &scheduleTimes); err != nil {
			log.Printf("invalid schedule JSON for medication %s: %v", m.ID, err)
			continue
		}

		// Expand schedule into concrete UTC datetimes for the last 24 hours.
		// Check today and yesterday in the medication's timezone.
		nowLocal := now.In(loc)
		todayLocal := nowLocal.Format(model.DateFormat)
		yesterdayLocal := nowLocal.AddDate(0, 0, -1).Format(model.DateFormat)

		for _, day := range []string{yesterdayLocal, todayLocal} {
			for _, st := range scheduleTimes {
				t, err := time.ParseInLocation(model.DateFormat+" 15:04", day+" "+st, loc)
				if err != nil {
					continue
				}
				scheduledUTC := t.UTC()

				// Must be within last 24 hours
				if scheduledUTC.Before(cutoff) {
					continue
				}

				// Must be >30 min past due
				if now.Sub(scheduledUTC) <= 30*time.Minute {
					continue
				}

				covered, err := IsDoseCovered(db, m.ID, scheduledUTC)
				if err != nil {
					return nil, err
				}
				if covered {
					continue
				}

				alerts = append(alerts, Alert{
					EntryID:   m.ID,
					AlertType: "missed_medication",
					Value:     st,
					Timestamp: scheduledUTC.Format(model.DateTimeFormat),
				})
			}
		}
	}

	return alerts, nil
}
