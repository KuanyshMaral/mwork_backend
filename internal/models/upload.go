package models

import (
	"time"
)

type Upload struct {
	BaseModel
	UserID     string `gorm:"not null;index"`
	EntityType string // "model_profile", "portfolio", "casting"
	EntityID   string
	FileType   string // "image", "video", "document"
	Usage      string // "avatar", "portfolio_photo", "casting_attachment"
	Path       string `gorm:"not null"`
	MimeType   string
	Size       int64
	IsPublic   bool `gorm:"default:true"`

	OriginalName    string     `gorm:"column:original_name"`                    // Original filename from user
	URL             string     `gorm:"column:url"`                              // Public URL for accessing the file
	ThumbnailPath   string     `gorm:"column:thumbnail_path"`                   // Path to thumbnail (for images)
	Variants        string     `gorm:"column:variants;type:jsonb"`              // JSON with different sizes
	Metadata        string     `gorm:"column:metadata;type:jsonb"`              // Additional metadata
	StorageProvider string     `gorm:"column:storage_provider;default:'local'"` // 'local', 's3', 'cloudflare_r2'
	ExpiresAt       *time.Time `gorm:"column:expires_at"`                       // For temporary files
	DownloadCount   int        `gorm:"column:download_count;default:0"`         // Track downloads
	LastAccessedAt  *time.Time `gorm:"column:last_accessed_at"`                 // Last access time
}
