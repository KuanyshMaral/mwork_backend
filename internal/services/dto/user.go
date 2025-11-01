package dto

import (
	"time"

	"mwork_backend/internal/models"
)

// =======================
// Auth DTOs
// =======================
//
// !!!!!!!!!!!!!
// СТРУКТУРА LoginResponse БЫЛА УДАЛЕНА.
// Вместо нее используется AuthResponse из dto/auth.go
// !!!!!!!!!!!!!
//

// UserResponse содержит ПОЛНЫЕ данные о пользователе (в отличие от UserDTO)
// Используется для эндпоинтов типа /users/me
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
//
// !!!!!!!!!!!!!
// СТРУКТУРА UpdateProfileRequestUser БЫЛА УДАЛЕНА.
// Вместо нее используется UpdateProfileRequest из dto/profile.go
// !!!!!!!!!!!!!
//
