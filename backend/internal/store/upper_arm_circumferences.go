package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const upperArmCircumferenceColumns = `id, baby_id, logged_by, updated_by, timestamp,
	circumference_cm, measurement_source, notes, created_at, updated_at`

// scanUpperArmCircumference scans a single upper arm circumference row from the given scanner.
func scanUpperArmCircumference(s scanner) (*model.UpperArmCircumference, error) {
	var u model.UpperArmCircumference
	var updatedBy, measurementSource, notes sql.NullString
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&u.ID, &u.BabyID, &u.LoggedBy, &updatedBy, &tsStr,
		&u.CircumferenceCm, &measurementSource, &notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	u.Timestamp, u.CreatedAt, u.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	u.UpdatedBy = nullStr(updatedBy)
	u.MeasurementSource = nullStr(measurementSource)
	u.Notes = nullStr(notes)

	return &u, nil
}

// CreateUpperArmCircumference inserts a new upper arm circumference entry and returns it.
func CreateUpperArmCircumference(db *sql.DB, babyID, loggedBy, timestamp string, circumferenceCm float64, measurementSource, notes *string) (*model.UpperArmCircumference, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO upper_arm_circumferences (id, baby_id, logged_by, timestamp, circumference_cm, measurement_source, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, circumferenceCm, measurementSource, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create upper arm circumference: %w", err)
	}

	return GetUpperArmCircumferenceByID(db, babyID, id)
}

// GetUpperArmCircumferenceByID retrieves an upper arm circumference by its ID, scoped to the given baby.
func GetUpperArmCircumferenceByID(db *sql.DB, babyID, id string) (*model.UpperArmCircumference, error) {
	row := db.QueryRow(
		`SELECT `+upperArmCircumferenceColumns+` FROM upper_arm_circumferences WHERE id = ? AND baby_id = ?`,
		id, babyID,
	)
	return scanUpperArmCircumference(row)
}

// ListUpperArmCircumferences returns a paginated list of upper arm circumferences for a baby in ULID descending order.
func ListUpperArmCircumferences(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.UpperArmCircumference], error) {
	return ListUpperArmCircumferencesWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListUpperArmCircumferencesWithTZ returns a paginated list of upper arm circumferences with timezone-aware date filtering.
func ListUpperArmCircumferencesWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.UpperArmCircumference], error) {
	return listMetricWithTZ(db, "upper_arm_circumferences", upperArmCircumferenceColumns, babyID, from, to, cursor, limit, loc, scanUpperArmCircumference, func(u *model.UpperArmCircumference) string { return u.ID })
}

// UpdateUpperArmCircumference updates an upper arm circumference entry.
func UpdateUpperArmCircumference(db *sql.DB, babyID, id, updatedBy, timestamp string, circumferenceCm float64, measurementSource, notes *string) (*model.UpperArmCircumference, error) {
	res, err := db.Exec(
		`UPDATE upper_arm_circumferences SET
			updated_by = ?, timestamp = ?, circumference_cm = ?,
			measurement_source = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, circumferenceCm,
		measurementSource, notes,
		id, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update upper arm circumference: %w", err)
	}

	if err := checkRowsAffected(res, "update upper arm circumference"); err != nil {
		return nil, err
	}

	return GetUpperArmCircumferenceByID(db, babyID, id)
}

// DeleteUpperArmCircumference hard-deletes an upper arm circumference entry.
func DeleteUpperArmCircumference(db *sql.DB, babyID, id string) error {
	return deleteByID(db, "upper_arm_circumferences", babyID, id)
}
