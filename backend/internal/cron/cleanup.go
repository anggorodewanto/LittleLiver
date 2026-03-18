// Package cron provides periodic cleanup tasks for LittleLiver.
package cron

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/storage"
)

// CleanupInvites deletes all invite codes older than 24 hours.
// Returns the number of rows deleted.
func CleanupInvites(db *sql.DB, now time.Time) (int64, error) {
	cutoff := now.Add(-24 * time.Hour).Format(time.DateTime)
	res, err := db.Exec("DELETE FROM invites WHERE created_at < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup invites: %w", err)
	}
	return res.RowsAffected()
}

// CleanupSessions deletes all expired sessions.
// Returns the number of rows deleted.
func CleanupSessions(db *sql.DB, now time.Time) (int64, error) {
	res, err := db.Exec("DELETE FROM sessions WHERE expires_at <= ?", now.Format(time.DateTime))
	if err != nil {
		return 0, fmt.Errorf("cleanup sessions: %w", err)
	}
	return res.RowsAffected()
}

// orphanPhoto holds the data needed to clean up an orphaned photo.
type orphanPhoto struct {
	ID           string
	R2Key        string
	ThumbnailKey sql.NullString
}

// CleanupPhotos deletes orphaned photo_uploads rows and their R2 objects.
// Orphaned means: (linked_at IS NULL AND uploaded_at < now - 24h) OR (baby_id IS NULL).
// Returns the number of rows deleted.
func CleanupPhotos(db *sql.DB, objStore storage.ObjectStore, now time.Time) (int64, error) {
	cutoff := now.Add(-24 * time.Hour).Format(time.DateTime)

	rows, err := db.Query(
		`SELECT id, r2_key, thumbnail_key FROM photo_uploads
		 WHERE (linked_at IS NULL AND uploaded_at < ?) OR baby_id IS NULL`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("cleanup photos query: %w", err)
	}
	defer rows.Close()

	var photos []orphanPhoto
	for rows.Next() {
		var p orphanPhoto
		if err := rows.Scan(&p.ID, &p.R2Key, &p.ThumbnailKey); err != nil {
			return 0, fmt.Errorf("cleanup photos scan: %w", err)
		}
		photos = append(photos, p)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("cleanup photos iterate: %w", err)
	}

	ctx := context.Background()
	var deleted int64
	for _, p := range photos {
		// Delete R2 objects first, then the DB row
		if err := objStore.Delete(ctx, p.R2Key); err != nil {
			log.Printf("cleanup photos: delete r2 key %s: %v", p.R2Key, err)
			continue
		}
		if p.ThumbnailKey.Valid {
			if err := objStore.Delete(ctx, p.ThumbnailKey.String); err != nil {
				log.Printf("cleanup photos: delete thumbnail %s: %v", p.ThumbnailKey.String, err)
			}
		}

		_, err := db.Exec("DELETE FROM photo_uploads WHERE id = ?", p.ID)
		if err != nil {
			log.Printf("cleanup photos: delete row %s: %v", p.ID, err)
			continue
		}
		deleted++
	}

	return deleted, nil
}

// RunAll executes all cleanup tasks and returns the first error encountered.
func RunAll(db *sql.DB, objStore storage.ObjectStore) error {
	now := time.Now().UTC()

	invites, err := CleanupInvites(db, now)
	if err != nil {
		return fmt.Errorf("run all: %w", err)
	}
	if invites > 0 {
		log.Printf("cron: cleaned up %d old invites", invites)
	}

	sessions, err := CleanupSessions(db, now)
	if err != nil {
		return fmt.Errorf("run all: %w", err)
	}
	if sessions > 0 {
		log.Printf("cron: cleaned up %d expired sessions", sessions)
	}

	photos, err := CleanupPhotos(db, objStore, now)
	if err != nil {
		return fmt.Errorf("run all: %w", err)
	}
	if photos > 0 {
		log.Printf("cron: cleaned up %d orphaned photos", photos)
	}

	return nil
}
