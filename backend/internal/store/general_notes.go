package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const generalNoteColumns = `id, baby_id, logged_by, updated_by, timestamp,
	content, photo_keys, category, created_at, updated_at`

// scanGeneralNote scans a single general note row from the given scanner.
func scanGeneralNote(s scanner) (*model.GeneralNote, error) {
	var n model.GeneralNote
	var updatedBy, photoKeys, category sql.NullString
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&n.ID, &n.BabyID, &n.LoggedBy, &updatedBy, &tsStr,
		&n.Content, &photoKeys, &category,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	n.Timestamp, n.CreatedAt, n.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	n.UpdatedBy = nullStr(updatedBy)
	n.PhotoKeys = nullStr(photoKeys)
	n.Category = nullStr(category)

	return &n, nil
}

// CreateGeneralNote inserts a new general note and returns it.
func CreateGeneralNote(db *sql.DB, babyID, loggedBy, timestamp, content string, photoKeys, category *string) (*model.GeneralNote, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO general_notes (id, baby_id, logged_by, timestamp, content, photo_keys, category)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, content, photoKeys, category,
	)
	if err != nil {
		return nil, fmt.Errorf("create general note: %w", err)
	}

	return GetGeneralNoteByID(db, babyID, id)
}

// GetGeneralNoteByID retrieves a general note by its ID, scoped to the given baby.
func GetGeneralNoteByID(db *sql.DB, babyID, noteID string) (*model.GeneralNote, error) {
	row := db.QueryRow(
		`SELECT `+generalNoteColumns+` FROM general_notes WHERE id = ? AND baby_id = ?`,
		noteID, babyID,
	)
	return scanGeneralNote(row)
}

// ListGeneralNotes returns a paginated list of general notes for a baby in ULID descending order.
func ListGeneralNotes(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.GeneralNote], error) {
	return ListGeneralNotesWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListGeneralNotesWithTZ returns a paginated list of general notes with timezone-aware date filtering.
func ListGeneralNotesWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.GeneralNote], error) {
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
		"SELECT %s FROM general_notes WHERE %s ORDER BY id DESC LIMIT ?",
		generalNoteColumns,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit+1)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list general notes: %w", err)
	}
	defer rows.Close()

	var results []model.GeneralNote
	for rows.Next() {
		n, err := scanGeneralNote(rows)
		if err != nil {
			return nil, fmt.Errorf("scan general note: %w", err)
		}
		results = append(results, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	page := &model.MetricPage[model.GeneralNote]{}

	if len(results) > limit {
		page.Data = results[:limit]
		nextCursor := results[limit-1].ID
		page.NextCursor = &nextCursor
	} else {
		page.Data = results
	}

	if page.Data == nil {
		page.Data = make([]model.GeneralNote, 0)
	}

	return page, nil
}

// UpdateGeneralNote updates a general note.
func UpdateGeneralNote(db *sql.DB, babyID, noteID, updatedBy, timestamp, content string, photoKeys, category *string) (*model.GeneralNote, error) {
	res, err := db.Exec(
		`UPDATE general_notes SET
			updated_by = ?, timestamp = ?, content = ?,
			photo_keys = ?, category = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, content,
		photoKeys, category,
		noteID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update general note: %w", err)
	}

	if err := checkRowsAffected(res, "update general note"); err != nil {
		return nil, err
	}

	return GetGeneralNoteByID(db, babyID, noteID)
}

// DeleteGeneralNote hard-deletes a general note.
func DeleteGeneralNote(db *sql.DB, babyID, noteID string) error {
	return deleteByID(db, "general_notes", babyID, noteID)
}
