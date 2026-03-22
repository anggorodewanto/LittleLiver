package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// mockS3API implements s3API for testing.
type mockS3API struct {
	putObjectFunc    func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	getObjectFunc    func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	deleteObjectFunc func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	headBucketFunc   func(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

func (m *mockS3API) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3API) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getObjectFunc != nil {
		return m.getObjectFunc(ctx, params, optFns...)
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(nil))}, nil
}

func (m *mockS3API) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.deleteObjectFunc != nil {
		return m.deleteObjectFunc(ctx, params, optFns...)
	}
	return &s3.DeleteObjectOutput{}, nil
}

func (m *mockS3API) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	if m.headBucketFunc != nil {
		return m.headBucketFunc(ctx, params, optFns...)
	}
	return &s3.HeadBucketOutput{}, nil
}

// mockPresigner implements s3Presigner for testing.
type mockPresigner struct {
	presignGetObjectFunc func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

func (m *mockPresigner) PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	if m.presignGetObjectFunc != nil {
		return m.presignGetObjectFunc(ctx, params, optFns...)
	}
	return &v4.PresignedHTTPRequest{URL: "https://fake.example.com/signed"}, nil
}

func TestR2Client_Put_Success(t *testing.T) {
	t.Parallel()
	var capturedKey, capturedBucket, capturedCT string
	api := &mockS3API{
		putObjectFunc: func(_ context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			capturedKey = *params.Key
			capturedBucket = *params.Bucket
			capturedCT = *params.ContentType
			return &s3.PutObjectOutput{}, nil
		},
	}
	client := newR2ClientWithMocks(api, &mockPresigner{}, "test-bucket")

	err := client.Put(context.Background(), "photos/test.jpg", bytes.NewReader([]byte("data")), "image/jpeg")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if capturedKey != "photos/test.jpg" {
		t.Errorf("expected key 'photos/test.jpg', got %q", capturedKey)
	}
	if capturedBucket != "test-bucket" {
		t.Errorf("expected bucket 'test-bucket', got %q", capturedBucket)
	}
	if capturedCT != "image/jpeg" {
		t.Errorf("expected content type 'image/jpeg', got %q", capturedCT)
	}
}

func TestR2Client_Put_Error(t *testing.T) {
	t.Parallel()
	api := &mockS3API{
		putObjectFunc: func(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			return nil, fmt.Errorf("network error")
		},
	}
	client := newR2ClientWithMocks(api, &mockPresigner{}, "test-bucket")

	err := client.Put(context.Background(), "key", bytes.NewReader(nil), "text/plain")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "put object key") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestR2Client_Get_Success(t *testing.T) {
	t.Parallel()
	data := []byte("photo-bytes")
	api := &mockS3API{
		getObjectFunc: func(_ context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{
				Body: io.NopCloser(bytes.NewReader(data)),
			}, nil
		},
	}
	client := newR2ClientWithMocks(api, &mockPresigner{}, "test-bucket")

	got, err := client.Get(context.Background(), "photos/test.jpg")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("data mismatch: got %q, want %q", got, data)
	}
}

func TestR2Client_Get_Error(t *testing.T) {
	t.Parallel()
	api := &mockS3API{
		getObjectFunc: func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	client := newR2ClientWithMocks(api, &mockPresigner{}, "test-bucket")

	_, err := client.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "get object missing") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestR2Client_Delete_Success(t *testing.T) {
	t.Parallel()
	var capturedKey string
	api := &mockS3API{
		deleteObjectFunc: func(_ context.Context, params *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			capturedKey = *params.Key
			return &s3.DeleteObjectOutput{}, nil
		},
	}
	client := newR2ClientWithMocks(api, &mockPresigner{}, "test-bucket")

	err := client.Delete(context.Background(), "photos/old.jpg")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if capturedKey != "photos/old.jpg" {
		t.Errorf("expected key 'photos/old.jpg', got %q", capturedKey)
	}
}

func TestR2Client_Delete_Error(t *testing.T) {
	t.Parallel()
	api := &mockS3API{
		deleteObjectFunc: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			return nil, fmt.Errorf("permission denied")
		},
	}
	client := newR2ClientWithMocks(api, &mockPresigner{}, "test-bucket")

	err := client.Delete(context.Background(), "key")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "delete object key") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestR2Client_SignedURL_Success(t *testing.T) {
	t.Parallel()
	presigner := &mockPresigner{
		presignGetObjectFunc: func(_ context.Context, params *s3.GetObjectInput, _ ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
			return &v4.PresignedHTTPRequest{
				URL: fmt.Sprintf("https://r2.example.com/%s?signed=true&expires=3600", *params.Key),
			}, nil
		},
	}
	client := newR2ClientWithMocks(&mockS3API{}, presigner, "test-bucket")

	url, err := client.SignedURL(context.Background(), "photos/test.jpg")
	if err != nil {
		t.Fatalf("SignedURL failed: %v", err)
	}
	if !strings.Contains(url, "photos/test.jpg") {
		t.Errorf("expected URL to contain key, got %q", url)
	}
	if !strings.Contains(url, "signed=true") {
		t.Errorf("expected URL to contain signed=true, got %q", url)
	}
}

func TestR2Client_SignedURL_Error(t *testing.T) {
	t.Parallel()
	presigner := &mockPresigner{
		presignGetObjectFunc: func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
			return nil, fmt.Errorf("presign error")
		},
	}
	client := newR2ClientWithMocks(&mockS3API{}, presigner, "test-bucket")

	_, err := client.SignedURL(context.Background(), "key")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "presign object key") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestR2Client_HealthCheck_Success(t *testing.T) {
	t.Parallel()
	var capturedBucket string
	api := &mockS3API{
		headBucketFunc: func(_ context.Context, params *s3.HeadBucketInput, _ ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
			capturedBucket = *params.Bucket
			return &s3.HeadBucketOutput{}, nil
		},
	}
	client := newR2ClientWithMocks(api, &mockPresigner{}, "test-bucket")

	err := client.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	if capturedBucket != "test-bucket" {
		t.Errorf("expected bucket 'test-bucket', got %q", capturedBucket)
	}
}

func TestR2Client_HealthCheck_Error(t *testing.T) {
	t.Parallel()
	api := &mockS3API{
		headBucketFunc: func(_ context.Context, _ *s3.HeadBucketInput, _ ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
			return nil, fmt.Errorf("bucket not found")
		},
	}
	client := newR2ClientWithMocks(api, &mockPresigner{}, "test-bucket")

	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "r2 health check") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestSignedURLExpiry_IsOneHour(t *testing.T) {
	t.Parallel()
	if SignedURLExpiry.Hours() != 1.0 {
		t.Errorf("expected SignedURLExpiry=1h, got %v", SignedURLExpiry)
	}
}
