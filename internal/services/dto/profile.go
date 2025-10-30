package dto

import "time"

// ==========================
// Create Requests
// ==========================

type CreateModelProfileRequest struct {
	UserID         string   `json:"user_id" validate:"-"` // Устанавливается сервером
	Name           string   `json:"name" validate:"required"`
	Age            int      `json:"age" validate:"required,min=16,max=70"`
	Height         float64  `json:"height" validate:"omitempty,min=100,max=250"`
	Weight         float64  `json:"weight" validate:"omitempty,min=30,max=200"`
	Gender         string   `json:"gender" validate:"omitempty,is-gender"` // Кастомное правило
	Experience     int      `json:"experience" validate:"omitempty,min=0"`
	HourlyRate     float64  `json:"hourly_rate" validate:"omitempty,min=0"`
	Description    string   `json:"description" validate:"omitempty,max=2000"`
	ClothingSize   string   `json:"clothing_size"`
	ShoeSize       string   `json:"shoe_size"`
	City           string   `json:"city" validate:"required"`
	Languages      []string `json:"languages"`
	Categories     []string `json:"categories"`
	BarterAccepted bool     `json:"barter_accepted"`
	IsPublic       bool     `json:"is_public"`
}

type CreateEmployerProfileRequest struct {
	UserID        string `json:"user_id" validate:"-"` // Устанавливается сервером
	CompanyName   string `json:"company_name" validate:"required"`
	ContactPerson string `json:"contact_person"`
	Phone         string `json:"phone" validate:"omitempty,e164"` // (Пример: можно добавить `e164` для тел. номеров)
	Website       string `json:"website" validate:"omitempty,url"`
	City          string `json:"city"`
	CompanyType   string `json:"company_type"`
	Description   string `json:"description" validate:"omitempty,max=2000"`
}

// ==========================
// Update Requests
// ==========================

type UpdateProfileRequest struct {
	Name           *string  `json:"name,omitempty" validate:"omitempty,min=2"`
	City           *string  `json:"city,omitempty"`
	Description    *string  `json:"description,omitempty,max=2000"`
	Age            *int     `json:"age,omitempty" validate:"omitempty,min=16,max=70"`
	Height         *float64 `json:"height,omitempty" validate:"omitempty,min=100,max=250"`
	Weight         *float64 `json:"weight,omitempty" validate:"omitempty,min=30,max=200"`
	Gender         *string  `json:"gender,omitempty" validate:"omitempty,is-gender"` // Кастомное правило
	Experience     *int     `json:"experience,omitempty" validate:"omitempty,min=0"`
	HourlyRate     *float64 `json:"hourly_rate,omitempty"`
	ClothingSize   *string  `json:"clothing_size,omitempty"`
	ShoeSize       *string  `json:"shoe_size,omitempty"`
	Languages      []string `json:"languages,omitempty"`
	Categories     []string `json:"categories,omitempty"`
	BarterAccepted *bool    `json:"barter_accepted,omitempty"`
	IsPublic       *bool    `json:"is_public,omitempty"`

	// Employer-specific fields
	CompanyName   *string `json:"company_name,omitempty" validate:"omitempty,min=2"`
	ContactPerson *string `json:"contact_person,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	Website       *string `json:"website,omitempty" validate:"omitempty,url"`
	CompanyType   *string `json:"company_type,omitempty"`
}

// ==========================
// Search Criteria
// ==========================

type ProfileSearchCriteria struct {
	Query         string   `form:"query"`
	City          string   `form:"city"`
	Categories    []string `form:"categories[]"`
	Gender        string   `form:"gender" validate:"omitempty,is-gender"` // Кастомное правило
	MinAge        *int     `form:"min_age" validate:"omitempty,min=0"`
	MaxAge        *int     `form:"max_age" validate:"omitempty,gtefield=MinAge"`
	MinHeight     *int     `form:"min_height" validate:"omitempty,min=0"`
	MaxHeight     *int     `form:"max_height" validate:"omitempty,gtefield=MinHeight"`
	MinWeight     *int     `form:"min_weight" validate:"omitempty,min=0"`
	MaxWeight     *int     `form:"max_weight" validate:"omitempty,gtefield=MinWeight"`
	MinPrice      *int     `form:"min_price" validate:"omitempty,min=0"`
	MaxPrice      *int     `form:"max_price" validate:"omitempty,gtefield=MinPrice"`
	MinExperience *int     `form:"min_experience" validate:"omitempty,min=0"`
	Languages     []string `form:"languages[]"`
	AcceptsBarter *bool    `form:"accepts_barter"`
	MinRating     *float64 `form:"min_rating" validate:"omitempty,min=0,max=5"`
	Page          int      `form:"page" validate:"omitempty,min=1"`
	PageSize      int      `form:"page_size" validate:"omitempty,min=1,max=100"`
	SortBy        string   `form:"sort_by"`
	SortOrder     string   `form:"sort_order" validate:"omitempty,oneof=asc desc"`
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
