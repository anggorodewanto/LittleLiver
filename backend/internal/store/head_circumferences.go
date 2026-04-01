package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const headCircumferenceColumns = `id, baby_id, logged_by, updated_by, timestamp,
	circumference_cm, measurement_source, notes, created_at, updated_at`

// scanHeadCircumference scans a single head circumference row from the given scanner.
func scanHeadCircumference(s scanner) (*model.HeadCircumference, error) {
	var hc model.HeadCircumference
	var updatedBy, measurementSource, notes sql.NullString
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&hc.ID, &hc.BabyID, &hc.LoggedBy, &updatedBy, &tsStr,
		&hc.CircumferenceCm, &measurementSource, &notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	hc.Timestamp, hc.CreatedAt, hc.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	hc.UpdatedBy = nullStr(updatedBy)
	hc.MeasurementSource = nullStr(measurementSource)
	hc.Notes = nullStr(notes)

	return &hc, nil
}

// CreateHeadCircumference inserts a new head circumference entry and returns it.
func CreateHeadCircumference(db *sql.DB, babyID, loggedBy, timestamp string, circumferenceCm float64, measurementSource, notes *string) (*model.HeadCircumference, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO head_circumferences (id, baby_id, logged_by, timestamp, circumference_cm, measurement_source, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, circumferenceCm, measurementSource, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create head circumference: %w", err)
	}

	return GetHeadCircumferenceByID(db, babyID, id)
}

// GetHeadCircumferenceByID retrieves a head circumference by its ID, scoped to the given baby.
func GetHeadCircumferenceByID(db *sql.DB, babyID, id string) (*model.HeadCircumference, error) {
	row := db.QueryRow(
		`SELECT `+headCircumferenceColumns+` FROM head_circumferences WHERE id = ? AND baby_id = ?`,
		id, babyID,
	)
	return scanHeadCircumference(row)
}

// ListHeadCircumferences returns a paginated list of head circumferences for a baby in ULID descending order.
func ListHeadCircumferences(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.HeadCircumference], error) {
	return ListHeadCircumferencesWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListHeadCircumferencesWithTZ returns a paginated list of head circumferences with timezone-aware date filtering.
func ListHeadCircumferencesWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.HeadCircumference], error) {
	return listMetricWithTZ(db, "head_circumferences", headCircumferenceColumns, babyID, from, to, cursor, limit, loc, scanHeadCircumference, func(hc *model.HeadCircumference) string { return hc.ID })
}

// UpdateHeadCircumference updates a head circumference entry.
func UpdateHeadCircumference(db *sql.DB, babyID, id, updatedBy, timestamp string, circumferenceCm float64, measurementSource, notes *string) (*model.HeadCircumference, error) {
	res, err := db.Exec(
		`UPDATE head_circumferences SET
			updated_by = ?, timestamp = ?, circumference_cm = ?,
			measurement_source = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, circumferenceCm,
		measurementSource, notes,
		id, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update head circumference: %w", err)
	}

	if err := checkRowsAffected(res, "update head circumference"); err != nil {
		return nil, err
	}

	return GetHeadCircumferenceByID(db, babyID, id)
}

// DeleteHeadCircumference hard-deletes a head circumference entry.
func DeleteHeadCircumference(db *sql.DB, babyID, id string) error {
	return deleteByID(db, "head_circumferences", babyID, id)
}
