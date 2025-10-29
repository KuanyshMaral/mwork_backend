package handlers

import (
	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService services.UserService // <-- Изменено
	authService services.AuthService // <-- Добавлено
}

func NewUserHandler(userService services.UserService, authService services.AuthService) *UserHandler { // <-- Обновлен конструктор
	return &UserHandler{
		userService: userService,
		authService: authService,
	}
}

func (h *UserHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)
		auth.GET("/verify", h.VerifyEmail)
		auth.POST("/password/request-reset", h.RequestPasswordReset)
		auth.POST("/password/reset", h.ResetPassword)
	}

	profile := r.Group("/profile")
	profile.Use(middleware.AuthMiddleware())
	{
		profile.GET("", h.GetProfile)
		profile.PUT("", h.UpdateProfile)
		profile.POST("/password/change", h.ChangePassword) // <-- Этот метод относится к auth
	}

	admin := r.Group("/admin/users")
	admin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleAdmin))
	{
		admin.GET("", h.GetUsers)
		admin.PUT("/:userId/status", h.UpdateUserStatus)
		admin.PUT("/:userId/verify-employer", h.VerifyEmployer)
		admin.GET("/stats/registration", h.GetRegistrationStats)
	}
}

// Auth handlers

func (h *UserHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Используем authService
	if err := h.authService.Register(&req); err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful. Please check your email to verify your account.",
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Используем authService
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

func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Используем authService
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

func (h *UserHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Используем authService
	if err := h.authService.Logout(req.RefreshToken); err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *UserHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		appErrors.HandleError(c, appErrors.NewBadRequestError("Token is required"))
		return
	}

	// Используем authService
	if err := h.authService.VerifyEmail(token); err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

func (h *UserHandler) RequestPasswordReset(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleError(c, appErrors.NewBadRequestError("Invalid email").WithDetails(err.Error()))
		return
	}

	// Используем authService
	if err := h.authService.RequestPasswordReset(req.Email); err != nil {
		// Always return success to prevent email enumeration
		c.JSON(http.StatusOK, gin.H{
			"message": "If an account exists with this email, a password reset link has been sent.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "If an account exists with this email, a password reset link has been sent.",
	})
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Используем authService
	if err := h.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// Profile handlers

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	// Используем userService (корректно)
	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.UpdateProfileRequestUser
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Используем userService (корректно)
	if err := h.userService.UpdateProfile(userID, &req); err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Используем authService
	if err := h.authService.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// Admin handlers

func (h *UserHandler) GetUsers(c *gin.Context) {
	var filter dto.AdminUserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		appErrors.HandleError(c, appErrors.NewBadRequestError("Invalid query parameters").WithDetails(err.Error()))
		return
	}

	// Set defaults
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}

	// Используем userService (корректно)
	users, total, err := h.userService.GetUsers(filter)
	if err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  filter.Page,
		"pages": (total + int64(filter.PageSize) - 1) / int64(filter.PageSize),
	})
}

func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	userID := c.Param("userId")

	var req struct {
		Status models.UserStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err)
		return
	}

	// Используем userService (корректно)
	if err := h.userService.UpdateUserStatus(adminID, userID, req.Status); err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User status updated successfully"})
}

func (h *UserHandler) VerifyEmployer(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	employerID := c.Param("userId")

	// Используем userService (корректно)
	if err := h.userService.VerifyEmployer(adminID, employerID); err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employer verified successfully"})
}

func (h *UserHandler) GetRegistrationStats(c *gin.Context) {
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if parsedDays, err := parseIntParam(daysParam); err == nil {
			days = parsedDays
		}
	}

	// Используем userService (корректно)
	stats, err := h.userService.GetRegistrationStats(days)
	if err != nil {
		var appErr *appErrors.AppError
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, stats)
}
