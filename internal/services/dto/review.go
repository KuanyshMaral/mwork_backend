package dto

import "time"

// ======================
// Request DTOs
// ======================

type CreateReviewRequest struct {
	ModelID    string  `json:"model_id" validate:"required,uuid"`
	EmployerID string  `json:"employer_id" validate:"-"`                       // Set by server from auth
	CastingID  *string `json:"casting_id,omitempty" validate:"omitempty,uuid"` // omitempty, т.к. поле *string
	Rating     int     `json:"rating" validate:"required,min=1,max=5"`
	ReviewText string  `json:"review_text" validate:"omitempty,max=2000"`
}

type UpdateReviewRequest struct {
	Rating     *int    `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
	ReviewText *string `json:"review_text,omitempty" validate:"omitempty,max=2000"`
}

// ======================
// Search Criteria DTO (for Query Params)
// ======================
//
// !!!!!!!!!!!!!
// СТРУКТУРА ReviewSearchCriteria БЫЛА УДАЛЕНА.
// Она перенесена в dto/search.go
// !!!!!!!!!!!!!
//

// ======================
// Response DTOs (No validation needed)
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
