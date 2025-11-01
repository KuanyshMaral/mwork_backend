package handlers

import (
	"net/http"

	"mwork_backend/internal/middleware" // <-- Import middleware
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors" // <-- Added import

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	*BaseHandler     // <-- 1. Embed BaseHandler
	analyticsService services.AnalyticsService
}

// 2. Update the constructor
func NewAnalyticsHandler(base *BaseHandler, analyticsService services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		BaseHandler:      base, // <-- 3. Assign it
		analyticsService: analyticsService,
	}
}

// RegisterRoutes registers all analytics routes for Gin
func (h *AnalyticsHandler) RegisterRoutes(r *gin.RouterGroup) {
	// All analytics routes will be under /api/v1/analytics
	analytics := r.Group("/analytics")
	// ✅ DB: Все маршруты здесь должны быть защищены как минимум AuthMiddleware
	// (Предполагая, что аналитика не является публичной)
	analytics.Use(middleware.AuthMiddleware())
	{
		// Platform overview
		analytics.GET("/platform/overview", h.GetPlatformOverview)
		analytics.GET("/platform/growth", h.GetPlatformGrowthMetrics)
		analytics.GET("/platform/health", h.GetPlatformHealthMetrics)

		// User analytics
		analytics.GET("/users", h.GetUserAnalytics)
		analytics.GET("/users/acquisition", h.GetUserAcquisitionMetrics)
		analytics.GET("/users/retention", h.GetUserRetentionMetrics)
		analytics.GET("/users/active/count", h.GetActiveUsersCount)

		// Casting analytics
		analytics.GET("/castings", h.GetCastingAnalytics)
		analytics.GET("/castings/:employer_id/performance", h.GetCastingPerformanceMetrics)

		// Matching analytics
		analytics.GET("/matching", h.GetMatchingAnalytics)
		analytics.GET("/matching/efficiency", h.GetMatchingEfficiencyMetrics)

		// Financial analytics
		analytics.GET("/financial", h.GetFinancialAnalytics)

		// Geographic analytics
		analytics.GET("/geographic", h.GetGeographicAnalytics)
		analytics.GET("/geographic/cities", h.GetCityPerformanceMetrics)

		// Category analytics
		analytics.GET("/categories", h.GetCategoryAnalytics)
		analytics.GET("/categories/popular", h.GetPopularCategories)

		// Performance metrics
		analytics.GET("/performance", h.GetPerformanceMetrics)
		analytics.GET("/realtime", h.GetRealTimeMetrics)

		// System health
		analytics.GET("/system/health", h.GetSystemHealthMetrics)

		// Admin dashboard
		analytics.GET("/admin/dashboard", h.GetAdminDashboard)

		// Custom reports
		analytics.POST("/reports/custom", h.GenerateCustomReport)
		analytics.GET("/reports/predefined", h.GetPredefinedReports)
	}
}

// --- Handler Functions ---

func (h *AnalyticsHandler) GetPlatformOverview(c *gin.Context) {
	// ✅ DB: Получаем DB и Context
	db := h.GetDB(c)
	ctx := c.Request.Context()

	// 4. Use BaseHandler parsers
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30) // 30-day default
	if err != nil {
		h.HandleServiceError(c, err) // 5. Use HandleServiceError
		return
	}

	// ✅ DB: Передаем db и ctx
	overview, err := h.analyticsService.GetPlatformOverview(db, ctx, dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, overview)
}

