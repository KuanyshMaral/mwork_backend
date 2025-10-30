package handlers

import (
	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/logger" // <-- Добавлен импорт
	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"net/http"

	"github.com/gin-gonic/gin"
	// "mwork_backend/internal/middleware" // <-- Больше не нужен
)

type UserHandler struct {
	*BaseHandler // <-- 1. Встраиваем BaseHandler
	userService  services.UserService
	authService  services.AuthService
}

// 2. Обновляем конструктор
func NewUserHandler(base *BaseHandler, userService services.UserService, authService services.AuthService) *UserHandler {
	return &UserHandler{
		BaseHandler: base, // <-- 3. Сохраняем его
		userService: userService,
		authService: authService,
	}
}

// RegisterRoutes не требует изменений
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
	// AuthMiddleware все еще нужен здесь, чтобы gin.Context получил "userID"
	// (Если вы не вынесли его на более высокий уровень)
	profile.Use(middleware.AuthMiddleware())
	{
		profile.GET("", h.GetProfile)
		profile.PUT("", h.UpdateProfile)
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

func (h *UserHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	// 4. Используем BindAndValidate_JSON
	if !h.BindAndValidate_JSON(c, &req) {
		return // Ошибка уже залоггирована и отправлена
	}

	// 5. Используем HandleServiceError
	if err := h.authService.Register(&req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful. Please check your email to verify your account.",
	})
}

func (h *UserHandler) Login(c *gin.Context) {
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

func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
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

func (h *UserHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if err := h.authService.Logout(req.RefreshToken); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *UserHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		// Для простых проверок можно по-прежнему использовать appErrors
		appErrors.HandleError(c, appErrors.NewBadRequestError("Token is required"))
		return
	}

	if err := h.authService.VerifyEmail(token); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

func (h *UserHandler) RequestPasswordReset(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
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
		"message": "If an account exists with this email, a password reset link has been sent.",
	})
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if err := h.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// --- Profile handlers ---

func (h *UserHandler) GetProfile(c *gin.Context) {
	// 6. Используем GetAndAuthorizeUserID
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return // Ошибка уже отправлена
	}

	profile, err := h.userService.GetProfile(userID)
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

	var req dto.UpdateProfileRequestUser
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if err := h.userService.UpdateProfile(userID, &req); err != nil {
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

	if err := h.authService.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// --- Admin handlers ---

func (h *UserHandler) GetUsers(c *gin.Context) {
	var filter dto.AdminUserFilter
	// 7. Используем BindAndValidate_Query для фильтров
	if !h.BindAndValidate_Query(c, &filter) {
		return
	}

	// 8. Используем ParsePagination для пагинации
	filter.Page, filter.PageSize = ParsePagination(c)

	users, total, err := h.userService.GetUsers(filter)
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

	if err := h.userService.UpdateUserStatus(adminID, userID, req.Status); err != nil {
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

	if err := h.userService.VerifyEmployer(adminID, employerID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employer verified successfully"})
}

func (h *UserHandler) GetRegistrationStats(c *gin.Context) {
	// 9. Используем ParseQueryInt из base_handler
	days := ParseQueryInt(c, "days", 30)

	stats, err := h.userService.GetRegistrationStats(days)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}
