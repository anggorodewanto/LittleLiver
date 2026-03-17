package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const abdomenColumns = `id, baby_id, logged_by, updated_by, timestamp,
	firmness, tenderness, girth_cm, photo_keys, notes, created_at, updated_at`

// scanAbdomen scans a single abdomen observation row from the given scanner.
func scanAbdomen(s scanner) (*model.AbdomenObservation, error) {
	var a model.AbdomenObservation
	var updatedBy, photoKeys, notes sql.NullString
	var girthCm sql.NullFloat64
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&a.ID, &a.BabyID, &a.LoggedBy, &updatedBy, &tsStr,
		&a.Firmness, &a.Tenderness, &girthCm, &photoKeys, &notes,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	a.Timestamp, err = parseTime(tsStr)
	if err != nil {
		return nil, fmt.Errorf("parse timestamp: %w", err)
	}
	a.CreatedAt, err = parseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	a.UpdatedAt, err = parseTime(updatedStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	if updatedBy.Valid {
		a.UpdatedBy = &updatedBy.String
	}
	if girthCm.Valid {
		a.GirthCm = &girthCm.Float64
	}
	if photoKeys.Valid {
		a.PhotoKeys = &photoKeys.String
	}
	if notes.Valid {
		a.Notes = &notes.String
	}

	return &a, nil
}

// CreateAbdomen inserts a new abdomen observation and returns it.
func CreateAbdomen(db *sql.DB, babyID, loggedBy, timestamp, firmness string, tenderness bool, girthCm *float64, notes *string) (*model.AbdomenObservation, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO abdomen_observations (id, baby_id, logged_by, timestamp, firmness, tenderness, girth_cm, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, firmness, tenderness, girthCm, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create abdomen: %w", err)
	}

	return GetAbdomenByID(db, babyID, id)
}

// GetAbdomenByID retrieves an abdomen observation by its ID, scoped to the given baby.
func GetAbdomenByID(db *sql.DB, babyID, abdomenID string) (*model.AbdomenObservation, error) {
	row := db.QueryRow(
		`SELECT `+abdomenColumns+` FROM abdomen_observations WHERE id = ? AND baby_id = ?`,
		abdomenID, babyID,
	)
	return scanAbdomen(row)
}

// ListAbdomen returns a paginated list of abdomen observations for a baby in ULID descending order.
func ListAbdomen(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.AbdomenObservation], error) {
	return ListAbdomenWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListAbdomenWithTZ returns a paginated list of abdomen observations with timezone-aware date filtering.
func ListAbdomenWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.AbdomenObservation], error) {
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
		"SELECT %s FROM abdomen_observations WHERE %s ORDER BY id DESC LIMIT ?",
		abdomenColumns,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit+1)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list abdomen: %w", err)
	}
	defer rows.Close()

	var observations []model.AbdomenObservation
	for rows.Next() {
		a, err := scanAbdomen(rows)
		if err != nil {
			return nil, fmt.Errorf("scan abdomen: %w", err)
		}
		observations = append(observations, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	page := &model.MetricPage[model.AbdomenObservation]{
		Data: make([]model.AbdomenObservation, 0),
	}

	if len(observations) > limit {
		page.Data = observations[:limit]
		nextCursor := observations[limit-1].ID
		page.NextCursor = &nextCursor
	} else {
		page.Data = observations
	}

	if page.Data == nil {
		page.Data = make([]model.AbdomenObservation, 0)
	}

	return page, nil
}

// UpdateAbdomen updates an abdomen observation.
func UpdateAbdomen(db *sql.DB, babyID, abdomenID, updatedBy, timestamp, firmness string, tenderness bool, girthCm *float64, notes *string) (*model.AbdomenObservation, error) {
	res, err := db.Exec(
		`UPDATE abdomen_observations SET
			updated_by = ?, timestamp = ?, firmness = ?,
			tenderness = ?, girth_cm = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, firmness,
		tenderness, girthCm, notes,
		abdomenID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update abdomen: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update abdomen: rows affected: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("update abdomen: %w", sql.ErrNoRows)
	}

	return GetAbdomenByID(db, babyID, abdomenID)
}

// DeleteAbdomen hard-deletes an abdomen observation.
func DeleteAbdomen(db *sql.DB, babyID, abdomenID string) error {
	res, err := db.Exec(
		"DELETE FROM abdomen_observations WHERE id = ? AND baby_id = ?",
		abdomenID, babyID,
	)
	if err != nil {
		return fmt.Errorf("delete abdomen: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete abdomen: rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("delete abdomen: %w", sql.ErrNoRows)
	}

	return nil
}
