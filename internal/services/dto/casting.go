package dto

import (
	"mwork_backend/internal/models"
	"time"
)

// --- Casting Requests ---

type CreateCastingRequest struct {
	EmployerID      string    `json:"employer_id" validate:"-"` // Устанавливается сервером
	Title           string    `json:"title" validate:"required,min=3,max=100"`
	Description     string    `json:"description" validate:"omitempty,max=5000"`
	PaymentMin      float64   `json:"payment_min" validate:"omitempty,min=0"`
	PaymentMax      float64   `json:"payment_max" validate:"omitempty,min=0,gtefield=PaymentMin"`
	CastingDate     time.Time `json:"casting_date"`
	CastingTime     string    `json:"casting_time"`
	Address         string    `json:"address"`
	City            string    `json:"city" validate:"required"`
	Categories      []string  `json:"categories"`
	Gender          string    `json:"gender" validate:"omitempty,is-gender"` // Кастомное правило
	AgeMin          *int      `json:"age_min" validate:"omitempty,min=0"`
	AgeMax          *int      `json:"age_max" validate:"omitempty,min=0,gtefield=AgeMin"`
	HeightMin       *float64  `json:"height_min" validate:"omitempty,min=0"`
	HeightMax       *float64  `json:"height_max" validate:"omitempty,min=0,gtefield=HeightMin"`
	WeightMin       *float64  `json:"weight_min" validate:"omitempty,min=0"`
	WeightMax       *float64  `json:"weight_max" validate:"omitempty,min=0,gtefield=WeightMin"`
	ClothingSize    string    `json:"clothing_size"`
	ShoeSize        string    `json:"shoe_size"`
	ExperienceLevel string    `json:"experience_level"`
	Languages       []string  `json:"languages"`
	JobType         string    `json:"job_type" validate:"omitempty,is-job-type"` // Кастомное правило
}

type UpdateCastingRequest struct {
	Title           *string    `json:"title,omitempty" validate:"omitempty,min=3,max=100"`
	Description     *string    `json:"description,omitempty" validate:"omitempty,max=5000"`
	PaymentMin      *float64   `json:"payment_min,omitempty" validate:"omitempty,min=0"`
	PaymentMax      *float64   `json:"payment_max,omitempty" validate:"omitempty,min=0,gtefield=PaymentMin"`
	CastingDate     *time.Time `json:"casting_date,omitempty"`
	CastingTime     *string    `json:"casting_time,omitempty"`
	Address         *string    `json:"address,omitempty"`
	City            *string    `json:"city,omitempty"`
	Categories      []string   `json:"categories,omitempty"`
	Gender          *string    `json:"gender,omitempty" validate:"omitempty,is-gender"`
	AgeMin          *int       `json:"age_min,omitempty"`
	AgeMax          *int       `json:"age_max,omitempty"`
	HeightMin       *float64   `json:"height_min,omitempty"`
	HeightMax       *float64   `json:"height_max,omitempty"`
	WeightMin       *float64   `json:"weight_min,omitempty"`
	WeightMax       *float64   `json:"weight_max,omitempty"`
	ClothingSize    *string    `json:"clothing_size,omitempty"`
	ShoeSize        *string    `json:"shoe_size,omitempty"`
	ExperienceLevel *string    `json:"experience_level,omitempty"`
	Languages       []string   `json:"languages,omitempty"`
	JobType         *string    `json:"job_type,omitempty" validate:"omitempty,is-job-type"`
}

type CreateResponseRequest struct {
	ModelID   string  `json:"model_id" validate:"-"`   // Устанавливается сервером
	CastingID string  `json:"casting_id" validate:"-"` // Устанавливается из URL
	Message   *string `json:"message" validate:"omitempty,max=1000"`
}

type UpdateResponseStatusRequest struct {
	Status models.ResponseStatus `json:"status" validate:"required,is-response-status"` // Кастомное правило
}

// --- Casting Responses ---

