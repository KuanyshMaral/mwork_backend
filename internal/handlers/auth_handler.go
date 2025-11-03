package handlers

import (
	"mwork_backend/internal/logger"
	"mwork_backend/internal/middleware"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	*BaseHandler
	authService services.AuthService
}

func NewAuthHandler(base *BaseHandler, authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		BaseHandler: base,
		authService: authService,
	}
}

// <-- ✅ ВОТ ИСПРАВЛЕНИЕ
//
// RegisterRoutes регистрирует все маршруты для аутентификации
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Создаем подгруппу /api/v1/auth
	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)
		auth.POST("/verify-email", h.VerifyEmail)
		auth.POST("/request-password-reset", h.RequestPasswordReset)
		auth.POST("/reset-password", h.ResetPassword)
	}

	admin := rg.Group("/admin")
	admin.Use(middleware.AuthMiddleware())  //
	admin.Use(middleware.AdminMiddleware()) //
	{
		// Роут: POST /api/v1/admin/users
		admin.POST("/users", h.AdminCreateUser) //
	}

	// Примечание: h.GetCurrentUser не регистрируется здесь.
	// Он, очевидно, требует аутентификации и должен быть частью
	// другой группы (например, /profile или /users/me),
	// которая уже защищена middleware.
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	db := h.GetDB(c)

	err := h.authService.Register(db, &req)
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

	db := h.GetDB(c)

	response, err := h.authService.Login(db, &req)
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

	db := h.GetDB(c)

	response, err := h.authService.RefreshToken(db, req.RefreshToken)
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

	db := h.GetDB(c)

	if err := h.authService.Logout(db, req.RefreshToken); err != nil {
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

	db := h.GetDB(c)

	if err := h.authService.VerifyEmail(db, req.Token); err != nil {
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

	db := h.GetDB(c)

	if err := h.authService.RequestPasswordReset(db, req.Email); err != nil {
		logger.CtxWarn(c.Request.Context(), "Password reset request failed (hiding from user)",
			"error", err.Error(),
			"email", req.Email,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "If the email exists, a password reset link has been sent",
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.PasswordResetConfirm
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	db := h.GetDB(c)

	if err := h.authService.ResetPassword(db, req.Token, req.NewPassword); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password successfully reset",
	})
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":   userID,
		"role": c.GetString("role"),
	})
}

func (h *AuthHandler) AdminCreateUser(c *gin.Context) {
	var req dto.AdminCreateUserRequest // 👈 Используем новый DTO
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	db := h.GetDB(c)

	user, err := h.authService.AdminCreateUser(db, &req)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// ❗️ Важно: Убираем хеш пароля из ответа
	user.PasswordHash = ""

	c.JSON(http.StatusCreated, user)
}
