package dto

import (
	"time"

	"mwork_backend/internal/models"
)

// RegisterRequest - запрос регистрации
type RegisterRequest struct {
	Email    string          `json:"email" binding:"required,email"`
	Password string          `json:"password" binding:"required,min=8"`
	Role     models.UserRole `json:"role" binding:"required,oneof=model employer"`

	// Общие поля
	City string `json:"city" binding:"required"`

	// Поля для модели
	Name string `json:"name,omitempty" binding:"required_if=Role model"`

	// Поля для работодателя
	CompanyName string `json:"company_name,omitempty" binding:"required_if=Role employer"`
}

// LoginRequest - запрос входа
type LoginRequest struct {
	Email               string `json:"email" binding:"required,email"`
	Password            string `json:"password" binding:"required"`
	RequireVerification bool   `json:"require_verification"` // опционально
}

// RefreshTokenRequest - запрос обновления токена
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest - запрос выхода
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// VerifyEmailRequest - запрос подтверждения email
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// PasswordResetRequest - запрос сброса пароля
type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// PasswordResetConfirm - подтверждение сброса пароля
type PasswordResetConfirm struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// AuthResponse - ответ с токенами
type AuthResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	User         UserDTO `json:"user"`
}

// UserDTO - базовая информация о пользователе
type UserDTO struct {
	ID         string            `json:"id"`
	Email      string            `json:"email"`
	Role       models.UserRole   `json:"role"`
	Status     models.UserStatus `json:"status"`
	IsVerified bool              `json:"is_verified"`
	CreatedAt  time.Time         `json:"created_at"`
}
