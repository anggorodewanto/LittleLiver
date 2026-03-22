package storage_test

import (
	"bytes"
	"context"
	"strings"
	"sync"
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

	got, ct, ok := store.GetWithMeta("test/key.jpg")
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

	_, _, ok := store.GetWithMeta("test/key.jpg")
	if ok {
		t.Error("expected object to be deleted")
	}
}

func TestMemoryStore_Delete_Nonexistent(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	// Deleting a nonexistent key should not error
	err := store.Delete(ctx, "nonexistent")
	if err != nil {
		t.Errorf("expected no error for deleting nonexistent key, got: %v", err)
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
	if !strings.Contains(url, "test/key.jpg") {
		t.Errorf("expected URL to contain key, got %q", url)
	}
	if !strings.Contains(url, "signed=true") {
		t.Errorf("expected URL to contain signed=true, got %q", url)
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
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
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

func TestMemoryStore_Keys_Empty(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()

	keys := store.Keys()
	if len(keys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(keys))
	}
}

func TestMemoryStore_Get(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	data := []byte("test data")
	_ = store.Put(ctx, "test/get.jpg", bytes.NewReader(data), "image/jpeg")

	got, err := store.Get(ctx, "test/get.jpg")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("data mismatch: got %q, want %q", got, data)
	}
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestMemoryStore_GetWithMeta_NotFound(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()

	data, ct, ok := store.GetWithMeta("nonexistent")
	if ok {
		t.Error("expected ok=false for nonexistent key")
	}
	if data != nil {
		t.Errorf("expected nil data, got %v", data)
	}
	if ct != "" {
		t.Errorf("expected empty content type, got %q", ct)
	}
}

func TestMemoryStore_Put_Overwrite(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	_ = store.Put(ctx, "key", bytes.NewReader([]byte("original")), "text/plain")
	_ = store.Put(ctx, "key", bytes.NewReader([]byte("updated")), "application/json")

	got, ct, ok := store.GetWithMeta("key")
	if !ok {
		t.Fatal("expected object to exist")
	}
	if string(got) != "updated" {
		t.Errorf("expected overwritten data, got %q", got)
	}
	if ct != "application/json" {
		t.Errorf("expected overwritten content type, got %q", ct)
	}
}

func TestMemoryStore_Put_LargeData(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	large := make([]byte, 5*1024*1024) // 5MB
	for i := range large {
		large[i] = byte(i % 256)
	}

	err := store.Put(ctx, "large.bin", bytes.NewReader(large), "application/octet-stream")
	if err != nil {
		t.Fatalf("Put large data failed: %v", err)
	}

	got, err := store.Get(ctx, "large.bin")
	if err != nil {
		t.Fatalf("Get large data failed: %v", err)
	}
	if len(got) != len(large) {
		t.Errorf("expected %d bytes, got %d", len(large), len(got))
	}
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a'+i%26)) + ".jpg"
			_ = store.Put(ctx, key, bytes.NewReader([]byte("data")), "image/jpeg")
			_, _ = store.Get(ctx, key)
			_, _ = store.SignedURL(ctx, key)
			_ = store.Keys()
		}(i)
	}
	wg.Wait()
}

func TestMemoryStore_Put_EmptyData(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	err := store.Put(ctx, "empty", bytes.NewReader([]byte{}), "text/plain")
	if err != nil {
		t.Fatalf("Put empty data failed: %v", err)
	}

	got, err := store.Get(ctx, "empty")
	if err != nil {
		t.Fatalf("Get empty data failed: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 bytes, got %d", len(got))
	}
}

func TestMemoryStore_MultipleKeys_IndependentLifecycle(t *testing.T) {
	t.Parallel()
	store := storage.NewMemoryStore()
	ctx := context.Background()

	_ = store.Put(ctx, "a", bytes.NewReader([]byte("dataA")), "text/plain")
	_ = store.Put(ctx, "b", bytes.NewReader([]byte("dataB")), "text/plain")

	// Delete only 'a'
	_ = store.Delete(ctx, "a")

	// 'a' is gone
	_, err := store.Get(ctx, "a")
	if err == nil {
		t.Error("expected error after deleting 'a'")
	}

	// 'b' is still there
	got, err := store.Get(ctx, "b")
	if err != nil {
		t.Fatalf("expected 'b' to still exist: %v", err)
	}
	if string(got) != "dataB" {
		t.Errorf("expected 'dataB', got %q", got)
	}

	keys := store.Keys()
	if len(keys) != 1 {
		t.Errorf("expected 1 key remaining, got %d", len(keys))
	}
}

// Verify MemoryStore implements ObjectStore interface.
var _ storage.ObjectStore = (*storage.MemoryStore)(nil)
