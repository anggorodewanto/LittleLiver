package store

import (
	"testing"
)

func TestUpsertUser_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpsertUser(db, "google123", "test@example.com", "Test User")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestUpsertUser_Insert(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google123", "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	if user.GoogleID != "google123" {
		t.Errorf("expected google_id=google123, got %q", user.GoogleID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email=test@example.com, got %q", user.Email)
	}
	if user.Name != "Test User" {
		t.Errorf("expected name=Test User, got %q", user.Name)
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
}

func TestUpsertUser_Update(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	first, err := UpsertUser(db, "google123", "old@example.com", "Old Name")
	if err != nil {
		t.Fatalf("first UpsertUser failed: %v", err)
	}

	second, err := UpsertUser(db, "google123", "new@example.com", "New Name")
	if err != nil {
		t.Fatalf("second UpsertUser failed: %v", err)
	}

	if second.ID != first.ID {
		t.Errorf("expected same ID on upsert, got first=%q second=%q", first.ID, second.ID)
	}
	if second.Email != "new@example.com" {
		t.Errorf("expected updated email=new@example.com, got %q", second.Email)
	}
	if second.Name != "New Name" {
		t.Errorf("expected updated name=New Name, got %q", second.Name)
	}
}
