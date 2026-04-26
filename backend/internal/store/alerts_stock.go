package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// getLowStockAlerts emits one alert per medication whose total remaining doses
// (sum of quantity_remaining across non-depleted containers, divided by
// dose_amount) is at or below the low_stock_threshold (default 3).
// Medications without dose_amount/dose_unit are skipped.
func getLowStockAlerts(db *sql.DB, babyID string) ([]Alert, error) {
	rows, err := db.Query(
		`SELECT m.id, m.name, m.dose_amount, m.low_stock_threshold,
		        COALESCE(SUM(c.quantity_remaining), 0) AS total_remaining
		 FROM medications m
		 LEFT JOIN medication_containers c
		   ON c.medication_id = m.id AND c.depleted = 0
		 WHERE m.baby_id = ? AND m.active = 1
		   AND m.dose_amount IS NOT NULL AND m.dose_unit IS NOT NULL
		 GROUP BY m.id, m.name, m.dose_amount, m.low_stock_threshold`,
		babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("low stock query: %w", err)
	}
	defer rows.Close()

	var alerts []Alert
	now := time.Now().UTC().Format(model.DateTimeFormat)
	for rows.Next() {
		var medID, name string
		var doseAmount float64
		var threshold sql.NullInt64
		var totalRemaining float64
		if err := rows.Scan(&medID, &name, &doseAmount, &threshold, &totalRemaining); err != nil {
			return nil, fmt.Errorf("scan low stock row: %w", err)
		}
		if doseAmount <= 0 {
			continue
		}
		thr := defaultLowStockThreshold
		if threshold.Valid {
			thr = int(threshold.Int64)
		}
		dosesLeft := totalRemaining / doseAmount
		if dosesLeft > float64(thr) {
			continue
		}
		alerts = append(alerts, Alert{
			EntryID:        fmt.Sprintf("%s_low_stock", medID),
			AlertType:      AlertTypeLowStock,
			Value:          dosesLeft,
			Timestamp:      now,
			MedicationID:   medID,
			MedicationName: name,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("low stock rows iteration: %w", err)
	}
	return alerts, nil
}

// getNearExpiryAlerts emits one alert per non-depleted container whose
// effective expiry (the earlier of manufacturer expiration_date and
// opened_at + max_days_after_opening) is within the medication's
// expiry_warning_days (default 3) or already past.
// Containers with no expiry information are skipped.
func getNearExpiryAlerts(db *sql.DB, babyID string) ([]Alert, error) {
	rows, err := db.Query(
		`SELECT c.id, c.medication_id, m.name,
		        c.opened_at, c.max_days_after_opening, c.expiration_date,
		        m.expiry_warning_days
		 FROM medication_containers c
		 JOIN medications m ON m.id = c.medication_id
		 WHERE c.baby_id = ? AND c.depleted = 0`,
		babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("near expiry query: %w", err)
	}
	defer rows.Close()

	now := time.Now().UTC()
	var alerts []Alert
	for rows.Next() {
		var containerID, medID, medName string
		var openedAtStr, expirationDate sql.NullString
		var maxDays, warnDays sql.NullInt64
		if err := rows.Scan(
			&containerID, &medID, &medName,
			&openedAtStr, &maxDays, &expirationDate, &warnDays,
		); err != nil {
			return nil, fmt.Errorf("scan near expiry row: %w", err)
		}

		eff, ok := computeEffectiveExpiry(openedAtStr, maxDays, expirationDate)
		if !ok {
			continue
		}

		warn := defaultExpiryWarningDays
		if warnDays.Valid {
			warn = int(warnDays.Int64)
		}

		threshold := now.AddDate(0, 0, warn)
		if eff.After(threshold) {
			continue
		}

		alerts = append(alerts, Alert{
			EntryID:        fmt.Sprintf("%s_near_expiry", containerID),
			AlertType:      AlertTypeNearExpiry,
			Value:          eff.Format(model.DateFormat),
			Timestamp:      now.Format(model.DateTimeFormat),
			MedicationID:   medID,
			MedicationName: medName,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("near expiry rows iteration: %w", err)
	}
	return alerts, nil
}

// computeEffectiveExpiry returns the earliest valid expiry date for a container,
// considering both manufacturer expiration_date and opened_at + max_days_after_opening.
// Returns ok=false when neither source provides an expiry date.
func computeEffectiveExpiry(openedAtStr sql.NullString, maxDays sql.NullInt64, expirationDate sql.NullString) (time.Time, bool) {
	var candidate time.Time
	have := false

	if expirationDate.Valid {
		t, err := time.Parse(model.DateFormat, expirationDate.String)
		if err == nil {
			// End-of-day: a container marked YYYY-MM-DD is good through that whole day.
			candidate = t.AddDate(0, 0, 1).Add(-time.Second)
			have = true
		}
	}

	if openedAtStr.Valid && maxDays.Valid {
		opened, err := ParseTime(openedAtStr.String)
		if err == nil {
			postOpen := opened.AddDate(0, 0, int(maxDays.Int64))
			if !have || postOpen.Before(candidate) {
				candidate = postOpen
				have = true
			}
		}
	}

	return candidate, have
}
