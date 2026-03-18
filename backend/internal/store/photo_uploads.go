package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// CreatePhotoUpload inserts a new photo_uploads row and returns the created record.
func CreatePhotoUpload(db *sql.DB, babyID, r2Key, thumbnailKey string) (*model.PhotoUpload, error) {
	id := model.NewULID()

	_, err := db.Exec(
		`INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key) VALUES (?, ?, ?, ?)`,
		id, babyID, r2Key, thumbnailKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create photo_upload: %w", err)
	}

	return getPhotoUploadByID(db, id)
}

// GetPhotoUploadByR2Key retrieves a photo upload by its R2 key.
func GetPhotoUploadByR2Key(db *sql.DB, r2Key string) (*model.PhotoUpload, error) {
	row := db.QueryRow(
		`SELECT id, baby_id, r2_key, thumbnail_key, uploaded_at, linked_at FROM photo_uploads WHERE r2_key = ?`,
		r2Key,
	)
	return scanPhotoUpload(row)
}

func getPhotoUploadByID(db *sql.DB, id string) (*model.PhotoUpload, error) {
	row := db.QueryRow(
		`SELECT id, baby_id, r2_key, thumbnail_key, uploaded_at, linked_at FROM photo_uploads WHERE id = ?`,
		id,
	)
	return scanPhotoUpload(row)
}

// MaxPhotosPerMetric is the maximum number of photos that can be linked to a single metric entry.
const MaxPhotosPerMetric = 4

// ValidateAndLinkPhotos validates that the given R2 keys exist in photo_uploads
// for the specified baby and sets linked_at on each. Returns an error if any key
// is invalid, belongs to a different baby, or exceeds the 4-photo limit.
func ValidateAndLinkPhotos(db *sql.DB, babyID string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	if len(keys) > MaxPhotosPerMetric {
		return fmt.Errorf("exceeds maximum of %d photos per entry", MaxPhotosPerMetric)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, key := range keys {
		var id string
		var photoBabyID sql.NullString
		var linkedAt sql.NullString
		err := tx.QueryRow(
			`SELECT id, baby_id, linked_at FROM photo_uploads WHERE r2_key = ?`, key,
		).Scan(&id, &photoBabyID, &linkedAt)
		if err != nil {
			return fmt.Errorf("photo key %q not found: %w", key, err)
		}
		if !photoBabyID.Valid || photoBabyID.String != babyID {
			return fmt.Errorf("photo key %q does not belong to baby %s", key, babyID)
		}
		if linkedAt.Valid {
			return fmt.Errorf("photo key %q is already linked to another entry", key)
		}

		_, err = tx.Exec(
			`UPDATE photo_uploads SET linked_at = CURRENT_TIMESTAMP WHERE r2_key = ?`, key,
		)
		if err != nil {
			return fmt.Errorf("link photo %q: %w", key, err)
		}
	}

	return tx.Commit()
}

// UnlinkPhotos sets linked_at = NULL for the given R2 keys using a single query.
func UnlinkPhotos(db *sql.DB, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(keys))
	placeholders = placeholders[:len(placeholders)-1] // trim trailing comma
	args := make([]any, len(keys))
	for i, k := range keys {
		args[i] = k
	}
	_, err := db.Exec(
		fmt.Sprintf(`UPDATE photo_uploads SET linked_at = NULL WHERE r2_key IN (%s)`, placeholders),
		args...,
	)
	if err != nil {
		return fmt.Errorf("unlink photos: %w", err)
	}
	return nil
}

// GetPhotoUploadsByR2Keys retrieves photo uploads for the given R2 keys using a single query.
func GetPhotoUploadsByR2Keys(db *sql.DB, keys []string) ([]model.PhotoUpload, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(keys))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(keys))
	for i, k := range keys {
		args[i] = k
	}
	rows, err := db.Query(
		fmt.Sprintf(`SELECT id, baby_id, r2_key, thumbnail_key, uploaded_at, linked_at FROM photo_uploads WHERE r2_key IN (%s)`, placeholders),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get photos by r2 keys: %w", err)
	}
	defer rows.Close()

	// Index results by r2_key for ordered output
	byKey := make(map[string]model.PhotoUpload, len(keys))
	for rows.Next() {
		p, err := scanPhotoUpload(rows)
		if err != nil {
			return nil, err
		}
		byKey[p.R2Key] = *p
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate photo rows: %w", err)
	}

	// Return in original key order
	photos := make([]model.PhotoUpload, 0, len(keys))
	for _, k := range keys {
		if p, ok := byKey[k]; ok {
			photos = append(photos, p)
		}
	}
	return photos, nil
}

func scanPhotoUpload(s scanner) (*model.PhotoUpload, error) {
	var p model.PhotoUpload
	var babyID, thumbnailKey, linkedAtStr sql.NullString
	var uploadedAtStr string

	err := s.Scan(&p.ID, &babyID, &p.R2Key, &thumbnailKey, &uploadedAtStr, &linkedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scan photo_upload: %w", err)
	}

	p.BabyID = nullStr(babyID)
	p.ThumbnailKey = nullStr(thumbnailKey)

	p.UploadedAt, err = ParseTime(uploadedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse uploaded_at: %w", err)
	}

	if linkedAtStr.Valid {
		t, err := ParseTime(linkedAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse linked_at: %w", err)
		}
		p.LinkedAt = &t
	}

	return &p, nil
}
