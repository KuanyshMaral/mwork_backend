package handlers

import (
	"net/http"
	"strconv"
	"time"

	"mwork_backend/internal/appErrors" // <-- Добавлен импорт
	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type CastingHandler struct {
	castingService services.CastingService
}

func NewCastingHandler(castingService services.CastingService) *CastingHandler {
	return &CastingHandler{
		castingService: castingService,
	}
}

func (h *CastingHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Public routes
	public := r.Group("/castings")
	{
		public.GET("", h.SearchCastings)
		public.GET("/:castingId", h.GetCasting)
		public.GET("/active", h.GetActiveCastings)
		public.GET("/city/:city", h.GetCastingsByCity)
	}

	// Protected routes - Employer only
	castings := r.Group("/castings")
	castings.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleEmployer))
	{
		castings.POST("", h.CreateCasting)
		castings.GET("/my", h.GetMyCastings)
		castings.PUT("/:castingId", h.UpdateCasting)
		castings.DELETE("/:castingId", h.DeleteCasting)
		castings.PUT("/:castingId/status", h.UpdateCastingStatus)
		castings.GET("/:castingId/stats", h.GetCastingStatsForCasting)
		castings.GET("/stats/my", h.GetMyStats)
	}

	// Protected routes - Model matching
	matching := r.Group("/castings")
	matching.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleModel))
	{
		matching.GET("/matching", h.GetMatchingCastings)
	}

	// Admin routes
	admin := r.Group("/admin/castings")
	admin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleAdmin))
	{
		admin.POST("/close-expired", h.CloseExpiredCastings)
		admin.GET("/stats/platform", h.GetPlatformStats)
		admin.GET("/stats/matching", h.GetMatchingStats)
		admin.GET("/distribution/city", h.GetCastingDistributionByCity)
		admin.GET("/count/active", h.GetActiveCastingsCount)
		admin.GET("/categories/popular", h.GetPopularCategories)
	}
}

// Helper functions

func (h *CastingHandler) parseLimit(c *gin.Context) int {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	return limit
}

func (h *CastingHandler) parseDateRange(c *gin.Context) (time.Time, time.Time) {
	dateFrom := time.Now().AddDate(0, -1, 0) // Default: last month
	dateTo := time.Now()

	if dateFromParam := c.Query("date_from"); dateFromParam != "" {
		if parsed, err := time.Parse("2006-01-02", dateFromParam); err == nil {
			dateFrom = parsed
		}
	}

	if dateToParam := c.Query("date_to"); dateToParam != "" {
		if parsed, err := time.Parse("2006-01-02", dateToParam); err == nil {
			dateTo = parsed
		}
	}

	return dateFrom, dateTo
}

// --- УДАЛЕНЫ УСТАРЕВШИЕ handleServiceError и handleListResponse ---

// Public handlers

func (h *CastingHandler) SearchCastings(c *gin.Context) {
	var criteria dto.CastingSearchCriteria
	if err := c.ShouldBindQuery(&criteria); err != nil {
		appErrors.HandleValidationError(c, err) // <-- ИСПРАВЛЕНО
		return
	}

	// Set defaults
	if criteria.Page == 0 {
		criteria.Page = 1
	}
	if criteria.PageSize == 0 {
		criteria.PageSize = 20
	}

	castings, total, err := h.castingService.SearchCastings(criteria)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"castings": castings,
		"total":    total,
		"page":     criteria.Page,
		"pages":    (total + int64(criteria.PageSize) - 1) / int64(criteria.PageSize),
	})
}

func (h *CastingHandler) GetCasting(c *gin.Context) {
	castingID := c.Param("castingId")
	userID := ""

	// Get userID if authenticated
	if authUserID, exists := c.Get("userID"); exists {
		userID = authUserID.(string)
	}

	casting, err := h.castingService.GetCasting(castingID, userID)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, casting)
}

func (h *CastingHandler) GetActiveCastings(c *gin.Context) {
	limit := h.parseLimit(c)
	castings, err := h.castingService.GetActiveCastings(limit)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"castings": castings,
		"total":    len(castings),
	})
}