func (h *AnalyticsHandler) GetPlatformGrowthMetrics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	days := ParseQueryInt(c, "days", 30) // 4. Use BaseHandler parser

	// ✅ DB: Передаем db и ctx
	metrics, err := h.analyticsService.GetPlatformGrowthMetrics(db, ctx, days)
	if err != nil {
		h.HandleServiceError(c, err) // 5. Use HandleServiceError
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetUserAnalytics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// ✅ DB: Передаем db и ctx
	analytics, err := h.analyticsService.GetUserAnalytics(db, ctx, dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetUserAcquisitionMetrics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// ✅ DB: Передаем db и ctx
	metrics, err := h.analyticsService.GetUserAcquisitionMetrics(db, ctx, dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetUserRetentionMetrics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	days := ParseQueryInt(c, "days", 30)

	// ✅ DB: Передаем db и ctx
	metrics, err := h.analyticsService.GetUserRetentionMetrics(db, ctx, days)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetCastingAnalytics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// ✅ DB: Передаем db и ctx
	analytics, err := h.analyticsService.GetCastingAnalytics(db, ctx, dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetCastingPerformanceMetrics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	employerID := c.Param("employer_id")
	if employerID == "" {
		apperrors.HandleError(c, apperrors.NewBadRequestError("employer_id is required"))
		return
	}

	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// ✅ DB: Передаем db и ctx
	metrics, err := h.analyticsService.GetCastingPerformanceMetrics(db, ctx, employerID, dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetMatchingAnalytics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// ✅ DB: Передаем db и ctx
	analytics, err := h.analyticsService.GetMatchingAnalytics(db, ctx, dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetMatchingEfficiencyMetrics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	days := ParseQueryInt(c, "days", 30)

	// ✅ DB: Передаем db и ctx
	metrics, err := h.analyticsService.GetMatchingEfficiencyMetrics(db, ctx, days)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetFinancialAnalytics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// ✅ DB: Передаем db и ctx
	analytics, err := h.analyticsService.GetFinancialAnalytics(db, ctx, dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetGeographicAnalytics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()

	// ✅ DB: Передаем db и ctx
	analytics, err := h.analyticsService.GetGeographicAnalytics(db, ctx)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetCityPerformanceMetrics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	topN := ParseQueryInt(c, "top_n", 10)

	// ✅ DB: Передаем db и ctx
	metrics, err := h.analyticsService.GetCityPerformanceMetrics(db, ctx, topN)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetCategoryAnalytics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()

	// ✅ DB: Передаем db и ctx
	analytics, err := h.analyticsService.GetCategoryAnalytics(db, ctx)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetPopularCategories(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	days := ParseQueryInt(c, "days", 30)
	limit := ParseQueryInt(c, "limit", 10)

	// ✅ DB: Передаем db и ctx
	categories, err := h.analyticsService.GetPopularCategories(db, ctx, days, limit)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, categories)
}

func (h *AnalyticsHandler) GetPerformanceMetrics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// ✅ DB: Передаем db и ctx
	metrics, err := h.analyticsService.GetPerformanceMetrics(db, ctx, dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetPlatformHealthMetrics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()

	// ✅ DB: Передаем db и ctx
	metrics, err := h.analyticsService.GetPlatformHealthMetrics(db, ctx)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetRealTimeMetrics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()

	// ✅ DB: Передаем db и ctx
	metrics, err := h.analyticsService.GetRealTimeMetrics(db, ctx)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetActiveUsersCount(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()

	// ✅ DB: Передаем db и ctx
	count, err := h.analyticsService.GetActiveUsersCount(db, ctx)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"active_users": count})
}

func (h *AnalyticsHandler) GetAdminDashboard(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	// 6. Use GetAndAuthorizeUserID
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB: Передаем db и ctx
	dashboard, err := h.analyticsService.GetAdminDashboard(db, ctx, adminID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

func (h *AnalyticsHandler) GetSystemHealthMetrics(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()

	// ✅ DB: Передаем db и ctx
	metrics, err := h.analyticsService.GetSystemHealthMetrics(db, ctx)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GenerateCustomReport(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	// 6. Use GetAndAuthorizeUserID (as per TODO)
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	var req dto.CustomReportRequest
	// 7. Use BindAndValidate_JSON
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Передаем db и ctx
	report, err := h.analyticsService.GenerateCustomReport(db, ctx, &req)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *AnalyticsHandler) GetPredefinedReports(c *gin.Context) {
	db := h.GetDB(c)
	ctx := c.Request.Context()
	// 6. Use GetAndAuthorizeUserID (as per TODO)
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// ✅ DB: Передаем db и ctx
	reports, err := h.analyticsService.GetPredefinedReports(db, ctx)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, reports)
}