type CastingResponse struct {
	ID              string                `json:"id"`
	EmployerID      string                `json:"employer_id"`
	Title           string                `json:"title"`
	Description     string                `json:"description"`
	PaymentMin      float64               `json:"payment_min"`
	PaymentMax      float64               `json:"payment_max"`
	CastingDate     *time.Time            `json:"casting_date,omitempty"`
	CastingTime     *string               `json:"casting_time,omitempty"`
	Address         *string               `json:"address,omitempty"`
	City            string                `json:"city"`
	Categories      []string              `json:"categories"`
	Gender          string                `json:"gender"`
	AgeMin          *int                  `json:"age_min,omitempty"`
	AgeMax          *int                  `json:"age_max,omitempty"`
	HeightMin       *float64              `json:"height_min,omitempty"`
	HeightMax       *float64              `json:"height_max,omitempty"`
	WeightMin       *float64              `json:"weight_min,omitempty"`
	WeightMax       *float64              `json:"weight_max,omitempty"`
	ClothingSize    *string               `json:"clothing_size,omitempty"`
	ShoeSize        *string               `json:"shoe_size,omitempty"`
	ExperienceLevel *string               `json:"experience_level,omitempty"`
	Languages       []string              `json:"languages"`
	JobType         string                `json:"job_type"`
	Status          models.CastingStatus  `json:"status"`
	Views           int                   `json:"views"`
	Employer        interface{}           `json:"employer,omitempty"`
	Responses       []ResponseSummary     `json:"responses,omitempty"`
	Stats           *CastingStatsResponse `json:"stats,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

type ResponseSummary struct {
	ID        string                `json:"id"`
	ModelID   string                `json:"model_id"`
	ModelName string                `json:"model_name"`
	Message   *string               `json:"message,omitempty"`
	Status    models.ResponseStatus `json:"status"`
	CreatedAt time.Time             `json:"created_at"`
	Viewed    bool                  `json:"viewed"`
	Model     interface{}           `json:"model,omitempty"`
}

type CastingStatsResponse struct {
	TotalResponses    int64 `json:"total_responses"`
	PendingResponses  int64 `json:"pending_responses"`
	AcceptedResponses int64 `json:"accepted_responses"`
	RejectedResponses int64 `json:"rejected_responses"`
}

// --- Search Criteria ---

type CastingSearchCriteria struct {
	Query      string     `form:"query"`
	City       string     `form:"city"`
	Categories []string   `form:"categories[]"`
	Gender     string     `form:"gender"`
	MinAge     *int       `form:"min_age"`
	MaxAge     *int       `form:"max_age"`
	MinHeight  *int       `form:"min_height"`
	MaxHeight  *int       `form:"max_height"`
	MinSalary  *int       `form:"min_salary"`
	MaxSalary  *int       `form:"max_salary"`
	JobType    string     `form:"job_type" validate:"omitempty,is-job-type"`     // Кастомное правило
	Status     string     `form:"status" validate:"omitempty,is-casting-status"` // Кастомное правило
	EmployerID string     `form:"employer_id"`
	DateFrom   *time.Time `form:"date_from"`
	DateTo     *time.Time `form:"date_to"`
	Page       int        `form:"page" validate:"omitempty,min=1"`
	PageSize   int        `form:"page_size" validate:"omitempty,min=1,max=100"`
	SortBy     string     `form:"sort_by"`
	SortOrder  string     `form:"sort_order"`
}

type PlatformStatsResponse struct {
	TotalCastings    int64     `json:"total_castings"`
	ActiveCastings   int64     `json:"active_castings"`
	ClosedCastings   int64     `json:"closed_castings"`
	DraftCastings    int64     `json:"draft_castings"`
	TotalViews       int64     `json:"total_views"`
	TotalResponses   int64     `json:"total_responses"`
	AverageResponses float64   `json:"average_responses"`
	DateFrom         time.Time `json:"date_from"`
	DateTo           time.Time `json:"date_to"`
}

type PlatformCastingStatsResponse struct {
	TotalCastings   int64     `json:"totalCastings"`
	ActiveCastings  int64     `json:"activeCastings"`
	SuccessRate     float64   `json:"successRate"`
	AvgResponseRate float64   `json:"avgResponseRate"`
	AvgResponseTime float64   `json:"avgResponseTime"`
	DateFrom        time.Time `json:"dateFrom"`
	DateTo          time.Time `json:"dateTo"`
}

// MatchingStatsResponse - ответ со статистикой мэтчинга
type MatchingStatsResponse struct {
	TotalMatches    int64     `json:"totalMatches"`
	AvgMatchScore   float64   `json:"avgMatchScore"`
	AvgSatisfaction float64   `json:"avgSatisfaction"`
	MatchRate       float64   `json:"matchRate"`
	ResponseRate    float64   `json:"responseRate"`
	TimeToMatch     float64   `json:"timeToMatch"`
	DateFrom        time.Time `json:"dateFrom"`
	DateTo          time.Time `json:"dateTo"`
}

// CategoryCountResponse - ответ с количеством по категориям
type CategoryCountResponse struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}
