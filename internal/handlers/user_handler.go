package handlers

import (
	// "mwork_backend/internal/logger" // <-- Больше не нужен здесь
	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	// "mwork_backend/pkg/apperrors" // <-- Не используется напрямую (только в base_handler)
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	*BaseHandler
	userService services.UserService
	authService services.AuthService // <-- Оставляем для ChangePassword
}

func NewUserHandler(base *BaseHandler, userService services.UserService, authService services.AuthService) *UserHandler {
	return &UserHandler{
		BaseHandler: base,
		userService: userService,
		authService: authService, // <-- Оставляем
	}
}

// RegisterRoutes - удалена группа /auth
func (h *UserHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Группа /auth удалена, она теперь в AuthHandler

	profile := r.Group("/profile")
	profile.Use(middleware.AuthMiddleware())
	{
		profile.GET("", h.GetProfile)
		profile.PUT("", h.UpdateProfile)
		// ChangePassword - это действие над профилем, которое использует authService,
		// поэтому оно остается здесь.
		profile.POST("/password/change", h.ChangePassword)
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

// --- Auth handlers ---
// Все методы Register, Login, RefreshToken, Logout, VerifyEmail,
// RequestPasswordReset, ResetPassword УДАЛЕНЫ.
// Они теперь в auth_handler.go

// --- Profile handlers ---

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	db := h.GetDB(c)
	profile, err := h.userService.GetProfile(db, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.UpdateProfileRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	db := h.GetDB(c)
	if err := h.userService.UpdateProfile(db, userID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	db := h.GetDB(c)
	// Используем authService, как и раньше, но с передачей db
	if err := h.authService.ChangePassword(db, userID, req.CurrentPassword, req.NewPassword); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// --- Admin handlers ---

func (h *UserHandler) GetUsers(c *gin.Context) {
	var filter dto.AdminUserFilter
	if !h.BindAndValidate_Query(c, &filter) {
		return
	}

	filter.Page, filter.PageSize = ParsePagination(c)

	db := h.GetDB(c)
	users, total, err := h.userService.GetUsers(db, filter)
	if err != nil {
		h.HandleServiceError(c, err)
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
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	userID := c.Param("userId")

	var req struct {
		Status models.UserStatus `json:"status" binding:"required"`
	}
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	db := h.GetDB(c)
	if err := h.userService.UpdateUserStatus(db, adminID, userID, req.Status); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User status updated successfully"})
}

func (h *UserHandler) VerifyEmployer(c *gin.Context) {
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	employerID := c.Param("userId")

	db := h.GetDB(c)
	if err := h.userService.VerifyEmployer(db, adminID, employerID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employer verified successfully"})
}

func (h *UserHandler) GetRegistrationStats(c *gin.Context) {
	days := ParseQueryInt(c, "days", 30)

	db := h.GetDB(c)
	stats, err := h.userService.GetRegistrationStats(db, days)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}
