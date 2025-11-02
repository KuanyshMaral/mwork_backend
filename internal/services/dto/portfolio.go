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
