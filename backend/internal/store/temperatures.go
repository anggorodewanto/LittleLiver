package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const temperatureColumns = `id, baby_id, logged_by, updated_by, timestamp,
	value, method, notes, created_at, updated_at`

// scanTemperature scans a single temperature row from the given scanner.
func scanTemperature(s scanner) (*model.Temperature, error) {
	var t model.Temperature
	var updatedBy, notes sql.NullString
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&t.ID, &t.BabyID, &t.LoggedBy, &updatedBy, &tsStr,
		&t.Value, &t.Method, &notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	t.Timestamp, t.CreatedAt, t.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	t.UpdatedBy = nullStr(updatedBy)
	t.Notes = nullStr(notes)

	return &t, nil
}

// CreateTemperature inserts a new temperature entry and returns it.
func CreateTemperature(db *sql.DB, babyID, loggedBy, timestamp string, value float64, method string, notes *string) (*model.Temperature, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO temperatures (id, baby_id, logged_by, timestamp, value, method, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, value, method, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create temperature: %w", err)
	}

	return GetTemperatureByID(db, babyID, id)
}

// GetTemperatureByID retrieves a temperature by its ID, scoped to the given baby.
func GetTemperatureByID(db *sql.DB, babyID, tempID string) (*model.Temperature, error) {
	row := db.QueryRow(
		`SELECT `+temperatureColumns+` FROM temperatures WHERE id = ? AND baby_id = ?`,
		tempID, babyID,
	)
	return scanTemperature(row)
}

// ListTemperatures returns a paginated list of temperatures for a baby in ULID descending order.
func ListTemperatures(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.Temperature], error) {
	return ListTemperaturesWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListTemperaturesWithTZ returns a paginated list of temperatures with timezone-aware date filtering.
func ListTemperaturesWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.Temperature], error) {
	return listMetricWithTZ(db, "temperatures", temperatureColumns, babyID, from, to, cursor, limit, loc, scanTemperature, func(t *model.Temperature) string { return t.ID })
}

// UpdateTemperature updates a temperature entry.
func UpdateTemperature(db *sql.DB, babyID, tempID, updatedBy, timestamp string, value float64, method string, notes *string) (*model.Temperature, error) {
	res, err := db.Exec(
		`UPDATE temperatures SET
			updated_by = ?, timestamp = ?, value = ?,
			method = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, value,
		method, notes,
		tempID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update temperature: %w", err)
	}

	if err := checkRowsAffected(res, "update temperature"); err != nil {
		return nil, err
	}

	return GetTemperatureByID(db, babyID, tempID)
}

// DeleteTemperature hard-deletes a temperature entry.
func DeleteTemperature(db *sql.DB, babyID, tempID string) error {
	return deleteByID(db, "temperatures", babyID, tempID)
}
