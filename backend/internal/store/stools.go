package store

import (
	"database/sql"
	"fmt"
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
	return CreateStoolWithPhotos(db, babyID, loggedBy, timestamp, colorRating, colorLabel, consistency, volumeEstimate, nil, notes)
}

// CreateStoolWithPhotos inserts a new stool entry with optional photo keys and returns it.
func CreateStoolWithPhotos(db *sql.DB, babyID, loggedBy, timestamp string, colorRating int, colorLabel, consistency, volumeEstimate, photoKeys, notes *string) (*model.Stool, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO stools (id, baby_id, logged_by, timestamp, color_rating, color_label, consistency, volume_estimate, photo_keys, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, colorRating, colorLabel, consistency, volumeEstimate, photoKeys, notes,
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
	return listMetricWithTZ(db, "stools", stoolColumns, babyID, from, to, cursor, limit, loc, scanStool, func(s *model.Stool) string { return s.ID })
}

// UpdateStool updates a stool entry.
func UpdateStool(db *sql.DB, babyID, stoolID, updatedBy, timestamp string, colorRating int, colorLabel, consistency, volumeEstimate, notes *string) (*model.Stool, error) {
	return UpdateStoolWithPhotos(db, babyID, stoolID, updatedBy, timestamp, colorRating, colorLabel, consistency, volumeEstimate, nil, notes)
}

// UpdateStoolWithPhotos updates a stool entry with optional photo keys.
func UpdateStoolWithPhotos(db *sql.DB, babyID, stoolID, updatedBy, timestamp string, colorRating int, colorLabel, consistency, volumeEstimate, photoKeys, notes *string) (*model.Stool, error) {
	res, err := db.Exec(
		`UPDATE stools SET
			updated_by = ?, timestamp = ?, color_rating = ?,
			color_label = ?, consistency = ?, volume_estimate = ?,
			photo_keys = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, colorRating,
		colorLabel, consistency, volumeEstimate,
		photoKeys, notes,
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
