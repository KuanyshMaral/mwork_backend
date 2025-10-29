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
	Role       models.UserRole   `form:"role"`
	Status     models.UserStatus `form:"status"`
	IsVerified *bool             `form:"is_verified"`
	DateFrom   *time.Time        `form:"date_from"`
	DateTo     *time.Time        `form:"date_to"`
	Search     string            `form:"search"`
	Page       int               `form:"page" binding:"min=1"`
	PageSize   int               `form:"page_size" binding:"min=1,max=100"`
}

// =======================
// Profile DTOs
// =======================

// UpdateProfileRequest используется для обновления профиля пользователя
type UpdateProfileRequestUser struct {
	// Model fields
	Name           *string  `json:"name,omitempty"`
	City           *string  `json:"city,omitempty"`
	Age            *int     `json:"age,omitempty"`
	Height         *float64 `json:"height,omitempty"`
	Weight         *float64 `json:"weight,omitempty"`
	Gender         *string  `json:"gender,omitempty"`
	Experience     *string  `json:"experience,omitempty"`
	HourlyRate     *float64 `json:"hourly_rate,omitempty"`
	Description    *string  `json:"description,omitempty"`
	ClothingSize   *string  `json:"clothing_size,omitempty"`
	ShoeSize       *string  `json:"shoe_size,omitempty"`
	BarterAccepted *bool    `json:"barter_accepted,omitempty"`
	IsPublic       *bool    `json:"is_public,omitempty"`

	// Employer fields
	CompanyName   *string `json:"company_name,omitempty"`
	ContactPerson *string `json:"contact_person,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	Website       *string `json:"website,omitempty"`
	CompanyType   *string `json:"company_type,omitempty"`
}
