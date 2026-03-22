package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Config holds the configuration for connecting to Cloudflare R2.
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
}

// s3API abstracts the S3 client operations for testability.
type s3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

// s3Presigner abstracts the S3 presign client for testability.
type s3Presigner interface {
	PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// R2Client is a Cloudflare R2 implementation of ObjectStore using the S3-compatible API.
type R2Client struct {
	client     s3API
	presigner  s3Presigner
	bucketName string
}

// NewR2Client creates a new R2Client with the given configuration.
func NewR2Client(ctx context.Context, cfg R2Config) (*R2Client, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		// R2 requires path-style addressing
		o.UsePathStyle = true
	})

	return &R2Client{
		client:     client,
		presigner:  s3.NewPresignClient(client),
		bucketName: cfg.BucketName,
	}, nil
}

// newR2ClientWithMocks creates an R2Client with injected mocks (for testing).
func newR2ClientWithMocks(api s3API, presigner s3Presigner, bucket string) *R2Client {
	return &R2Client{
		client:     api,
		presigner:  presigner,
		bucketName: bucket,
	}
}

// Put uploads data to R2 under the given key with the specified content type.
func (r *R2Client) Put(ctx context.Context, key string, reader io.Reader, contentType string) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}

// Get retrieves the object data at the given key from R2.
func (r *R2Client) Get(ctx context.Context, key string) ([]byte, error) {
	resp, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", key, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read object %s: %w", key, err)
	}
	return data, nil
}

// Delete removes the object at the given key from R2.
func (r *R2Client) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object %s: %w", key, err)
	}
	return nil
}

// SignedURLExpiry is the TTL for presigned object URLs.
const SignedURLExpiry = 1 * time.Hour

// SignedURL returns a presigned URL for accessing the object at the given key.
func (r *R2Client) SignedURL(ctx context.Context, key string) (string, error) {
	presigned, err := r.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = SignedURLExpiry
	})
	if err != nil {
		return "", fmt.Errorf("presign object %s: %w", key, err)
	}
	return presigned.URL, nil
}

// Verify R2Client implements ObjectStore at compile time.
var _ ObjectStore = (*R2Client)(nil)

// HealthCheck verifies R2 connectivity by performing a HeadBucket operation.
func (r *R2Client) HealthCheck(ctx context.Context) error {
	_, err := r.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(r.bucketName),
	})
	if err != nil {
		return fmt.Errorf("r2 health check: %w", err)
	}
	return nil
}
