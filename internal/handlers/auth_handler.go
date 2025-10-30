package handlers

import (
	"mwork_backend/internal/logger" // <-- Добавлен импорт
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	*BaseHandler // <-- 1. Встраиваем BaseHandler
	authService  services.AuthService
}

// 2. Обновляем конструктор
func NewAuthHandler(base *BaseHandler, authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		BaseHandler: base, // <-- 3. Сохраняем его
		authService: authService,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	// 4. Используем BindAndValidate_JSON
	if !h.BindAndValidate_JSON(c, &req) {
		return // Ошибка уже залоггирована и отправлена
	}

	// 5. Используем HandleServiceError
	err := h.authService.Register(&req)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful. Please check your email to verify your account.",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	response, err := h.authService.Login(&req)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	response, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if err := h.authService.Logout(req.RefreshToken); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully logged out",
	})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if err := h.authService.VerifyEmail(req.Token); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Email successfully verified",
	})
}

func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req dto.PasswordResetRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// Особый случай: мы не хотим возвращать ошибку пользователю,
	// но хотим ее залоггировать.
	if err := h.authService.RequestPasswordReset(req.Email); err != nil {
		// Не используем h.HandleServiceError, т.к. он отправит ответ.
		// Логгируем вручную.
		logger.CtxWarn(c.Request.Context(), "Password reset request failed (hiding from user)",
			"error", err.Error(),
			"email", req.Email,
		)
	}

	// Всегда возвращаем OK для безопасности
	c.JSON(http.StatusOK, gin.H{
		"message": "If the email exists, a password reset link has been sent",
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.PasswordResetConfirm
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if err := h.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password successfully reset",
	})
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	// 6. Используем GetAndAuthorizeUserID
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return // Ошибка уже отправлена
	}

	c.JSON(http.StatusOK, gin.H{
		"id":   userID,
		"role": c.GetString("role"), // c.GetString("role") по-прежнему безопасно
	})
}
