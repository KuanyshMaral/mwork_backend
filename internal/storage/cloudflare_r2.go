package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// CloudflareR2Storage implements Storage interface for Cloudflare R2
// R2 is S3-compatible, so we use the same SDK
type CloudflareR2Storage struct {
	client   *s3.S3
	uploader *s3manager.Uploader
	bucket   string
	baseURL  string
}

// NewCloudflareR2Storage creates a new Cloudflare R2 storage instance
func NewCloudflareR2Storage(cfg Config) (*CloudflareR2Storage, error) {
	// R2 endpoint format: https://<account_id>.r2.cloudflarestorage.com
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required for Cloudflare R2")
	}

	awsConfig := &aws.Config{
		Region:           aws.String("auto"),
		Endpoint:         aws.String(cfg.Endpoint),
		Credentials:      credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create R2 session: %w", err)
	}

	client := s3.New(sess)
	uploader := s3manager.NewUploader(sess)

	baseURL := cfg.BaseURL
	if baseURL == "" {
		// Use R2 public URL if configured
		baseURL = fmt.Sprintf("https://%s.r2.dev", cfg.Bucket)
	}

	return &CloudflareR2Storage{
		client:   client,
		uploader: uploader,
		bucket:   cfg.Bucket,
		baseURL:  baseURL,
	}, nil
}

// Save uploads a file to R2
func (s *CloudflareR2Storage) Save(ctx context.Context, path string, reader io.Reader, contentType string) error {
	input := &s3manager.UploadInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(path),
		Body:        reader,
		ContentType: aws.String(contentType),
	}

	_, err := s.uploader.UploadWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to R2: %w", err)
	}

	return nil
}

// Get retrieves a file from R2
func (s *CloudflareR2Storage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}

	result, err := s.client.GetObjectWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get from R2: %w", err)
	}

	return result.Body, nil
}

// Delete removes a file from R2
func (s *CloudflareR2Storage) Delete(ctx context.Context, path string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}

	_, err := s.client.DeleteObjectWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	return nil
}

// Exists checks if a file exists in R2
func (s *CloudflareR2Storage) Exists(ctx context.Context, path string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}

	_, err := s.client.HeadObjectWithContext(ctx, input)
	if err != nil {
		return false, nil
	}

	return true, nil
}

// GetURL returns a public URL for the file
func (s *CloudflareR2Storage) GetURL(ctx context.Context, path string) (string, error) {
	return fmt.Sprintf("%s/%s", s.baseURL, path), nil
}

// GetSignedURL returns a temporary signed URL
func (s *CloudflareR2Storage) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}

	req, _ := s.client.GetObjectRequest(input)
	url, err := req.Presign(expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return url, nil
}

// GetSize returns the size of a file in R2
func (s *CloudflareR2Storage) GetSize(ctx context.Context, path string) (int64, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}

	result, err := s.client.HeadObjectWithContext(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}

	return *result.ContentLength, nil
}
