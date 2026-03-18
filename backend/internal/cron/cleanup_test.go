package cron_test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/cron"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// --- Invite cleanup tests ---

func TestCleanupInvites_DeletesOldInvites(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Insert an invite that is 25 hours old
	_, err := db.Exec(
		"INSERT INTO invites (code, baby_id, created_by, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		"111111", baby.ID, user.ID,
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
		time.Now().Add(-25*time.Hour).UTC().Format(time.DateTime),
	)
	if err != nil {
		t.Fatalf("insert old invite: %v", err)
	}

	now := time.Now().UTC()
	deleted, err := cron.CleanupInvites(db, now)
	if err != nil {
		t.Fatalf("CleanupInvites failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM invites WHERE code = '111111'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected old invite to be deleted, got count=%d", count)
	}
}

func TestCleanupInvites_KeepsRecentInvites(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Create a fresh invite (less than 24h old)
	inv, err := store.CreateInvite(db, baby.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	now := time.Now().UTC()
	deleted, err := cron.CleanupInvites(db, now)
	if err != nil {
		t.Fatalf("CleanupInvites failed: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM invites WHERE code = ?", inv.Code).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected recent invite to be kept, got count=%d", count)
	}
}

// --- Session cleanup tests ---

func TestCleanupSessions_DeletesExpiredSessions(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	// Insert an expired session
	_, err := db.Exec(
		"INSERT INTO sessions (id, user_id, token, expires_at) VALUES (?, ?, ?, ?)",
		"sess-expired", user.ID, "tok-expired",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
	)
	if err != nil {
		t.Fatalf("insert expired session: %v", err)
	}

	now := time.Now().UTC()
	deleted, err := cron.CleanupSessions(db, now)
	if err != nil {
		t.Fatalf("CleanupSessions failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = 'sess-expired'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected expired session to be deleted, got count=%d", count)
	}
}

func TestCleanupSessions_KeepsActiveSessions(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	// Create a valid session
	sess, err := store.CreateSession(db, user.ID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	now := time.Now().UTC()
	deleted, err := cron.CleanupSessions(db, now)
	if err != nil {
		t.Fatalf("CleanupSessions failed: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", sess.ID).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected active session to be kept, got count=%d", count)
	}
}

// --- Photo cleanup tests ---

func TestCleanupPhotos_DeletesUnlinkedOldPhotos(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Insert an unlinked photo uploaded > 24h ago
	_, err := db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at) VALUES (?, ?, ?, ?, ?)",
		"photo-old", baby.ID, "photos/old.jpg", "photos/thumb_old.jpg",
		time.Now().Add(-25*time.Hour).UTC().Format(time.DateTime),
	)
	if err != nil {
		t.Fatalf("insert old photo: %v", err)
	}

	memStore := storage.NewMemoryStore()
	// Put objects in the memory store so we can verify deletion
	memStore.Put(context.Background(), "photos/old.jpg", strings.NewReader("data"), "image/jpeg")
	memStore.Put(context.Background(), "photos/thumb_old.jpg", strings.NewReader("data"), "image/jpeg")

	now := time.Now().UTC()
	deleted, err := cron.CleanupPhotos(db, memStore, now)
	if err != nil {
		t.Fatalf("CleanupPhotos failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	// Verify DB row deleted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM photo_uploads WHERE id = 'photo-old'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected photo row to be deleted, got count=%d", count)
	}

	// Verify R2 objects deleted
	if _, _, ok := memStore.GetWithMeta("photos/old.jpg"); ok {
		t.Error("expected R2 object photos/old.jpg to be deleted")
	}
	if _, _, ok := memStore.GetWithMeta("photos/thumb_old.jpg"); ok {
		t.Error("expected R2 thumbnail photos/thumb_old.jpg to be deleted")
	}
}

func TestCleanupPhotos_DeletesBabyNullPhotos(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	// Insert a photo where baby_id IS NULL (baby was deleted, ON DELETE SET NULL)
	_, err := db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at) VALUES (?, NULL, ?, ?, ?)",
		"photo-orphan", "photos/orphan.jpg", "photos/thumb_orphan.jpg",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
	)
	if err != nil {
		t.Fatalf("insert orphan photo: %v", err)
	}

	memStore := storage.NewMemoryStore()
	memStore.Put(context.Background(), "photos/orphan.jpg", strings.NewReader("data"), "image/jpeg")
	memStore.Put(context.Background(), "photos/thumb_orphan.jpg", strings.NewReader("data"), "image/jpeg")

	now := time.Now().UTC()
	deleted, err := cron.CleanupPhotos(db, memStore, now)
	if err != nil {
		t.Fatalf("CleanupPhotos failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	// Verify DB row deleted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM photo_uploads WHERE id = 'photo-orphan'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected orphan photo row to be deleted, got count=%d", count)
	}

	// Verify R2 objects deleted
	if _, _, ok := memStore.GetWithMeta("photos/orphan.jpg"); ok {
		t.Error("expected R2 object to be deleted")
	}
}

