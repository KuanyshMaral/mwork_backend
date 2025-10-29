package handlers

import (
	"net/http"
	"strconv"

	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	searchService services.SearchService
}

func NewSearchHandler(searchService services.SearchService) *SearchHandler {
	return &SearchHandler{
		searchService: searchService,
	}
}

func (h *SearchHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Public search routes
	search := r.Group("/search")
	{
		search.POST("/castings", h.SearchCastings)
		search.POST("/castings/advanced", h.SearchCastingsAdvanced)
		search.GET("/castings/suggestions", h.GetCastingSearchSuggestions)

		search.POST("/models", h.SearchModels)
		search.POST("/models/advanced", h.SearchModelsAdvanced)
		search.GET("/models/suggestions", h.GetModelSearchSuggestions)

		search.POST("/employers", h.SearchEmployers)

		search.POST("/unified", h.UnifiedSearch)
		search.GET("/autocomplete", h.GetSearchAutoComplete)

		search.GET("/popular", h.GetPopularSearches)
		search.GET("/trends", h.GetSearchTrends)
	}

	// Protected search routes
	searchAuth := r.Group("/search")
	searchAuth.Use(middleware.AuthMiddleware())
	{
		searchAuth.GET("/history", h.GetSearchHistory)
		searchAuth.DELETE("/history", h.ClearSearchHistory)
	}

	// Admin routes
	admin := r.Group("/admin/search")
	admin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleAdmin))
	{
		admin.GET("/analytics", h.GetSearchAnalytics)
		admin.POST("/reindex", h.ReindexSearchData)
	}
}

// Casting search handlers

func (h *SearchHandler) SearchCastings(c *gin.Context) {
	var req dto.SearchCastingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	response, err := h.searchService.SearchCastings(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SearchHandler) SearchCastingsAdvanced(c *gin.Context) {
	var req dto.AdvancedCastingSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	response, err := h.searchService.SearchCastingsAdvanced(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SearchHandler) GetCastingSearchSuggestions(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter is required"})
		return
	}

	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	suggestions, err := h.searchService.GetCastingSearchSuggestions(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions": suggestions,
		"total":       len(suggestions),
	})
}

// Model search handlers

func (h *SearchHandler) SearchModels(c *gin.Context) {
	var req dto.SearchModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	response, err := h.searchService.SearchModels(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SearchHandler) SearchModelsAdvanced(c *gin.Context) {
	var req dto.AdvancedModelSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	response, err := h.searchService.SearchModelsAdvanced(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SearchHandler) GetModelSearchSuggestions(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter is required"})
		return
	}

	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	suggestions, err := h.searchService.GetModelSearchSuggestions(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions": suggestions,
		"total":       len(suggestions),
	})
}

// Employer search handlers

func (h *SearchHandler) SearchEmployers(c *gin.Context) {
	var req dto.SearchEmployersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	response, err := h.searchService.SearchEmployers(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Unified search handlers

func (h *SearchHandler) UnifiedSearch(c *gin.Context) {
	var req dto.UnifiedSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 30
	}

	response, err := h.searchService.UnifiedSearch(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SearchHandler) GetSearchAutoComplete(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter is required"})
		return
	}

	response, err := h.searchService.GetSearchAutoComplete(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Search analytics and features

func (h *SearchHandler) GetPopularSearches(c *gin.Context) {
	limit := 20
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	searches, err := h.searchService.GetPopularSearches(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"popular_searches": searches,
		"total":            len(searches),
	})
}

func (h *SearchHandler) GetSearchTrends(c *gin.Context) {
	days := 7
	if daysParam := c.Query("days"); daysParam != "" {
		if parsed, err := strconv.Atoi(daysParam); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	trends, err := h.searchService.GetSearchTrends(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trends)
}

func (h *SearchHandler) GetSearchHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)

	limit := 20
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	history, err := h.searchService.GetSearchHistory(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   len(history),
	})
}

func (h *SearchHandler) ClearSearchHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if err := h.searchService.ClearSearchHistory(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Search history cleared successfully"})
}

// Admin handlers

func (h *SearchHandler) GetSearchAnalytics(c *gin.Context) {
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if parsed, err := strconv.Atoi(daysParam); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	analytics, err := h.searchService.GetSearchAnalytics(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *SearchHandler) ReindexSearchData(c *gin.Context) {
	adminID := middleware.GetUserID(c)

	if err := h.searchService.ReindexSearchData(adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Search data reindexing started successfully"})
}
