package dto

import (
	"time"

	"mwork_backend/internal/models"
)

// =======================
// Auth DTOs
// =======================

type LoginResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	User         *UserResponse `json:"user"`
}

// UserResponse содержит данные о пользователе
type UserResponse struct {
	ID         string            `json:"id"`
	Email      string            `json:"email"`
	Role       models.UserRole   `json:"role"`
	Status     models.UserStatus `json:"status"`
	IsVerified bool              `json:"is_verified"`
	Profile    interface{}       `json:"profile,omitempty"`
}

// =======================
// Admin DTOs
// =======================

// AdminUserFilter используется для фильтрации пользователей администратором
type AdminUserFilter struct {
	Role       models.UserRole   `form:"role" validate:"omitempty,is-user-role"`     // Custom rule
	Status     models.UserStatus `form:"status" validate:"omitempty,is-user-status"` // Custom rule (assumed)
	IsVerified *bool             `form:"is_verified"`
	DateFrom   *time.Time        `form:"date_from" validate:"omitempty"`
	DateTo     *time.Time        `form:"date_to" validate:"omitempty,gtefield=DateFrom"` // Ensures To > From
	Search     string            `form:"search"`
	Page       int               `form:"page" validate:"omitempty,min=1"`
	PageSize   int               `form:"page_size" validate:"omitempty,min=1,max=100"`
}

// =======================
// Profile DTOs
// =======================

// UpdateProfileRequest используется для обновления профиля пользователя
type UpdateProfileRequestUser struct {
	// Model fields
	Name           *string  `json:"name,omitempty" validate:"omitempty,min=2"`
	City           *string  `json:"city,omitempty"`
	Age            *int     `json:"age,omitempty" validate:"omitempty,min=16,max=70"`
	Height         *float64 `json:"height,omitempty" validate:"omitempty,min=100,max=250"`
	Weight         *float64 `json:"weight,omitempty" validate:"omitempty,min=30,max=200"`
	Gender         *string  `json:"gender,omitempty" validate:"omitempty,is-gender"` // Custom rule
	Experience     *string  `json:"experience,omitempty"`
	HourlyRate     *float64 `json:"hourly_rate,omitempty" validate:"omitempty,min=0"`
	Description    *string  `json:"description,omitempty" validate:"omitempty,max=2000"`
	ClothingSize   *string  `json:"clothing_size,omitempty"`
	ShoeSize       *string  `json:"shoe_size,omitempty"`
	BarterAccepted *bool    `json:"barter_accepted,omitempty"`
	IsPublic       *bool    `json:"is_public,omitempty"`

	// Employer fields
	CompanyName   *string `json:"company_name,omitempty" validate:"omitempty,min=2"`
	ContactPerson *string `json:"contact_person,omitempty"`
	Phone         *string `json:"phone,omitempty" validate:"omitempty,e164"` // e.g., +1234567890
	Website       *string `json:"website,omitempty" validate:"omitempty,url"`
	CompanyType   *string `json:"company_type,omitempty"`
}
