package backup_test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/ablankz/LittleLiver/backend/internal/backup"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestBackup_ProducesValidSQLiteFile(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	// Insert some test data
	user := testutil.CreateTestUser(t, db)
	_ = testutil.CreateTestBaby(t, db, user.ID)

	memStore := storage.NewMemoryStore()
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	key, err := backup.Run(context.Background(), db, memStore, now)
	if err != nil {
		t.Fatalf("backup.Run failed: %v", err)
	}

	// Verify the key has the expected format
	expected := "backups/littleliver-2026-01-15.db"
	if key != expected {
		t.Errorf("expected key %q, got %q", expected, key)
	}

	// Verify the backup file was stored in R2
	data, err := memStore.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("backup file not found in store: %v", err)
	}

	// A valid SQLite file starts with "SQLite format 3\000"
	if len(data) < 16 {
		t.Fatalf("backup file too small: %d bytes", len(data))
	}
	header := string(data[:16])
	if !strings.HasPrefix(header, "SQLite format 3") {
		t.Errorf("backup file is not a valid SQLite file, header: %q", header)
	}
}

func TestBackup_StoredInR2WithCorrectContentType(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	memStore := storage.NewMemoryStore()
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)

	key, err := backup.Run(context.Background(), db, memStore, now)
	if err != nil {
		t.Fatalf("backup.Run failed: %v", err)
	}

	// Check the content type
	_, contentType, ok := memStore.GetWithMeta(key)
	if !ok {
		t.Fatal("backup file not found in store")
	}
	if contentType != "application/x-sqlite3" {
		t.Errorf("expected content type application/x-sqlite3, got %q", contentType)
	}
}

func TestBackup_RestoredDataMatchesOriginal(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	// Insert test data
	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Insert a feeding entry
	_, err := db.Exec(
		"INSERT INTO feedings (id, baby_id, logged_by, timestamp, feed_type, volume_ml) VALUES (?, ?, ?, ?, ?, ?)",
		"feed-1", baby.ID, user.ID, "2026-01-15 08:00:00", "formula", 120.0,
	)
	if err != nil {
		t.Fatalf("insert feeding: %v", err)
	}

	memStore := storage.NewMemoryStore()
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	key, err := backup.Run(context.Background(), db, memStore, now)
	if err != nil {
		t.Fatalf("backup.Run failed: %v", err)
	}

	// Get the backup data
	backupData, err := memStore.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("get backup data: %v", err)
	}

	// Restore the backup to a temp file and open it
	restoredDB, cleanup := openBackupDB(t, backupData)
	defer cleanup()

	// Verify the data matches
	var feedID, feedBabyID, feedType string
	var volumeML float64
	err = restoredDB.QueryRow("SELECT id, baby_id, feed_type, volume_ml FROM feedings WHERE id = 'feed-1'").
		Scan(&feedID, &feedBabyID, &feedType, &volumeML)
	if err != nil {
		t.Fatalf("query restored feeding: %v", err)
	}
	if feedID != "feed-1" {
		t.Errorf("expected feed ID 'feed-1', got %q", feedID)
	}
	if feedBabyID != baby.ID {
		t.Errorf("expected baby ID %q, got %q", baby.ID, feedBabyID)
	}
	if feedType != "formula" {
		t.Errorf("expected feed type 'formula', got %q", feedType)
	}
	if volumeML != 120.0 {
		t.Errorf("expected volume 120.0, got %f", volumeML)
	}

	// Verify user data matches
	var userName, userEmail string
	err = restoredDB.QueryRow("SELECT name, email FROM users WHERE id = ?", user.ID).
		Scan(&userName, &userEmail)
	if err != nil {
		t.Fatalf("query restored user: %v", err)
	}
	if userName != user.Name {
		t.Errorf("expected user name %q, got %q", user.Name, userName)
	}

	// Verify baby data matches
	var babyName string
	err = restoredDB.QueryRow("SELECT name FROM babies WHERE id = ?", baby.ID).
		Scan(&babyName)
	if err != nil {
		t.Fatalf("query restored baby: %v", err)
	}
	if babyName != baby.Name {
		t.Errorf("expected baby name %q, got %q", baby.Name, babyName)
	}
}

func TestBackup_KeyFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "standard date",
			time:     time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			expected: "backups/littleliver-2026-01-15.db",
		},
		{
			name:     "end of year",
			time:     time.Date(2025, 12, 31, 23, 59, 0, 0, time.UTC),
			expected: "backups/littleliver-2025-12-31.db",
		},
		{
			name:     "beginning of year",
			time:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: "backups/littleliver-2026-01-01.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := testutil.SetupTestDB(t)
			defer db.Close()

			memStore := storage.NewMemoryStore()
			key, err := backup.Run(context.Background(), db, memStore, tt.time)
			if err != nil {
				t.Fatalf("backup.Run failed: %v", err)
			}
			if key != tt.expected {
				t.Errorf("expected key %q, got %q", tt.expected, key)
			}
		})
	}
}

func TestBackup_UploadError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	failStore := &failingPutStore{}
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	_, err := backup.Run(context.Background(), db, failStore, now)
	if err == nil {
		t.Fatal("expected error when upload fails")
	}
}

// openBackupDB writes backup data to a temp file and opens it as a SQLite DB.
func openBackupDB(t *testing.T, data []byte) (*sql.DB, func()) {
	t.Helper()
	tmpFile := t.TempDir() + "/restored.db"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	db, err := sql.Open("sqlite", tmpFile)
	if err != nil {
		t.Fatalf("open restored db: %v", err)
	}
	return db, func() { db.Close() }
}

func TestRunner_StartStop(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	memStore := storage.NewMemoryStore()

	runner := backup.NewRunner(db, memStore, 50*time.Millisecond)
	runner.Start()

	// Let it run at least one tick
	time.Sleep(120 * time.Millisecond)

	// Stop should return without hanging
	done := make(chan struct{})
	go func() {
		runner.Stop()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Runner.Stop() did not return within 2 seconds")
	}

	// Verify at least one backup was created
	keys := memStore.Keys()
	found := false
	for _, k := range keys {
		if strings.HasPrefix(k, "backups/littleliver-") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one backup to be created by the runner")
	}
}

func TestRunner_StartStopWithErrors(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)

	// Close DB to force backup errors
	db.Close()

	memStore := storage.NewMemoryStore()
	runner := backup.NewRunner(db, memStore, 50*time.Millisecond)
	runner.Start()
	time.Sleep(80 * time.Millisecond)
	runner.Stop()
	// Just ensure it doesn't panic on error
}

func TestRunner_DefaultInterval(t *testing.T) {
	t.Parallel()
	if backup.DefaultInterval != 24*time.Hour {
		t.Errorf("expected DefaultInterval=24h, got %v", backup.DefaultInterval)
	}
}

// failingPutStore is an ObjectStore that always fails on Put.
type failingPutStore struct{}

func (f *failingPutStore) Put(_ context.Context, _ string, _ io.Reader, _ string) error {
	return fmt.Errorf("simulated upload error")
}

func (f *failingPutStore) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *failingPutStore) Delete(_ context.Context, _ string) error {
	return fmt.Errorf("not implemented")
}

func (f *failingPutStore) SignedURL(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("not implemented")
}
