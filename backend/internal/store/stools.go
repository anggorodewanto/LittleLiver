package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const stoolColumns = `id, baby_id, logged_by, updated_by, timestamp,
	color_rating, color_label, consistency, volume_estimate, photo_keys,
	notes, created_at, updated_at`

// scanStool scans a single stool row from the given scanner.
func scanStool(s scanner) (*model.Stool, error) {
	var st model.Stool
	var updatedBy, colorLabel, consistency, volumeEstimate, photoKeys, notes sql.NullString
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&st.ID, &st.BabyID, &st.LoggedBy, &updatedBy, &tsStr,
		&st.ColorRating, &colorLabel, &consistency, &volumeEstimate, &photoKeys,
		&notes, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	st.Timestamp, st.CreatedAt, st.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	st.UpdatedBy = nullStr(updatedBy)
	st.ColorLabel = nullStr(colorLabel)
	st.Consistency = nullStr(consistency)
	st.VolumeEstimate = nullStr(volumeEstimate)
	st.PhotoKeys = nullStr(photoKeys)
	st.Notes = nullStr(notes)

	return &st, nil
}

// CreateStool inserts a new stool entry and returns it.
func CreateStool(db *sql.DB, babyID, loggedBy, timestamp string, colorRating int, colorLabel, consistency, volumeEstimate, notes *string) (*model.Stool, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO stools (id, baby_id, logged_by, timestamp, color_rating, color_label, consistency, volume_estimate, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, colorRating, colorLabel, consistency, volumeEstimate, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create stool: %w", err)
	}

	return GetStoolByID(db, babyID, id)
}

// GetStoolByID retrieves a stool by its ID, scoped to the given baby.
func GetStoolByID(db *sql.DB, babyID, stoolID string) (*model.Stool, error) {
	row := db.QueryRow(
		`SELECT `+stoolColumns+` FROM stools WHERE id = ? AND baby_id = ?`,
		stoolID, babyID,
	)
	return scanStool(row)
}

// ListStools returns a paginated list of stools for a baby in ULID descending order.
func ListStools(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.Stool], error) {
	return ListStoolsWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListStoolsWithTZ returns a paginated list of stools with timezone-aware date filtering.
func ListStoolsWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.Stool], error) {
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
		"SELECT %s FROM stools WHERE %s ORDER BY id DESC LIMIT ?",
		stoolColumns,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit+1)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list stools: %w", err)
	}
	defer rows.Close()

	var stools []model.Stool
	for rows.Next() {
		s, err := scanStool(rows)
		if err != nil {
			return nil, fmt.Errorf("scan stool: %w", err)
		}
		stools = append(stools, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	page := &model.MetricPage[model.Stool]{}

	if len(stools) > limit {
		page.Data = stools[:limit]
		nextCursor := stools[limit-1].ID
		page.NextCursor = &nextCursor
	} else {
		page.Data = stools
	}

	if page.Data == nil {
		page.Data = make([]model.Stool, 0)
	}

	return page, nil
}

// UpdateStool updates a stool entry.
func UpdateStool(db *sql.DB, babyID, stoolID, updatedBy, timestamp string, colorRating int, colorLabel, consistency, volumeEstimate, notes *string) (*model.Stool, error) {
	res, err := db.Exec(
		`UPDATE stools SET
			updated_by = ?, timestamp = ?, color_rating = ?,
			color_label = ?, consistency = ?, volume_estimate = ?,
			notes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, colorRating,
		colorLabel, consistency, volumeEstimate,
		notes,
		stoolID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update stool: %w", err)
	}

	if err := checkRowsAffected(res, "update stool"); err != nil {
		return nil, err
	}

	return GetStoolByID(db, babyID, stoolID)
}

// DeleteStool hard-deletes a stool entry.
func DeleteStool(db *sql.DB, babyID, stoolID string) error {
	return deleteByID(db, "stools", babyID, stoolID)
}
