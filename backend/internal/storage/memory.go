package storage

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// storedObject holds the data and content type for an in-memory stored object.
type storedObject struct {
	Data        []byte
	ContentType string
}

// MemoryStore is an in-memory implementation of ObjectStore for testing.
type MemoryStore struct {
	mu      sync.RWMutex
	objects map[string]*storedObject
}

// NewMemoryStore creates a new MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		objects: make(map[string]*storedObject),
	}
}

// Put stores data in memory under the given key.
func (m *MemoryStore) Put(_ context.Context, key string, r io.Reader, contentType string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read data: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = &storedObject{Data: data, ContentType: contentType}
	return nil
}

// Delete removes an object from memory.
func (m *MemoryStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	return nil
}

// SignedURL returns a fake signed URL for testing.
func (m *MemoryStore) SignedURL(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.objects[key]; !ok {
		return "", fmt.Errorf("object not found: %s", key)
	}
	return fmt.Sprintf("https://fake-r2.example.com/%s?signed=true", key), nil
}

// Get retrieves the object data at the given key (implements ObjectStore).
func (m *MemoryStore) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	obj, ok := m.objects[key]
	if !ok {
		return nil, fmt.Errorf("object not found: %s", key)
	}
	return obj.Data, nil
}

// GetWithMeta retrieves the stored object data with content type (test helper).
func (m *MemoryStore) GetWithMeta(key string) ([]byte, string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	obj, ok := m.objects[key]
	if !ok {
		return nil, "", false
	}
	return obj.Data, obj.ContentType, true
}

// Keys returns all stored keys (test helper).
func (m *MemoryStore) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.objects))
	for k := range m.objects {
		keys = append(keys, k)
	}
	return keys
}
