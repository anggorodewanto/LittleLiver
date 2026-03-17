package store

import (
	"database/sql"
	"fmt"
	"strings"
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

	w.Timestamp, err = parseTime(tsStr)
	if err != nil {
		return nil, fmt.Errorf("parse timestamp: %w", err)
	}
	w.CreatedAt, err = parseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	w.UpdatedAt, err = parseTime(updatedStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	if updatedBy.Valid {
		w.UpdatedBy = &updatedBy.String
	}
	if measurementSource.Valid {
		w.MeasurementSource = &measurementSource.String
	}
	if notes.Valid {
		w.Notes = &notes.String
	}

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
		utcTo := t.Add(24*time.Hour - time.Second).UTC().Format(model.DateTimeFormat)
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, utcTo)
	}

	if cursor != nil {
		conditions = append(conditions, "id < ?")
		args = append(args, *cursor)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM weights WHERE %s ORDER BY id DESC LIMIT ?",
		weightColumns,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit+1)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list weights: %w", err)
	}
	defer rows.Close()

	var weights []model.Weight
	for rows.Next() {
		w, err := scanWeight(rows)
		if err != nil {
			return nil, fmt.Errorf("scan weight: %w", err)
		}
		weights = append(weights, *w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	page := &model.MetricPage[model.Weight]{
		Data: make([]model.Weight, 0),
	}

	if len(weights) > limit {
		page.Data = weights[:limit]
		nextCursor := weights[limit-1].ID
		page.NextCursor = &nextCursor
	} else {
		page.Data = weights
	}

	if page.Data == nil {
		page.Data = make([]model.Weight, 0)
	}

	return page, nil
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

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update weight: rows affected: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("update weight: %w", sql.ErrNoRows)
	}

	return GetWeightByID(db, babyID, weightID)
}

// DeleteWeight hard-deletes a weight entry.
func DeleteWeight(db *sql.DB, babyID, weightID string) error {
	res, err := db.Exec(
		"DELETE FROM weights WHERE id = ? AND baby_id = ?",
		weightID, babyID,
	)
	if err != nil {
		return fmt.Errorf("delete weight: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete weight: rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("delete weight: %w", sql.ErrNoRows)
	}

	return nil
}
