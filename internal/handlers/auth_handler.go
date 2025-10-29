package handlers

import (
	"net/http"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService services.AuthService
}

func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	response, err := h.authService.Register(req)
	if err != nil {
		if err == services.ErrEmailAlreadyExists {
			appErrors.HandleError(c, appErrors.NewConflictError("Email already registered"))
			return
		}
		appErrors.HandleError(c, appErrors.NewInternalError("Registration failed"))
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	response, err := h.authService.Login(req)
	if err != nil {
		if err == services.ErrInvalidCredentials {
			appErrors.HandleError(c, appErrors.NewUnauthorizedError("Invalid email or password"))
			return
		}
		if err == services.ErrUserNotVerified {
			appErrors.HandleError(c, appErrors.NewForbiddenError("Email not verified"))
			return
		}
		appErrors.HandleError(c, appErrors.NewInternalError("Login failed"))
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	response, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		if err == services.ErrInvalidToken {
			appErrors.HandleError(c, appErrors.NewUnauthorizedError("Invalid or expired refresh token"))
			return
		}
		appErrors.HandleError(c, appErrors.NewInternalError("Token refresh failed"))
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	if err := h.authService.Logout(req.RefreshToken); err != nil {
		appErrors.HandleError(c, appErrors.NewInternalError("Logout failed"))
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Successfully logged out",
	})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	if err := h.authService.VerifyEmail(req.Token); err != nil {
		appErrors.HandleError(c, appErrors.NewNotFoundError("Invalid verification token"))
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Email successfully verified",
	})
}

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req dto.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Всегда возвращаем успех для безопасности (не раскрываем существование email)
	_ = h.authService.RequestPasswordReset(req.Email)

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "If the email exists, a password reset link has been sent",
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.PasswordResetConfirm
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	if err := h.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		appErrors.HandleError(c, appErrors.NewNotFoundError("Invalid or expired reset token"))
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Password successfully reset",
	})
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		appErrors.HandleError(c, appErrors.NewUnauthorizedError("User not authenticated"))
		return
	}

	// Здесь можно добавить логику получения полной информации о пользователе
	// Пока возвращаем базовые данные из токена
	c.JSON(http.StatusOK, gin.H{
		"id":   userID,
		"role": c.GetString("role"),
	})
}
