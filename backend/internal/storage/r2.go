// Package storage provides object storage functionality for LittleLiver.
package storage

import (
	"context"
	"io"
)

// ObjectStore defines the interface for object storage operations (R2/S3-compatible).
type ObjectStore interface {
	// Put uploads data to the store under the given key with the specified content type.
	Put(ctx context.Context, key string, r io.Reader, contentType string) error

	// Delete removes the object at the given key.
	Delete(ctx context.Context, key string) error

	// SignedURL returns a time-limited URL for accessing the object at the given key.
	SignedURL(ctx context.Context, key string) (string, error)
}
