package store

import (
	"database/sql"
	"fmt"
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

	a.Timestamp, a.CreatedAt, a.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	a.UpdatedBy = nullStr(updatedBy)
	a.GirthCm = nullFloat(girthCm)
	a.PhotoKeys = nullStr(photoKeys)
	a.Notes = nullStr(notes)

	return &a, nil
}

// CreateAbdomen inserts a new abdomen observation and returns it.
func CreateAbdomen(db *sql.DB, babyID, loggedBy, timestamp, firmness string, tenderness bool, girthCm *float64, notes *string) (*model.AbdomenObservation, error) {
	return CreateAbdomenWithPhotos(db, babyID, loggedBy, timestamp, firmness, tenderness, girthCm, nil, notes)
}

// CreateAbdomenWithPhotos inserts a new abdomen observation with optional photo keys and returns it.
func CreateAbdomenWithPhotos(db *sql.DB, babyID, loggedBy, timestamp, firmness string, tenderness bool, girthCm *float64, photoKeys, notes *string) (*model.AbdomenObservation, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO abdomen_observations (id, baby_id, logged_by, timestamp, firmness, tenderness, girth_cm, photo_keys, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, firmness, tenderness, girthCm, photoKeys, notes,
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
	return listMetricWithTZ(db, "abdomen_observations", abdomenColumns, babyID, from, to, cursor, limit, loc, scanAbdomen, func(a *model.AbdomenObservation) string { return a.ID })
}

// UpdateAbdomen updates an abdomen observation.
func UpdateAbdomen(db *sql.DB, babyID, abdomenID, updatedBy, timestamp, firmness string, tenderness bool, girthCm *float64, notes *string) (*model.AbdomenObservation, error) {
	return UpdateAbdomenWithPhotos(db, babyID, abdomenID, updatedBy, timestamp, firmness, tenderness, girthCm, nil, notes)
}

// UpdateAbdomenWithPhotos updates an abdomen observation with optional photo keys.
func UpdateAbdomenWithPhotos(db *sql.DB, babyID, abdomenID, updatedBy, timestamp, firmness string, tenderness bool, girthCm *float64, photoKeys, notes *string) (*model.AbdomenObservation, error) {
	res, err := db.Exec(
		`UPDATE abdomen_observations SET
			updated_by = ?, timestamp = ?, firmness = ?,
			tenderness = ?, girth_cm = ?, photo_keys = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, firmness,
		tenderness, girthCm, photoKeys, notes,
		abdomenID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update abdomen: %w", err)
	}

	if err := checkRowsAffected(res, "update abdomen"); err != nil {
		return nil, err
	}

	return GetAbdomenByID(db, babyID, abdomenID)
}

// DeleteAbdomen hard-deletes an abdomen observation.
func DeleteAbdomen(db *sql.DB, babyID, abdomenID string) error {
	return deleteByID(db, "abdomen_observations", babyID, abdomenID)
}