func TestCleanupPhotos_KeepsLinkedPhotos(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Insert a linked photo (uploaded > 24h ago but linked)
	_, err := db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at, linked_at) VALUES (?, ?, ?, ?, ?, ?)",
		"photo-linked", baby.ID, "photos/linked.jpg", "photos/thumb_linked.jpg",
		time.Now().Add(-48*time.Hour).UTC().Format(time.DateTime),
		time.Now().Add(-47*time.Hour).UTC().Format(time.DateTime),
	)
	if err != nil {
		t.Fatalf("insert linked photo: %v", err)
	}

	memStore := storage.NewMemoryStore()
	memStore.Put(context.Background(), "photos/linked.jpg", strings.NewReader("data"), "image/jpeg")

	now := time.Now().UTC()
	deleted, err := cron.CleanupPhotos(db, memStore, now)
	if err != nil {
		t.Fatalf("CleanupPhotos failed: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}

	// Verify DB row still exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM photo_uploads WHERE id = 'photo-linked'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected linked photo to be kept, got count=%d", count)
	}
}

func TestCleanupPhotos_KeepsRecentUnlinkedPhotos(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Insert an unlinked photo uploaded recently (< 24h ago)
	_, err := db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at) VALUES (?, ?, ?, ?, ?)",
		"photo-recent", baby.ID, "photos/recent.jpg", "photos/thumb_recent.jpg",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
	)
	if err != nil {
		t.Fatalf("insert recent photo: %v", err)
	}

	memStore := storage.NewMemoryStore()
	now := time.Now().UTC()
	deleted, err := cron.CleanupPhotos(db, memStore, now)
	if err != nil {
		t.Fatalf("CleanupPhotos failed: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM photo_uploads WHERE id = 'photo-recent'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected recent unlinked photo to be kept, got count=%d", count)
	}
}

func TestCleanupPhotos_HandlesNilThumbnailKey(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	// Insert a photo with NULL baby_id and NULL thumbnail_key
	_, err := db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at) VALUES (?, NULL, ?, NULL, ?)",
		"photo-no-thumb", "photos/nothumb.jpg",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
	)
	if err != nil {
		t.Fatalf("insert photo: %v", err)
	}

	memStore := storage.NewMemoryStore()
	memStore.Put(context.Background(), "photos/nothumb.jpg", strings.NewReader("data"), "image/jpeg")

	now := time.Now().UTC()
	deleted, err := cron.CleanupPhotos(db, memStore, now)
	if err != nil {
		t.Fatalf("CleanupPhotos failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	if _, _, ok := memStore.GetWithMeta("photos/nothumb.jpg"); ok {
		t.Error("expected R2 object to be deleted")
	}
}

// --- Mixed scenario test ---

func TestCleanupPhotos_MixedScenario(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// 1. Old unlinked photo (should be deleted)
	_, _ = db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at) VALUES (?, ?, ?, ?, ?)",
		"p1", baby.ID, "photos/p1.jpg", "photos/thumb_p1.jpg",
		time.Now().Add(-25*time.Hour).UTC().Format(time.DateTime),
	)
	// 2. Orphan photo with null baby_id (should be deleted)
	_, _ = db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at) VALUES (?, NULL, ?, ?, ?)",
		"p2", "photos/p2.jpg", "photos/thumb_p2.jpg",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
	)
	// 3. Linked old photo (should be kept)
	_, _ = db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at, linked_at) VALUES (?, ?, ?, ?, ?, ?)",
		"p3", baby.ID, "photos/p3.jpg", "photos/thumb_p3.jpg",
		time.Now().Add(-48*time.Hour).UTC().Format(time.DateTime),
		time.Now().Add(-47*time.Hour).UTC().Format(time.DateTime),
	)
	// 4. Recent unlinked photo (should be kept)
	_, _ = db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at) VALUES (?, ?, ?, ?, ?)",
		"p4", baby.ID, "photos/p4.jpg", "photos/thumb_p4.jpg",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
	)

	memStore := storage.NewMemoryStore()
	for _, key := range []string{"photos/p1.jpg", "photos/thumb_p1.jpg", "photos/p2.jpg", "photos/thumb_p2.jpg", "photos/p3.jpg", "photos/thumb_p3.jpg", "photos/p4.jpg", "photos/thumb_p4.jpg"} {
		memStore.Put(context.Background(), key, strings.NewReader("data"), "image/jpeg")
	}

	now := time.Now().UTC()
	deleted, err := cron.CleanupPhotos(db, memStore, now)
	if err != nil {
		t.Fatalf("CleanupPhotos failed: %v", err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", deleted)
	}

	// p1 and p2 should be gone, p3 and p4 should remain
	for _, id := range []string{"p1", "p2"} {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM photo_uploads WHERE id = ?", id).Scan(&count)
		if count != 0 {
			t.Errorf("expected photo %s to be deleted", id)
		}
	}
	for _, id := range []string{"p3", "p4"} {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM photo_uploads WHERE id = ?", id).Scan(&count)
		if count != 1 {
			t.Errorf("expected photo %s to be kept", id)
		}
	}
}

