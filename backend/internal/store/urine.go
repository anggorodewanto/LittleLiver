package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const urineColumns = `id, baby_id, logged_by, updated_by, timestamp,
	color, notes, created_at, updated_at`

// scanUrine scans a single urine row from the given scanner.
func scanUrine(s scanner) (*model.Urine, error) {
	var u model.Urine
	var updatedBy, color, notes sql.NullString
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&u.ID, &u.BabyID, &u.LoggedBy, &updatedBy, &tsStr,
		&color, &notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	u.Timestamp, err = parseTime(tsStr)
	if err != nil {
		return nil, fmt.Errorf("parse timestamp: %w", err)
	}
	u.CreatedAt, err = parseTime(createdStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	u.UpdatedAt, err = parseTime(updatedStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	if updatedBy.Valid {
		u.UpdatedBy = &updatedBy.String
	}
	if color.Valid {
		u.Color = &color.String
	}
	if notes.Valid {
		u.Notes = &notes.String
	}

	return &u, nil
}

// CreateUrine inserts a new urine entry and returns it.
func CreateUrine(db *sql.DB, babyID, loggedBy, timestamp string, color, notes *string) (*model.Urine, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO urine (id, baby_id, logged_by, timestamp, color, notes)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, color, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create urine: %w", err)
	}

	return GetUrineByID(db, babyID, id)
}

// GetUrineByID retrieves a urine entry by its ID, scoped to the given baby.
func GetUrineByID(db *sql.DB, babyID, urineID string) (*model.Urine, error) {
	row := db.QueryRow(
		`SELECT `+urineColumns+` FROM urine WHERE id = ? AND baby_id = ?`,
		urineID, babyID,
	)
	return scanUrine(row)
}

// ListUrine returns a paginated list of urine entries for a baby in ULID descending order.
func ListUrine(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.Urine], error) {
	return ListUrineWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListUrineWithTZ returns a paginated list of urine entries with timezone-aware date filtering.
func ListUrineWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.Urine], error) {
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
		"SELECT %s FROM urine WHERE %s ORDER BY id DESC LIMIT ?",
		urineColumns,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit+1)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list urine: %w", err)
	}
	defer rows.Close()

	var entries []model.Urine
	for rows.Next() {
		u, err := scanUrine(rows)
		if err != nil {
			return nil, fmt.Errorf("scan urine: %w", err)
		}
		entries = append(entries, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	page := &model.MetricPage[model.Urine]{
		Data: make([]model.Urine, 0),
	}

	if len(entries) > limit {
		page.Data = entries[:limit]
		nextCursor := entries[limit-1].ID
		page.NextCursor = &nextCursor
	} else {
		page.Data = entries
	}

	if page.Data == nil {
		page.Data = make([]model.Urine, 0)
	}

	return page, nil
}

// UpdateUrine updates a urine entry.
func UpdateUrine(db *sql.DB, babyID, urineID, updatedBy, timestamp string, color, notes *string) (*model.Urine, error) {
	res, err := db.Exec(
		`UPDATE urine SET
			updated_by = ?, timestamp = ?,
			color = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp,
		color, notes,
		urineID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update urine: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update urine: rows affected: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("update urine: %w", sql.ErrNoRows)
	}

	return GetUrineByID(db, babyID, urineID)
}

// DeleteUrine hard-deletes a urine entry.
func DeleteUrine(db *sql.DB, babyID, urineID string) error {
	res, err := db.Exec(
		"DELETE FROM urine WHERE id = ? AND baby_id = ?",
		urineID, babyID,
	)
	if err != nil {
		return fmt.Errorf("delete urine: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete urine: rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("delete urine: %w", sql.ErrNoRows)
	}

	return nil
}
