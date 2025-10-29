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

	// ИСПРАВЛЕНО: 'Register' возвращает только 'error' и принимает указатель '&req'
	err := h.authService.Register(&req)
	if err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	// ИСПРАВЛЕНО: 'Register' не возвращает 'response', поэтому отправляем стандартное сообщение
	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful. Please check your email to verify your account.",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// ИСПРАВЛЕНО: 'Login' принимает указатель '&req'
	response, err := h.authService.Login(&req)
	if err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
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

	// Эта часть была корректной
	response, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
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

	// Эта часть была корректной
	if err := h.authService.Logout(req.RefreshToken); err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	// ПРИМЕЧАНИЕ: dto.MessageResponse не был предоставлен в dto,
	// но если он у вас есть, этот код корректен.
	// Если его нет, используйте gin.H
	c.JSON(http.StatusOK, gin.H{ // Заменено на gin.H на случай отсутствия dto.MessageResponse
		"message": "Successfully logged out",
	})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Эта часть была корректной
	if err := h.authService.VerifyEmail(req.Token); err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{ // Заменено на gin.H
		"message": "Email successfully verified",
	})
}

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req dto.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Эта часть была корректной
	_ = h.authService.RequestPasswordReset(req.Email)

	c.JSON(http.StatusOK, gin.H{ // Заменено на gin.H
		"message": "If the email exists, a password reset link has been sent",
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.PasswordResetConfirm
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Эта часть была корректtной
	if err := h.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{ // Заменено на gin.H
		"message": "Password successfully reset",
	})
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		appErrors.HandleError(c, appErrors.NewUnauthorizedError("User not authenticated"))
		return
	}

	// Эта часть была корректной
	c.JSON(http.StatusOK, gin.H{
		"id":   userID,
		"role": c.GetString("role"),
	})
}
