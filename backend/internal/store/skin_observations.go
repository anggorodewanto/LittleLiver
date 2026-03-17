package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const skinObservationColumns = `id, baby_id, logged_by, updated_by, timestamp,
	jaundice_level, scleral_icterus, rashes, bruising, photo_keys, notes, created_at, updated_at`

// scanSkinObservation scans a single skin observation row from the given scanner.
func scanSkinObservation(s scanner) (*model.SkinObservation, error) {
	var o model.SkinObservation
	var updatedBy, jaundiceLevel, rashes, bruising, photoKeys, notes sql.NullString
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&o.ID, &o.BabyID, &o.LoggedBy, &updatedBy, &tsStr,
		&jaundiceLevel, &o.ScleralIcterus, &rashes, &bruising, &photoKeys, &notes,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	o.Timestamp, o.CreatedAt, o.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	o.UpdatedBy = nullStr(updatedBy)
	o.JaundiceLevel = nullStr(jaundiceLevel)
	o.Rashes = nullStr(rashes)
	o.Bruising = nullStr(bruising)
	o.PhotoKeys = nullStr(photoKeys)
	o.Notes = nullStr(notes)

	return &o, nil
}

// CreateSkinObservation inserts a new skin observation and returns it.
func CreateSkinObservation(db *sql.DB, babyID, loggedBy, timestamp string, jaundiceLevel *string, scleralIcterus bool, rashes, bruising, notes *string) (*model.SkinObservation, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO skin_observations (id, baby_id, logged_by, timestamp, jaundice_level, scleral_icterus, rashes, bruising, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, jaundiceLevel, scleralIcterus, rashes, bruising, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("create skin observation: %w", err)
	}

	return GetSkinObservationByID(db, babyID, id)
}

// GetSkinObservationByID retrieves a skin observation by its ID, scoped to the given baby.
func GetSkinObservationByID(db *sql.DB, babyID, skinID string) (*model.SkinObservation, error) {
	row := db.QueryRow(
		`SELECT `+skinObservationColumns+` FROM skin_observations WHERE id = ? AND baby_id = ?`,
		skinID, babyID,
	)
	return scanSkinObservation(row)
}

// ListSkinObservations returns a paginated list of skin observations for a baby in ULID descending order.
func ListSkinObservations(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.SkinObservation], error) {
	return ListSkinObservationsWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListSkinObservationsWithTZ returns a paginated list of skin observations with timezone-aware date filtering.
func ListSkinObservationsWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.SkinObservation], error) {
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
		"SELECT %s FROM skin_observations WHERE %s ORDER BY id DESC LIMIT ?",
		skinObservationColumns,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit+1)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list skin observations: %w", err)
	}
	defer rows.Close()

	var observations []model.SkinObservation
	for rows.Next() {
		o, err := scanSkinObservation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan skin observation: %w", err)
		}
		observations = append(observations, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	page := &model.MetricPage[model.SkinObservation]{}

	if len(observations) > limit {
		page.Data = observations[:limit]
		nextCursor := observations[limit-1].ID
		page.NextCursor = &nextCursor
	} else {
		page.Data = observations
	}

	if page.Data == nil {
		page.Data = make([]model.SkinObservation, 0)
	}

	return page, nil
}

// UpdateSkinObservation updates a skin observation.
func UpdateSkinObservation(db *sql.DB, babyID, skinID, updatedBy, timestamp string, jaundiceLevel *string, scleralIcterus bool, rashes, bruising, notes *string) (*model.SkinObservation, error) {
	res, err := db.Exec(
		`UPDATE skin_observations SET
			updated_by = ?, timestamp = ?, jaundice_level = ?,
			scleral_icterus = ?, rashes = ?, bruising = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, jaundiceLevel,
		scleralIcterus, rashes, bruising, notes,
		skinID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update skin observation: %w", err)
	}

	if err := checkRowsAffected(res, "update skin observation"); err != nil {
		return nil, err
	}

	return GetSkinObservationByID(db, babyID, skinID)
}

// DeleteSkinObservation hard-deletes a skin observation.
func DeleteSkinObservation(db *sql.DB, babyID, skinID string) error {
	return deleteByID(db, "skin_observations", babyID, skinID)
}
