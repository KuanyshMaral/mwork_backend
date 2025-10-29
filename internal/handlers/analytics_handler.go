package handlers

import (
	"net/http"
	"strconv"
	"time"

	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analyticsService services.AnalyticsService
}

func NewAnalyticsHandler(analyticsService services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
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
	dateFrom, dateTo, err := h.parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	overview, err := h.analyticsService.GetPlatformOverview(dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, overview)
}

func (h *AnalyticsHandler) GetPlatformGrowthMetrics(c *gin.Context) {
	days := h.parseIntQuery(c, "days", 30)

	metrics, err := h.analyticsService.GetPlatformGrowthMetrics(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetUserAnalytics(c *gin.Context) {
	dateFrom, dateTo, err := h.parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	analytics, err := h.analyticsService.GetUserAnalytics(dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetUserAcquisitionMetrics(c *gin.Context) {
	dateFrom, dateTo, err := h.parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	metrics, err := h.analyticsService.GetUserAcquisitionMetrics(dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetUserRetentionMetrics(c *gin.Context) {
	days := h.parseIntQuery(c, "days", 30)

	metrics, err := h.analyticsService.GetUserRetentionMetrics(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetCastingAnalytics(c *gin.Context) {
	dateFrom, dateTo, err := h.parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	analytics, err := h.analyticsService.GetCastingAnalytics(dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetCastingPerformanceMetrics(c *gin.Context) {
	employerID := c.Param("employer_id")
	if employerID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "employer_id is required"})
		return
	}

	dateFrom, dateTo, err := h.parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	metrics, err := h.analyticsService.GetCastingPerformanceMetrics(employerID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetMatchingAnalytics(c *gin.Context) {
	dateFrom, dateTo, err := h.parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	analytics, err := h.analyticsService.GetMatchingAnalytics(dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetMatchingEfficiencyMetrics(c *gin.Context) {
	days := h.parseIntQuery(c, "days", 30)

	metrics, err := h.analyticsService.GetMatchingEfficiencyMetrics(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetFinancialAnalytics(c *gin.Context) {
	dateFrom, dateTo, err := h.parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	analytics, err := h.analyticsService.GetFinancialAnalytics(dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetGeographicAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.GetGeographicAnalytics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetCityPerformanceMetrics(c *gin.Context) {
	topN := h.parseIntQuery(c, "top_n", 10)

	metrics, err := h.analyticsService.GetCityPerformanceMetrics(topN)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetCategoryAnalytics(c *gin.Context) {
	analytics, err := h.analyticsService.GetCategoryAnalytics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *AnalyticsHandler) GetPopularCategories(c *gin.Context) {
	days := h.parseIntQuery(c, "days", 30)
	limit := h.parseIntQuery(c, "limit", 10)

	categories, err := h.analyticsService.GetPopularCategories(days, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, categories)
}

func (h *AnalyticsHandler) GetPerformanceMetrics(c *gin.Context) {
	dateFrom, dateTo, err := h.parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	metrics, err := h.analyticsService.GetPerformanceMetrics(dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetPlatformHealthMetrics(c *gin.Context) {
	metrics, err := h.analyticsService.GetPlatformHealthMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetRealTimeMetrics(c *gin.Context) {
	metrics, err := h.analyticsService.GetRealTimeMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetActiveUsersCount(c *gin.Context) {
	count, err := h.analyticsService.GetActiveUsersCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"active_users": count})
}

func (h *AnalyticsHandler) GetAdminDashboard(c *gin.Context) {
	adminID, exists := c.Get("user_id") // Предполагаем, что middleware добавляет user_id
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	dashboard, err := h.analyticsService.GetAdminDashboard(adminID.(string))
	if err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

func (h *AnalyticsHandler) GetSystemHealthMetrics(c *gin.Context) {
	metrics, err := h.analyticsService.GetSystemHealthMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GenerateCustomReport(c *gin.Context) {
	var req dto.CustomReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	report, err := h.analyticsService.GenerateCustomReport(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}

func (h *AnalyticsHandler) GetPredefinedReports(c *gin.Context) {
	reports, err := h.analyticsService.GetPredefinedReports()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, reports)
}

// Helper methods

func (h *AnalyticsHandler) parseDateRange(c *gin.Context) (time.Time, time.Time, error) {
	dateFromStr := c.Query("date_from")
	dateToStr := c.Query("date_to")

	// По умолчанию: последние 30 дней
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -30) // Изменено с -1 месяца на -30 дней для большей предсказуемости

	if dateFromStr != "" {
		parsed, err := time.Parse(time.RFC3339, dateFromStr) // Используем RFC3339 (YYYY-MM-DDTHH:MM:SSZ)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		dateFrom = parsed
	}

	if dateToStr != "" {
		parsed, err := time.Parse(time.RFC3339, dateToStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		dateTo = parsed
	}

	return dateFrom, dateTo, nil
}

func (h *AnalyticsHandler) parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// ErrorResponse является общей структурой для ошибок
type ErrorResponse struct {
	Error string `json:"error"`
}
