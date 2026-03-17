package storage_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/storage"
)

func TestMemoryStore_PutAndGet(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	data := []byte("hello world")
	err := store.Put(ctx, "test/key.jpg", bytes.NewReader(data), "image/jpeg")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, ct, ok := store.Get("test/key.jpg")
	if !ok {
		t.Fatal("expected object to exist")
	}
	if !bytes.Equal(got, data) {
		t.Errorf("data mismatch: got %q, want %q", got, data)
	}
	if ct != "image/jpeg" {
		t.Errorf("content type mismatch: got %q, want %q", ct, "image/jpeg")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	_ = store.Put(ctx, "test/key.jpg", bytes.NewReader([]byte("data")), "image/jpeg")
	err := store.Delete(ctx, "test/key.jpg")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, _, ok := store.Get("test/key.jpg")
	if ok {
		t.Error("expected object to be deleted")
	}
}

func TestMemoryStore_SignedURL(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	_ = store.Put(ctx, "test/key.jpg", bytes.NewReader([]byte("data")), "image/jpeg")

	url, err := store.SignedURL(ctx, "test/key.jpg")
	if err != nil {
		t.Fatalf("SignedURL failed: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty URL")
	}
}

func TestMemoryStore_SignedURL_NotFound(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	_, err := store.SignedURL(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
}

func TestMemoryStore_Keys(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	_ = store.Put(ctx, "a.jpg", bytes.NewReader([]byte("a")), "image/jpeg")
	_ = store.Put(ctx, "b.png", bytes.NewReader([]byte("b")), "image/png")

	keys := store.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

// Verify MemoryStore implements ObjectStore interface.
var _ storage.ObjectStore = (*storage.MemoryStore)(nil)