// Helper to suppress unused import warning - also tests the RunAll integration
func TestRunAll(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	memStore := storage.NewMemoryStore()

	err := cron.RunAll(db, memStore)
	if err != nil {
		t.Fatalf("RunAll failed: %v", err)
	}
}

func TestCleanupInvites_ClosedDB(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	db.Close()

	_, err := cron.CleanupInvites(db, time.Now().UTC())
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestCleanupSessions_ClosedDB(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	db.Close()

	_, err := cron.CleanupSessions(db, time.Now().UTC())
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestCleanupPhotos_ClosedDB(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	db.Close()

	memStore := storage.NewMemoryStore()
	_, err := cron.CleanupPhotos(db, memStore, time.Now().UTC())
	if err == nil {
		t.Fatal("expected error for closed DB")
	}
}

func TestRunAll_WithData(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Insert old invite
	_, _ = db.Exec(
		"INSERT INTO invites (code, baby_id, created_by, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		"999999", baby.ID, user.ID,
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
		time.Now().Add(-25*time.Hour).UTC().Format(time.DateTime),
	)

	// Insert expired session
	_, _ = db.Exec(
		"INSERT INTO sessions (id, user_id, token, expires_at) VALUES (?, ?, ?, ?)",
		"sess-old", user.ID, "tok-old",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
	)

	// Insert orphan photo (null baby_id)
	_, _ = db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at) VALUES (?, NULL, ?, ?, ?)",
		"photo-runall", "photos/runall.jpg", "photos/thumb_runall.jpg",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
	)

	memStore := storage.NewMemoryStore()
	memStore.Put(context.Background(), "photos/runall.jpg", strings.NewReader("data"), "image/jpeg")
	memStore.Put(context.Background(), "photos/thumb_runall.jpg", strings.NewReader("data"), "image/jpeg")

	err := cron.RunAll(db, memStore)
	if err != nil {
		t.Fatalf("RunAll failed: %v", err)
	}

	// Verify all cleaned up
	var count int
	db.QueryRow("SELECT COUNT(*) FROM invites WHERE code = '999999'").Scan(&count)
	if count != 0 {
		t.Errorf("expected old invite deleted, got count=%d", count)
	}
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = 'sess-old'").Scan(&count)
	if count != 0 {
		t.Errorf("expected expired session deleted, got count=%d", count)
	}
	db.QueryRow("SELECT COUNT(*) FROM photo_uploads WHERE id = 'photo-runall'").Scan(&count)
	if count != 0 {
		t.Errorf("expected orphan photo deleted, got count=%d", count)
	}
}

func TestRunAll_InviteError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	db.Close() // close to force error

	memStore := storage.NewMemoryStore()
	err := cron.RunAll(db, memStore)
	if err == nil {
		t.Fatal("expected error from RunAll with closed DB")
	}
}

// failingStore is an ObjectStore that always returns errors on Delete.
type failingStore struct{}

func (f *failingStore) Put(_ context.Context, _ string, _ io.Reader, _ string) error {
	return fmt.Errorf("put not supported")
}

func (f *failingStore) Delete(_ context.Context, _ string) error {
	return fmt.Errorf("simulated R2 delete error")
}

func (f *failingStore) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, fmt.Errorf("get not supported")
}

func (f *failingStore) SignedURL(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("signed url not supported")
}

