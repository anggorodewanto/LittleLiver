package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

const imagingStudyColumns = `id, baby_id, logged_by, updated_by, timestamp,
	study_date, study_type, notes, photo_keys, created_at, updated_at`

func scanImagingStudy(s scanner) (*model.ImagingStudy, error) {
	var st model.ImagingStudy
	var updatedBy, notes sql.NullString
	var tsStr, createdStr, updatedStr string

	err := s.Scan(
		&st.ID, &st.BabyID, &st.LoggedBy, &updatedBy, &tsStr,
		&st.StudyDate, &st.StudyType, &notes, &st.PhotoKeys,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	st.Timestamp, st.CreatedAt, st.UpdatedAt, err = parseMetricTimes(tsStr, createdStr, updatedStr)
	if err != nil {
		return nil, err
	}

	st.UpdatedBy = nullStr(updatedBy)
	st.Notes = nullStr(notes)
	return &st, nil
}

// CreateImagingStudy inserts a new imaging study and returns it.
func CreateImagingStudy(db *sql.DB, babyID, loggedBy, timestamp, studyDate, studyType string, notes *string, photoKeys string) (*model.ImagingStudy, error) {
	id := model.NewULID()
	_, err := db.Exec(
		`INSERT INTO imaging_studies (id, baby_id, logged_by, timestamp, study_date, study_type, notes, photo_keys)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, babyID, loggedBy, timestamp, studyDate, studyType, notes, photoKeys,
	)
	if err != nil {
		return nil, fmt.Errorf("create imaging study: %w", err)
	}
	return GetImagingStudyByID(db, babyID, id)
}

// GetImagingStudyByID retrieves an imaging study scoped to a baby.
func GetImagingStudyByID(db *sql.DB, babyID, studyID string) (*model.ImagingStudy, error) {
	row := db.QueryRow(
		`SELECT `+imagingStudyColumns+` FROM imaging_studies WHERE id = ? AND baby_id = ?`,
		studyID, babyID,
	)
	return scanImagingStudy(row)
}

// ListImagingStudies returns paginated imaging studies for a baby in ULID DESC order.
func ListImagingStudies(db *sql.DB, babyID string, from, to, cursor *string, limit int) (*model.MetricPage[model.ImagingStudy], error) {
	return ListImagingStudiesWithTZ(db, babyID, from, to, cursor, limit, time.UTC)
}

// ListImagingStudiesWithTZ returns paginated imaging studies with timezone-aware date filtering.
func ListImagingStudiesWithTZ(db *sql.DB, babyID string, from, to, cursor *string, limit int, loc *time.Location) (*model.MetricPage[model.ImagingStudy], error) {
	return listMetricWithTZ(db, "imaging_studies", imagingStudyColumns, babyID, from, to, cursor, limit, loc, scanImagingStudy, func(s *model.ImagingStudy) string { return s.ID })
}

// UpdateImagingStudy updates an imaging study.
func UpdateImagingStudy(db *sql.DB, babyID, studyID, updatedBy, timestamp, studyDate, studyType string, notes *string, photoKeys string) (*model.ImagingStudy, error) {
	res, err := db.Exec(
		`UPDATE imaging_studies SET
			updated_by = ?, timestamp = ?, study_date = ?,
			study_type = ?, notes = ?, photo_keys = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND baby_id = ?`,
		updatedBy, timestamp, studyDate,
		studyType, notes, photoKeys,
		studyID, babyID,
	)
	if err != nil {
		return nil, fmt.Errorf("update imaging study: %w", err)
	}
	if err := checkRowsAffected(res, "update imaging study"); err != nil {
		return nil, err
	}
	return GetImagingStudyByID(db, babyID, studyID)
}

// DeleteImagingStudy hard-deletes an imaging study.
func DeleteImagingStudy(db *sql.DB, babyID, studyID string) error {
	return deleteByID(db, "imaging_studies", babyID, studyID)
}