func (h *CastingHandler) GetCastingsByCity(c *gin.Context) {
	city := c.Param("city")
	limit := h.parseLimit(c)
	castings, err := h.castingService.GetCastingsByCity(city, limit)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"castings": castings,
		"total":    len(castings),
	})
}

// Employer handlers

func (h *CastingHandler) CreateCasting(c *gin.Context) {
	employerID := middleware.GetUserID(c)

	var req dto.CreateCastingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err) // <-- ИСПРАВЛЕНО
		return
	}

	// Set employer ID from authenticated user
	req.EmployerID = employerID

	err := h.castingService.CreateCasting(&req)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Casting created successfully"})
}

func (h *CastingHandler) GetMyCastings(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	requesterID := employerID // For employer, requester ID is the same as employer ID

	castings, err := h.castingService.GetEmployerCastings(employerID, requesterID)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"castings": castings,
		"total":    len(castings),
	})
}

func (h *CastingHandler) UpdateCasting(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	castingID := c.Param("castingId")

	var req dto.UpdateCastingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err) // <-- ИСПРАВЛЕНО
		return
	}

	if err := h.castingService.UpdateCasting(castingID, employerID, &req); err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Casting updated successfully"})
}

func (h *CastingHandler) DeleteCasting(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	castingID := c.Param("castingId")

	if err := h.castingService.DeleteCasting(castingID, employerID); err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Casting deleted successfully"})
}

func (h *CastingHandler) UpdateCastingStatus(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	castingID := c.Param("castingId")

	var req struct {
		Status models.CastingStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		appErrors.HandleValidationError(c, err) // <-- ИСПРАВЛЕНО
		return
	}

	if err := h.castingService.UpdateCastingStatus(castingID, employerID, req.Status); err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Casting status updated successfully"})
}

func (h *CastingHandler) GetCastingStatsForCasting(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	castingID := c.Param("castingId")

	stats, err := h.castingService.GetCastingStatsForCasting(castingID, employerID)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *CastingHandler) GetMyStats(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	requesterID := employerID // For employer, requester ID is the same as employer ID

	stats, err := h.castingService.GetCastingStats(employerID, requesterID)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Model handlers

func (h *CastingHandler) GetMatchingCastings(c *gin.Context) {
	modelID := middleware.GetUserID(c)
	limit := h.parseLimit(c)
	castings, err := h.castingService.FindMatchingCastings(modelID, limit)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"castings": castings,
		"total":    len(castings),
	})
}

// Admin handlers

func (h *CastingHandler) CloseExpiredCastings(c *gin.Context) {
	if err := h.castingService.CloseExpiredCastings(); err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expired castings closed successfully"})
}

func (h *CastingHandler) GetPlatformStats(c *gin.Context) {
	dateFrom, dateTo := h.parseDateRange(c)
	stats, err := h.castingService.GetPlatformCastingStats(dateFrom, dateTo)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *CastingHandler) GetMatchingStats(c *gin.Context) {
	dateFrom, dateTo := h.parseDateRange(c)
	stats, err := h.castingService.GetMatchingStats(dateFrom, dateTo)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *CastingHandler) GetCastingDistributionByCity(c *gin.Context) {
	distribution, err := h.castingService.GetCastingDistributionByCity()
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"distribution": distribution})
}

func (h *CastingHandler) GetActiveCastingsCount(c *gin.Context) {
	count, err := h.castingService.GetActiveCastingsCount()
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"active_castings_count": count})
}

func (h *CastingHandler) GetPopularCategories(c *gin.Context) {
	limit := h.parseLimit(c)
	categories, err := h.castingService.GetPopularCategories(limit)
	if err != nil {
		var appErr *appErrors.AppError // <-- ИСПРАВЛЕНО
		if appErrors.As(err, &appErr) {
			appErrors.HandleError(c, appErr)
		} else {
			appErrors.HandleError(c, appErrors.InternalError(err))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}
