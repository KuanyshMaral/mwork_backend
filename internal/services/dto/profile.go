package dto

import "time"

// ==========================
// Create Requests
// ==========================

type CreateModelProfileRequest struct {
	UserID         string   `json:"user_id" binding:"required"`
	Name           string   `json:"name" binding:"required"`
	Age            int      `json:"age" binding:"required,min=16,max=70"`
	Height         float64  `json:"height" binding:"min=100,max=250"`
	Weight         float64  `json:"weight" binding:"min=30,max=200"`
	Gender         string   `json:"gender"`
	Experience     int      `json:"experience"`
	HourlyRate     float64  `json:"hourly_rate" binding:"min=0"`
	Description    string   `json:"description"`
	ClothingSize   string   `json:"clothing_size"`
	ShoeSize       string   `json:"shoe_size"`
	City           string   `json:"city" binding:"required"`
	Languages      []string `json:"languages"`
	Categories     []string `json:"categories"`
	BarterAccepted bool     `json:"barter_accepted"`
	IsPublic       bool     `json:"is_public"`
}

type CreateEmployerProfileRequest struct {
	UserID        string `json:"user_id" binding:"required"`
	CompanyName   string `json:"company_name" binding:"required"`
	ContactPerson string `json:"contact_person"`
	Phone         string `json:"phone"`
	Website       string `json:"website"`
	City          string `json:"city"`
	CompanyType   string `json:"company_type"`
	Description   string `json:"description"`
}

// ==========================
// Update Requests
// ==========================

type UpdateProfileRequest struct {
	// Common fields
	Name        *string `json:"name,omitempty"`
	City        *string `json:"city,omitempty"`
	Description *string `json:"description,omitempty"`

	// Model-specific fields
	Age            *int     `json:"age,omitempty"`
	Height         *float64 `json:"height,omitempty"`
	Weight         *float64 `json:"weight,omitempty"`
	Gender         *string  `json:"gender,omitempty"`
	Experience     *int     `json:"experience,omitempty"`
	HourlyRate     *float64 `json:"hourly_rate,omitempty"`
	ClothingSize   *string  `json:"clothing_size,omitempty"`
	ShoeSize       *string  `json:"shoe_size,omitempty"`
	Languages      []string `json:"languages,omitempty"`
	Categories     []string `json:"categories,omitempty"`
	BarterAccepted *bool    `json:"barter_accepted,omitempty"`
	IsPublic       *bool    `json:"is_public,omitempty"`

	// Employer-specific fields
	CompanyName   *string `json:"company_name,omitempty"`
	ContactPerson *string `json:"contact_person,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	Website       *string `json:"website,omitempty"`
	CompanyType   *string `json:"company_type,omitempty"`
}

// ==========================
// Search Criteria
// ==========================

type ProfileSearchCriteria struct {
	Query         string   `form:"query"`
	City          string   `form:"city"`
	Categories    []string `form:"categories[]"`
	Gender        string   `form:"gender"`
	MinAge        *int     `form:"min_age"`
	MaxAge        *int     `form:"max_age"`
	MinHeight     *int     `form:"min_height"`
	MaxHeight     *int     `form:"max_height"`
	MinWeight     *int     `form:"min_weight"`
	MaxWeight     *int     `form:"max_weight"`
	MinPrice      *int     `form:"min_price"`
	MaxPrice      *int     `form:"max_price"`
	MinExperience *int     `form:"min_experience"`
	Languages     []string `form:"languages[]"`
	AcceptsBarter *bool    `form:"accepts_barter"`
	MinRating     *float64 `form:"min_rating"`
	Page          int      `form:"page" binding:"min=1"`
	PageSize      int      `form:"page_size" binding:"min=1,max=100"`
	SortBy        string   `form:"sort_by"`
	SortOrder     string   `form:"sort_order"`
}

// ==========================
// Responses & Stats
// ==========================

type ProfileResponse struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"` // "model" or "employer"
	UserID    string      `json:"user_id"`
	Data      interface{} `json:"data"`
	Stats     interface{} `json:"stats,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type ModelProfileStats struct {
	TotalViews      int64   `json:"total_views"`
	AverageRating   float64 `json:"average_rating"`
	TotalReviews    int64   `json:"total_reviews"`
	PortfolioItems  int64   `json:"portfolio_items"`
	ActiveResponses int64   `json:"active_responses"`
	CompletedJobs   int64   `json:"completed_jobs"`
}

type EmployerProfileStats struct {
	TotalCastings  int64   `json:"total_castings"`
	ActiveCastings int64   `json:"active_castings"`
	CompletedJobs  int64   `json:"completed_jobs"`
	TotalResponses int64   `json:"total_responses"`
	AverageRating  float64 `json:"average_rating"`
}
