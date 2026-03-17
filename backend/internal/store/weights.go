package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const weightColumns = `id, baby_id, logged_by, updated_by, timestamp,
	weight_kg, measurement_source, notes, created_at, updated_at`

// scanWeight scans a single weight row from the given scanner.
func scanWeight(s scanner) (*model.Weight, error) {
	var w model.Weight
	var updatedBy, measurementSource, notes sql.NullString
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&w.ID, &w.BabyID, &w.LoggedBy, &updatedBy, &tsStr,
		&w.WeightKg, &measurementSource, &notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	w.Timestamp, w.CreatedAt, w.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	w.UpdatedBy = nullStr(updatedBy)
	w.MeasurementSource = nullStr(measurementSource)
	w.Notes = nullStr(notes)

	return &w, nil
}

// CreateWeight inserts a new weight entry and returns it.
func CreateWeight(db *sql.DB, babyID, loggedBy, timestamp string, weightKg float64, measurementSource, notes *string) (*model.Weight, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO weights (id, baby_id, logged_by, timestamp, weight_kg, measurement_source, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, weightKg, measurementSource, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create weight: %w", err)
	}

	return GetWeightByID(db, babyID, id)
}

// GetWeightByID retrieves a weight by its ID, scoped to the given baby.
func GetWeightByID(db *sql.DB, babyID, weightID string) (*model.Weight, error) {
	row := db.QueryRow(
		`SELECT `+weightColumns+` FROM weights WHERE id = ? AND baby_id = ?`,
		weightID, babyID,
	)
	return scanWeight(row)
}

// ListWeights returns a paginated list of weights for a baby in ULID descending order.
func ListWeights(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.Weight], error) {
	return ListWeightsWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListWeightsWithTZ returns a paginated list of weights with timezone-aware date filtering.
func ListWeightsWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.Weight], error) {
	return listMetricWithTZ(db, "weights", weightColumns, babyID, from, to, cursor, limit, loc, scanWeight, func(w *model.Weight) string { return w.ID })
}

// UpdateWeight updates a weight entry.
func UpdateWeight(db *sql.DB, babyID, weightID, updatedBy, timestamp string, weightKg float64, measurementSource, notes *string) (*model.Weight, error) {
	res, err := db.Exec(
		`UPDATE weights SET
			updated_by = ?, timestamp = ?, weight_kg = ?,
			measurement_source = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, weightKg,
		measurementSource, notes,
		weightID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update weight: %w", err)
	}

	if err := checkRowsAffected(res, "update weight"); err != nil {
		return nil, err
	}

	return GetWeightByID(db, babyID, weightID)
}

// DeleteWeight hard-deletes a weight entry.
func DeleteWeight(db *sql.DB, babyID, weightID string) error {
	return deleteByID(db, "weights", babyID, weightID)
}
