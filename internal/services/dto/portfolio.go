package dto

import (
	"time"
)

// Portfolio Request DTOs

type CreatePortfolioRequest struct {
	ModelID     string `json:"model_id" validate:"-"` // Set by server from auth
	Title       string `json:"title" validate:"required,min=3,max=100"`
	Description string `json:"description" validate:"omitempty,max=1000"`
	OrderIndex  int    `json:"order_index" validate:"omitempty,min=0"`
}

type UpdatePortfolioRequest struct {
	Title       *string `json:"title,omitempty" validate:"omitempty,min=3,max=100"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
	OrderIndex  *int    `json:"order_index,omitempty" validate:"omitempty,min=0"`
}

type UploadRequest struct {
	EntityType string `json:"entity_type" validate:"required"` // model_profile, portfolio, casting
	EntityID   string `json:"entity_id" validate:"required"`
	FileType   string `json:"file_type" validate:"required"` // image, video, document
	Usage      string `json:"usage" validate:"required"`     // avatar, portfolio_photo, casting_attachment
	IsPublic   bool   `json:"is_public"`
}

// Portfolio Response DTOs

type PortfolioResponse struct {
	ID          string          `json:"id"`
	ModelID     string          `json:"model_id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	OrderIndex  int             `json:"order_index"`
	Upload      *UploadResponse `json:"upload"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type UploadResponse struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	FileType   string    `json:"file_type"`
	Usage      string    `json:"usage"`
	Path       string    `json:"path"`
	MimeType   string    `json:"mime_type"`
	Size       int64     `json:"size"`
	IsPublic   bool      `json:"is_public"`
	URL        string    `json:"url"` // Generated URL for accessing the file
	CreatedAt  time.Time `json:"created_at"`
}

type UploadStats struct {
	TotalUploads int64            `json:"total_uploads"`
	TotalSize    int64            `json:"total_size"`
	ByFileType   map[string]int64 `json:"by_file_type"`
	ByUsage      map[string]int64 `json:"by_usage"`
	ActiveUsers  int64            `json:"active_users"`
	StorageUsed  int64            `json:"storage_used"`
	StorageLimit int64            `json:"storage_limit"`
}

// Portfolio Configuration DTO

type FileConfigPortfolio struct {
	MaxSize        int64
	AllowedTypes   []string
	AllowedUsages  map[string][]string
	StoragePath    string
	MaxUserStorage int64 // 100MB default
}

// Portfolio List Responses

type PortfolioListResponse struct {
	Items []*PortfolioResponse `json:"items"`
	Total int                  `json:"total"`
}

type UploadListResponse struct {
	Uploads []*UploadResponse `json:"uploads"`
	Total   int               `json:"total"`
}

// Portfolio Reorder DTO

type ReorderPortfolioRequest struct {
	ItemIDs []string `json:"item_ids" binding:"required"`
}

// Portfolio Visibility DTO

type PortfolioVisibilityRequest struct {
	IsPublic bool `json:"is_public" binding:"required"`
}

// Storage Usage DTO

type StorageUsageResponse struct {
	Used  int64 `json:"used"`
	Limit int64 `json:"limit"`
}

// File Upload Response

type FileUploadResponse struct {
	UploadID string `json:"upload_id"`
	URL      string `json:"url"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
}

// Portfolio Stats Response

type PortfolioStatsResponse struct {
	TotalItems  int       `json:"total_items"`
	TotalViews  int64     `json:"total_views"`
	TotalLikes  int64     `json:"total_likes"`
	TotalShares int64     `json:"total_shares"`
	LastUpdated time.Time `json:"last_updated"`
}
