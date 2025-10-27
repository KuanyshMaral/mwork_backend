package dto

import "time"

// ======================
// Request DTOs
// ======================

type CreateReviewRequest struct {
	ModelID    string  `json:"model_id" binding:"required"`
	EmployerID string  `json:"employer_id" binding:"required"`
	CastingID  *string `json:"casting_id,omitempty"`
	Rating     int     `json:"rating" binding:"required,min=1,max=5"`
	ReviewText string  `json:"review_text" binding:"max=2000"`
}

type UpdateReviewRequest struct {
	Rating     *int    `json:"rating,omitempty"`
	ReviewText *string `json:"review_text,omitempty"`
}

// ======================
// Response DTOs
// ======================

type ReviewResponse struct {
	ID         string    `json:"id"`
	ModelID    string    `json:"model_id"`
	EmployerID string    `json:"employer_id"`
	CastingID  *string   `json:"casting_id,omitempty"`
	Rating     int       `json:"rating"`
	ReviewText string    `json:"review_text"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	Model    *ModelInfo    `json:"model,omitempty"`
	Employer *EmployerInfo `json:"employer,omitempty"`
	Casting  *CastingInfo  `json:"casting,omitempty"`
}

type ReviewListResponse struct {
	Reviews    []*ReviewResponse `json:"reviews"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

type RatingResponse struct {
	AverageRating   float64     `json:"average_rating"`
	TotalReviews    int64       `json:"total_reviews"`
	RatingBreakdown map[int]int `json:"rating_breakdown"`
	RecentReviews   int64       `json:"recent_reviews"`
}

type UserReviewStats struct {
	TotalReviews    int64   `json:"total_reviews"`
	AverageRating   float64 `json:"average_rating"`
	PositiveReviews int64   `json:"positive_reviews"` // 4-5 stars
	ResponseRate    float64 `json:"response_rate"`
	Ranking         int     `json:"ranking"` // Among peers
}

type ReviewSearchCriteria struct {
	UserID    string    `form:"user_id"`
	UserRole  string    `form:"user_role"`
	MinRating int       `form:"min_rating" binding:"min=1,max=5"`
	MaxRating int       `form:"max_rating" binding:"min=1,max=5"`
	DateFrom  time.Time `form:"date_from"`
	DateTo    time.Time `form:"date_to"`
	HasText   *bool     `form:"has_text"`
	Page      int       `form:"page" binding:"min=1"`
	PageSize  int       `form:"page_size" binding:"min=1,max=100"`
}

// ======================
// Expanded Info DTOs
// ======================

type ModelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	City string `json:"city"`
}

type EmployerInfo struct {
	ID          string `json:"id"`
	CompanyName string `json:"company_name"`
	City        string `json:"city"`
	IsVerified  bool   `json:"is_verified"`
}

type CastingInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	City  string `json:"city"`
}

const (
	ReviewStatusPending  = "pending"
	ReviewStatusApproved = "approved"
	ReviewStatusRejected = "rejected"
)
