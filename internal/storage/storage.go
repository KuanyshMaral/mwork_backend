package storage

import (
	"context"
	"fmt"
	"io"
	"time"
)

// Storage defines the interface for file storage operations
type Storage interface {
	// Save stores a file at the given path
	Save(ctx context.Context, path string, reader io.Reader, contentType string) error

	// Get retrieves a file from the given path
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes a file at the given path
	Delete(ctx context.Context, path string) error

	// Exists checks if a file exists at the given path
	Exists(ctx context.Context, path string) (bool, error)

	// GetURL returns a public URL for the file
	GetURL(ctx context.Context, path string) (string, error)

	// GetSignedURL returns a temporary signed URL for private files
	GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)

	// GetSize returns the size of a file in bytes
	GetSize(ctx context.Context, path string) (int64, error)
}

// Config holds storage configuration
type Config struct {
	Type       string // local, s3, cloudflare_r2
	BasePath   string // For local storage
	BaseURL    string // Public URL base
	Bucket     string // For S3/R2
	Region     string // For S3
	AccessKey  string // For S3/R2
	SecretKey  string // For S3/R2
	Endpoint   string // For R2 or custom S3
	UseSSL     bool   // For S3/R2
	PublicRead bool   // Make files public by default
}

// NewStorage creates a new storage instance based on configuration
func NewStorage(cfg Config) (Storage, error) {
	switch cfg.Type {
	case "local":
		return NewLocalStorage(cfg)
	case "s3":
		return NewS3Storage(cfg)
	case "cloudflare_r2":
		return NewCloudflareR2Storage(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Type)
	}
}
