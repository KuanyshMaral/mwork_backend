package handlers

import (
	"net/http"

	"mwork_backend/internal/appErrors" // <-- Added import
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

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
		// TODO: Add authentication/authorization middleware for this route
		analytics.GET("/admin/dashboard", h.GetAdminDashboard)

		// Custom reports
		// TODO: Add authentication/authorization middleware for this route
		analytics.POST("/reports/custom", h.GenerateCustomReport)
		analytics.GET("/reports/predefined", h.GetPredefinedReports)
	}
}

// --- Handler Functions ---

func (h *AnalyticsHandler) GetPlatformOverview(c *gin.Context) {
	// 4. Use BaseHandler parsers
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30) // 30-day default
	if err != nil {
		h.HandleServiceError(c, err) // 5. Use HandleServiceError
		return
	}

	overview, err := h.analyticsService.GetPlatformOverview(dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, overview)
}

func (h *AnalyticsHandler) GetPlatformGrowthMetrics(c *gin.Context) {
	days := ParseQueryInt(c, "days", 30) // 4. Use BaseHandler parser

	metrics, err := h.analyticsService.GetPlatformGrowthMetrics(days)
	if err != nil {
		h.HandleServiceError(c, err) // 5. Use HandleServiceError
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetUserAnalytics(c *gin.Context) {
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	analytics, err := h.analyticsService.GetUserAnalytics(dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetUserAcquisitionMetrics(c *gin.Context) {
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	metrics, err := h.analyticsService.GetUserAcquisitionMetrics(dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetUserRetentionMetrics(c *gin.Context) {
	days := ParseQueryInt(c, "days", 30)

	metrics, err := h.analyticsService.GetUserRetentionMetrics(days)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetCastingAnalytics(c *gin.Context) {
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	analytics, err := h.analyticsService.GetCastingAnalytics(dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetCastingPerformanceMetrics(c *gin.Context) {
	employerID := c.Param("employer_id")
	if employerID == "" {
		appErrors.HandleError(c, appErrors.NewBadRequestError("employer_id is required"))
		return
	}

	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	metrics, err := h.analyticsService.GetCastingPerformanceMetrics(employerID, dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetMatchingAnalytics(c *gin.Context) {
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	analytics, err := h.analyticsService.GetMatchingAnalytics(dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetMatchingEfficiencyMetrics(c *gin.Context) {
	days := ParseQueryInt(c, "days", 30)

	metrics, err := h.analyticsService.GetMatchingEfficiencyMetrics(days)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetFinancialAnalytics(c *gin.Context) {
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	analytics, err := h.analyticsService.GetFinancialAnalytics(dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetGeographicAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.GetGeographicAnalytics()
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetCityPerformanceMetrics(c *gin.Context) {
	topN := ParseQueryInt(c, "top_n", 10)

	metrics, err := h.analyticsService.GetCityPerformanceMetrics(topN)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetCategoryAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.GetCategoryAnalytics()
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetPopularCategories(c *gin.Context) {
	days := ParseQueryInt(c, "days", 30)
	limit := ParseQueryInt(c, "limit", 10)

	categories, err := h.analyticsService.GetPopularCategories(days, limit)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, categories)
}

func (h *AnalyticsHandler) GetPerformanceMetrics(c *gin.Context) {
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	metrics, err := h.analyticsService.GetPerformanceMetrics(dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetPlatformHealthMetrics(c *gin.Context) {
	metrics, err := h.analyticsService.GetPlatformHealthMetrics()
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetRealTimeMetrics(c *gin.Context) {
	metrics, err := h.analyticsService.GetRealTimeMetrics()
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetActiveUsersCount(c *gin.Context) {
	count, err := h.analyticsService.GetActiveUsersCount()
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"active_users": count})
}

func (h *AnalyticsHandler) GetAdminDashboard(c *gin.Context) {
	// 6. Use GetAndAuthorizeUserID
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	dashboard, err := h.analyticsService.GetAdminDashboard(adminID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

func (h *AnalyticsHandler) GetSystemHealthMetrics(c *gin.Context) {
	metrics, err := h.analyticsService.GetSystemHealthMetrics()
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GenerateCustomReport(c *gin.Context) {
	// 6. Use GetAndAuthorizeUserID (as per TODO)
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	var req dto.CustomReportRequest
	// 7. Use BindAndValidate_JSON
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	report, err := h.analyticsService.GenerateCustomReport(&req)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *AnalyticsHandler) GetPredefinedReports(c *gin.Context) {
	// 6. Use GetAndAuthorizeUserID (as per TODO)
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	reports, err := h.analyticsService.GetPredefinedReports()
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, reports)
}