func TestCleanupPhotos_R2DeleteError_SkipsRow(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	// Insert orphan photo (baby_id IS NULL)
	_, err := db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at) VALUES (?, NULL, ?, ?, ?)",
		"photo-r2err", "photos/r2err.jpg", "photos/thumb_r2err.jpg",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
	)
	if err != nil {
		t.Fatalf("insert photo: %v", err)
	}

	fs := &failingStore{}
	now := time.Now().UTC()
	deleted, err := cron.CleanupPhotos(db, fs, now)
	if err != nil {
		t.Fatalf("CleanupPhotos failed: %v", err)
	}
	// Should skip deletion because R2 delete failed
	if deleted != 0 {
		t.Errorf("expected 0 deleted (R2 error skips), got %d", deleted)
	}

	// DB row should still exist since R2 delete failed
	var count int
	db.QueryRow("SELECT COUNT(*) FROM photo_uploads WHERE id = 'photo-r2err'").Scan(&count)
	if count != 1 {
		t.Errorf("expected photo row to still exist, got count=%d", count)
	}
}

func TestRunAll_SessionsError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	// Drop sessions table to force error in CleanupSessions
	_, err := db.Exec("DROP TABLE sessions")
	if err != nil {
		t.Fatalf("drop sessions table: %v", err)
	}

	memStore := storage.NewMemoryStore()
	err = cron.RunAll(db, memStore)
	if err == nil {
		t.Fatal("expected error from RunAll with missing sessions table")
	}
}

func TestRunAll_PhotosError(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	// Drop photo_uploads table to force error in CleanupPhotos
	_, err := db.Exec("DROP TABLE photo_uploads")
	if err != nil {
		t.Fatalf("drop photo_uploads table: %v", err)
	}

	memStore := storage.NewMemoryStore()
	err = cron.RunAll(db, memStore)
	if err == nil {
		t.Fatal("expected error from RunAll with missing photo_uploads table")
	}
}

// thumbFailStore fails only on thumbnail key deletes.
type thumbFailStore struct {
	inner *storage.MemoryStore
}

func (s *thumbFailStore) Put(ctx context.Context, key string, r io.Reader, ct string) error {
	return s.inner.Put(ctx, key, r, ct)
}

func (s *thumbFailStore) Delete(ctx context.Context, key string) error {
	if strings.HasPrefix(key, "photos/thumb_") {
		return fmt.Errorf("simulated thumbnail delete error")
	}
	return s.inner.Delete(ctx, key)
}

func (s *thumbFailStore) Get(ctx context.Context, key string) ([]byte, error) {
	return s.inner.Get(ctx, key)
}

func (s *thumbFailStore) SignedURL(ctx context.Context, key string) (string, error) {
	return s.inner.SignedURL(ctx, key)
}

func TestCleanupPhotos_ThumbnailDeleteError_StillDeletesRow(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	// Insert orphan photo with thumbnail
	_, err := db.Exec(
		"INSERT INTO photo_uploads (id, baby_id, r2_key, thumbnail_key, uploaded_at) VALUES (?, NULL, ?, ?, ?)",
		"photo-thumberr", "photos/thumberr.jpg", "photos/thumb_thumberr.jpg",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime),
	)
	if err != nil {
		t.Fatalf("insert photo: %v", err)
	}

	inner := storage.NewMemoryStore()
	inner.Put(context.Background(), "photos/thumberr.jpg", strings.NewReader("data"), "image/jpeg")
	inner.Put(context.Background(), "photos/thumb_thumberr.jpg", strings.NewReader("data"), "image/jpeg")
	tfs := &thumbFailStore{inner: inner}

	now := time.Now().UTC()
	deleted, err := cron.CleanupPhotos(db, tfs, now)
	if err != nil {
		t.Fatalf("CleanupPhotos failed: %v", err)
	}
	// Should still delete the row even if thumbnail delete fails
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM photo_uploads WHERE id = 'photo-thumberr'").Scan(&count)
	if count != 0 {
		t.Errorf("expected photo row to be deleted, got count=%d", count)
	}
}

func TestRunner_StartStopWithErrors(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)

	// Drop a table to force errors in RunAll during the runner
	_, _ = db.Exec("DROP TABLE invites")

	memStore := storage.NewMemoryStore()
	runner := cron.NewRunner(db, memStore, 50*time.Millisecond)
	runner.Start()
	time.Sleep(80 * time.Millisecond)
	runner.Stop()
	db.Close()
	// Just ensure it doesn't panic
}

// Ensure sql.DB is used (suppress linter)
var _ *sql.DB
