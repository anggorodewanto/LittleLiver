package store

import (
	"database/sql"
	"fmt"

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

	p.UploadedAt, err = parseTime(uploadedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse uploaded_at: %w", err)
	}

	if linkedAtStr.Valid {
		t, err := parseTime(linkedAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse linked_at: %w", err)
		}
		p.LinkedAt = &t
	}

	return &p, nil
}
