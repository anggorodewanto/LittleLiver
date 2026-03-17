package store

import (
	"database/sql"
	"fmt"
	"strings"
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
	var conditions []string
	var args []any

	conditions = append(conditions, "baby_id = ?")
	args = append(args, babyID)

	if from != nil {
		t, err := time.ParseInLocation(model.DateFormat, *from, loc)
		if err != nil {
			return nil, fmt.Errorf("parse from date: %w", err)
		}
		utcFrom := t.UTC().Format(model.DateTimeFormat)
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, utcFrom)
	}

	if to != nil {
		t, err := time.ParseInLocation(model.DateFormat, *to, loc)
		if err != nil {
			return nil, fmt.Errorf("parse to date: %w", err)
		}
		utcTo := t.Add(24 * time.Hour).UTC().Format(model.DateTimeFormat)
		conditions = append(conditions, "timestamp < ?")
		args = append(args, utcTo)
	}

	if cursor != nil {
		conditions = append(conditions, "id < ?")
		args = append(args, *cursor)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM temperatures WHERE %s ORDER BY id DESC LIMIT ?",
		temperatureColumns,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit+1)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list temperatures: %w", err)
	}
	defer rows.Close()

	var temps []model.Temperature
	for rows.Next() {
		t, err := scanTemperature(rows)
		if err != nil {
			return nil, fmt.Errorf("scan temperature: %w", err)
		}
		temps = append(temps, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	page := &model.MetricPage[model.Temperature]{}

	if len(temps) > limit {
		page.Data = temps[:limit]
		nextCursor := temps[limit-1].ID
		page.NextCursor = &nextCursor
	} else {
		page.Data = temps
	}

	if page.Data == nil {
		page.Data = make([]model.Temperature, 0)
	}

	return page, nil
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
