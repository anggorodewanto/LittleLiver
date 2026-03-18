package store

import (
	"testing"
)

func TestUpsertPushSubscription_InsertsNew(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	// Insert a user first (FK constraint)
	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	sub, err := UpsertPushSubscription(db, "u1", "https://push.example.com/sub1", "p256dh-key-1", "auth-key-1")
	if err != nil {
		t.Fatalf("UpsertPushSubscription: %v", err)
	}

	if sub.ID == "" {
		t.Error("expected non-empty ID")
	}
	if sub.UserID != "u1" {
		t.Errorf("expected UserID=u1, got %q", sub.UserID)
	}
	if sub.Endpoint != "https://push.example.com/sub1" {
		t.Errorf("expected endpoint, got %q", sub.Endpoint)
	}
	if sub.P256dh != "p256dh-key-1" {
		t.Errorf("expected p256dh, got %q", sub.P256dh)
	}
	if sub.Auth != "auth-key-1" {
		t.Errorf("expected auth, got %q", sub.Auth)
	}
	if sub.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestUpsertPushSubscription_UpsertsOnDuplicateEndpoint(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// First insert
	sub1, err := UpsertPushSubscription(db, "u1", "https://push.example.com/dup", "old-p256dh", "old-auth")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Second insert with same endpoint but different keys — should upsert
	sub2, err := UpsertPushSubscription(db, "u1", "https://push.example.com/dup", "new-p256dh", "new-auth")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	// ID should be the same (upserted, not a new row)
	if sub2.ID != sub1.ID {
		t.Errorf("expected same ID after upsert, got %q vs %q", sub1.ID, sub2.ID)
	}
	// Keys should be updated
	if sub2.P256dh != "new-p256dh" {
		t.Errorf("expected updated p256dh, got %q", sub2.P256dh)
	}
	if sub2.Auth != "new-auth" {
		t.Errorf("expected updated auth, got %q", sub2.Auth)
	}

	// Verify only one row exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM push_subscriptions WHERE endpoint = 'https://push.example.com/dup'").Scan(&count)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func TestDeletePushSubscription_DeletesByEndpoint(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	_, err = UpsertPushSubscription(db, "u1", "https://push.example.com/del", "key", "auth")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	err = DeletePushSubscription(db, "u1", "https://push.example.com/del")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Verify deleted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM push_subscriptions WHERE endpoint = 'https://push.example.com/del'").Scan(&count)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows after delete, got %d", count)
	}
}

func TestDeletePushSubscription_NotFoundReturnsNoRowsError(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	err = DeletePushSubscription(db, "u1", "https://push.example.com/nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent subscription, got nil")
	}
}

func TestGetPushSubscriptionsByUserID(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Insert two subscriptions
	_, err = UpsertPushSubscription(db, "u1", "https://push.example.com/1", "k1", "a1")
	if err != nil {
		t.Fatalf("upsert 1: %v", err)
	}
	_, err = UpsertPushSubscription(db, "u1", "https://push.example.com/2", "k2", "a2")
	if err != nil {
		t.Fatalf("upsert 2: %v", err)
	}

	subs, err := GetPushSubscriptionsByUserID(db, "u1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 subs, got %d", len(subs))
	}
}

func TestDeletePushSubscription_OnlyDeletesOwnSubscription(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test1')")
	if err != nil {
		t.Fatalf("insert user1: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u2', 'g2', 'b@b.com', 'Test2')")
	if err != nil {
		t.Fatalf("insert user2: %v", err)
	}

	// u1 subscribes
	_, err = UpsertPushSubscription(db, "u1", "https://push.example.com/u1", "k1", "a1")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// u2 tries to delete u1's subscription — should fail (not their subscription)
	err = DeletePushSubscription(db, "u2", "https://push.example.com/u1")
	if err == nil {
		t.Fatal("expected error when deleting another user's subscription")
	}

	// Verify subscription still exists
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM push_subscriptions WHERE endpoint = 'https://push.example.com/u1'").Scan(&count)
	if count != 1 {
		t.Errorf("expected subscription to still exist, got count=%d", count)
	}
}


func TestGetPushSubscriptionsByUserID_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('u1', 'g1', 'a@b.com', 'Test')")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	subs, err := GetPushSubscriptionsByUserID(db, "u1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if subs == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 subs, got %d", len(subs))
	}
}

// Verify that cascade delete from users also deletes push_subscriptions
func TestPushSubscription_CascadeDeleteOnUser(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users (id, google_id, email, name) VALUES ('ucd', 'gcd', 'cd@b.com', 'Cascade')")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	_, err = UpsertPushSubscription(db, "ucd", "https://push.example.com/cascade", "k", "a")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	_, err = db.Exec("DELETE FROM users WHERE id = 'ucd'")
	if err != nil {
		t.Fatalf("delete user: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM push_subscriptions WHERE user_id = 'ucd'").Scan(&count)
	if count != 0 {
		t.Errorf("expected cascade delete to remove subscriptions, got count=%d", count)
	}
}

// Helper to verify UpsertPushSubscription fails with invalid user (FK violation)
func TestUpsertPushSubscription_InvalidUserFails(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpsertPushSubscription(db, "nonexistent-user", "https://push.example.com/x", "k", "a")
	if err == nil {
		t.Fatal("expected FK violation error, got nil")
	}
}

// setupTestDB is already defined in db_test.go and accessible since we're in the same package.
