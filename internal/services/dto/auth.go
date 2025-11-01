package dto

import (
	"mwork_backend/internal/models"
	"time"
)

type RegisterRequest struct {
	Email       string          `json:"email" validate:"required,email"`
	Password    string          `json:"password" validate:"required,min=8"`
	Role        models.UserRole `json:"role" validate:"required,is-user-role"` // Используем кастомное правило
	City        string          `json:"city" validate:"required"`
	Name        string          `json:"name,omitempty" validate:"required_if=Role model"`
	CompanyName string          `json:"company_name,omitempty" validate:"required_if=Role employer"`
}

type LoginRequest struct {
	Email               string `json:"email" validate:"required,email"`
	Password            string `json:"password" validate:"required"`
	RequireVerification bool   `json:"require_verification"` // (опционально, нет валидации)
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type PasswordResetConfirm struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// AuthResponse - ЕДИНЫЙ ответ с токенами
type AuthResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	User         UserDTO `json:"user"`
}

// UserDTO - базовая информация о пользователе (для ответа при аутентификации)
type UserDTO struct {
	ID         string            `json:"id"`
	Email      string            `json:"email"`
	Role       models.UserRole   `json:"role"`
	Status     models.UserStatus `json:"status"`
	IsVerified bool              `json:"is_verified"`
	CreatedAt  time.Time         `json:"created_at"`
}
