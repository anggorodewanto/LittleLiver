package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const heightColumns = `id, baby_id, logged_by, updated_by, timestamp,
	height_cm, measurement_source, notes, created_at, updated_at`

func scanHeight(s scanner) (*model.Height, error) {
	var h model.Height
	var updatedBy, measurementSource, notes sql.NullString
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&h.ID, &h.BabyID, &h.LoggedBy, &updatedBy, &tsStr,
		&h.HeightCm, &measurementSource, &notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	h.Timestamp, h.CreatedAt, h.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	h.UpdatedBy = nullStr(updatedBy)
	h.MeasurementSource = nullStr(measurementSource)
	h.Notes = nullStr(notes)

	return &h, nil
}

// CreateHeight inserts a new height entry and returns it.
func CreateHeight(db *sql.DB, babyID, loggedBy, timestamp string, heightCm float64, measurementSource, notes *string) (*model.Height, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO heights (id, baby_id, logged_by, timestamp, height_cm, measurement_source, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, heightCm, measurementSource, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create height: %w", err)
	}

	return GetHeightByID(db, babyID, id)
}

// GetHeightByID retrieves a height by its ID, scoped to the given baby.
func GetHeightByID(db *sql.DB, babyID, heightID string) (*model.Height, error) {
	row := db.QueryRow(
		`SELECT `+heightColumns+` FROM heights WHERE id = ? AND baby_id = ?`,
		heightID, babyID,
	)
	return scanHeight(row)
}

// ListHeights returns a paginated list of heights for a baby in ULID descending order.
func ListHeights(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.Height], error) {
	return ListHeightsWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListHeightsWithTZ returns a paginated list of heights with timezone-aware date filtering.
func ListHeightsWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.Height], error) {
	return listMetricWithTZ(db, "heights", heightColumns, babyID, from, to, cursor, limit, loc, scanHeight, func(h *model.Height) string { return h.ID })
}

// UpdateHeight updates a height entry.
func UpdateHeight(db *sql.DB, babyID, heightID, updatedBy, timestamp string, heightCm float64, measurementSource, notes *string) (*model.Height, error) {
	res, err := db.Exec(
		`UPDATE heights SET
			updated_by = ?, timestamp = ?, height_cm = ?,
			measurement_source = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, heightCm,
		measurementSource, notes,
		heightID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update height: %w", err)
	}

	if err := checkRowsAffected(res, "update height"); err != nil {
		return nil, err
	}

	return GetHeightByID(db, babyID, heightID)
}

// DeleteHeight hard-deletes a height entry.
func DeleteHeight(db *sql.DB, babyID, heightID string) error {
	return deleteByID(db, "heights", babyID, heightID)
}
