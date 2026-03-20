// Package backup provides SQLite database backup functionality.
package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/storage"
)

// keyPrefix is the R2 key prefix for backup files.
const keyPrefix = "backups/littleliver-"

// Run creates a backup of the SQLite database and uploads it to the object store.
// It uses VACUUM INTO to create a consistent snapshot, uploads it to the store
// with a timestamped key, and cleans up the temporary file.
// Returns the R2 key of the uploaded backup file.
func Run(ctx context.Context, db *sql.DB, store storage.ObjectStore, now time.Time) (string, error) {
	key := keyPrefix + now.Format("2006-01-02") + ".db"

	// Create a temp file for the backup
	tmpFile, err := os.CreateTemp("", "littleliver-backup-*.db")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Use VACUUM INTO to create a clean, consistent backup
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", tmpPath); err != nil {
		return "", fmt.Errorf("vacuum into: %w", err)
	}

	// Open the backup file for upload
	f, err := os.Open(tmpPath)
	if err != nil {
		return "", fmt.Errorf("open backup file: %w", err)
	}
	defer f.Close()

	// Upload to object store
	if err := store.Put(ctx, key, f, "application/x-sqlite3"); err != nil {
		return "", fmt.Errorf("upload backup: %w", err)
	}

	return key, nil
}
