package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const labResultColumns = `id, baby_id, logged_by, updated_by, timestamp,
	test_name, value, unit, normal_range, notes, created_at, updated_at`

// scanLabResult scans a single lab result row from the given scanner.
func scanLabResult(s scanner) (*model.LabResult, error) {
	var l model.LabResult
	var updatedBy, unit, normalRange, notes sql.NullString
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&l.ID, &l.BabyID, &l.LoggedBy, &updatedBy, &tsStr,
		&l.TestName, &l.Value, &unit, &normalRange, &notes,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	l.Timestamp, l.CreatedAt, l.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	l.UpdatedBy = nullStr(updatedBy)
	l.Unit = nullStr(unit)
	l.NormalRange = nullStr(normalRange)
	l.Notes = nullStr(notes)

	return &l, nil
}

// CreateLabResult inserts a new lab result and returns it.
func CreateLabResult(db *sql.DB, babyID, loggedBy, timestamp, testName, value string, unit, normalRange, notes *string) (*model.LabResult, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO lab_results (id, baby_id, logged_by, timestamp, test_name, value, unit, normal_range, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, testName, value, unit, normalRange, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create lab result: %w", err)
	}

	return GetLabResultByID(db, babyID, id)
}

// GetLabResultByID retrieves a lab result by its ID, scoped to the given baby.
func GetLabResultByID(db *sql.DB, babyID, labID string) (*model.LabResult, error) {
	row := db.QueryRow(
		`SELECT `+labResultColumns+` FROM lab_results WHERE id = ? AND baby_id = ?`,
		labID, babyID,
	)
	return scanLabResult(row)
}

// ListLabResults returns a paginated list of lab results for a baby in ULID descending order.
func ListLabResults(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.LabResult], error) {
	return ListLabResultsWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListLabResultsWithTZ returns a paginated list of lab results with timezone-aware date filtering.
func ListLabResultsWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.LabResult], error) {
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
		"SELECT %s FROM lab_results WHERE %s ORDER BY id DESC LIMIT ?",
		labResultColumns,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit+1)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list lab results: %w", err)
	}
	defer rows.Close()

	var results []model.LabResult
	for rows.Next() {
		l, err := scanLabResult(rows)
		if err != nil {
			return nil, fmt.Errorf("scan lab result: %w", err)
		}
		results = append(results, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	page := &model.MetricPage[model.LabResult]{}

	if len(results) > limit {
		page.Data = results[:limit]
		nextCursor := results[limit-1].ID
		page.NextCursor = &nextCursor
	} else {
		page.Data = results
	}

	if page.Data == nil {
		page.Data = make([]model.LabResult, 0)
	}

	return page, nil
}

// UpdateLabResult updates a lab result.
func UpdateLabResult(db *sql.DB, babyID, labID, updatedBy, timestamp, testName, value string, unit, normalRange, notes *string) (*model.LabResult, error) {
	res, err := db.Exec(
		`UPDATE lab_results SET
			updated_by = ?, timestamp = ?, test_name = ?,
			value = ?, unit = ?, normal_range = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, testName,
		value, unit, normalRange, notes,
		labID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update lab result: %w", err)
	}

	if err := checkRowsAffected(res, "update lab result"); err != nil {
		return nil, err
	}

	return GetLabResultByID(db, babyID, labID)
}

// DeleteLabResult hard-deletes a lab result.
func DeleteLabResult(db *sql.DB, babyID, labID string) error {
	return deleteByID(db, "lab_results", babyID, labID)
}
