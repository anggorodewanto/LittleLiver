package store

import (
	"database/sql"
	"fmt"
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
	return listMetricWithTZ(db, "lab_results", labResultColumns, babyID, from, to, cursor, limit, loc, scanLabResult, func(l *model.LabResult) string { return l.ID })
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

// LabTestSuggestion represents a distinct test name with its most recently used unit and normal range.
type LabTestSuggestion struct {
	TestName    string  `json:"test_name"`
	Unit        *string `json:"unit,omitempty"`
	NormalRange *string `json:"normal_range,omitempty"`
}

// ListDistinctLabTests returns distinct test names for a baby with the most recently used unit and normal range.
func ListDistinctLabTests(db *sql.DB, babyID string) ([]LabTestSuggestion, error) {
	rows, err := db.Query(`
		SELECT test_name, unit, normal_range
		FROM lab_results
		WHERE baby_id = ?
		  AND id IN (
		    SELECT id FROM (
		      SELECT id, ROW_NUMBER() OVER (PARTITION BY test_name ORDER BY timestamp DESC) AS rn
		      FROM lab_results
		      WHERE baby_id = ?
		    ) WHERE rn = 1
		  )
		ORDER BY test_name`, babyID, babyID)
	if err != nil {
		return nil, fmt.Errorf("list distinct lab tests: %w", err)
	}
	defer rows.Close()

	var suggestions []LabTestSuggestion
	for rows.Next() {
		var s LabTestSuggestion
		var unit, normalRange sql.NullString
		if err := rows.Scan(&s.TestName, &unit, &normalRange); err != nil {
			return nil, fmt.Errorf("scan lab test suggestion: %w", err)
		}
		s.Unit = nullStr(unit)
		s.NormalRange = nullStr(normalRange)
		suggestions = append(suggestions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lab test suggestions: %w", err)
	}

	return suggestions, nil
}

// DeleteLabResult hard-deletes a lab result.
func DeleteLabResult(db *sql.DB, babyID, labID string) error {
	return deleteByID(db, "lab_results", babyID, labID)
}
