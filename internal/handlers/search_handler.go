package handlers

import (
	"net/http"

	"mwork_backend/pkg/apperrors" // <-- Добавлен импорт

	"mwork_backend/internal/middleware" // <-- Все еще нужен для RegisterRoutes
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	*BaseHandler
	searchService services.SearchService
}

func NewSearchHandler(base *BaseHandler, searchService services.SearchService) *SearchHandler {
	return &SearchHandler{
		BaseHandler:   base,
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

// --- Casting search handlers ---

func (h *SearchHandler) SearchCastings(c *gin.Context) {
	var req dto.SearchCastingsRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	response, err := h.searchService.SearchCastings(h.GetDB(c), &req)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SearchHandler) SearchCastingsAdvanced(c *gin.Context) {
	var req dto.AdvancedCastingSearchRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	response, err := h.searchService.SearchCastingsAdvanced(h.GetDB(c), &req)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SearchHandler) GetCastingSearchSuggestions(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		apperrors.HandleError(c, apperrors.NewBadRequestError("query parameter is required"))
		return
	}

	limit := ParseQueryInt(c, "limit", 10)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	suggestions, err := h.searchService.GetCastingSearchSuggestions(h.GetDB(c), query, limit)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions": suggestions,
		"total":       len(suggestions),
	})
}

// --- Model search handlers ---

func (h *SearchHandler) SearchModels(c *gin.Context) {
	var req dto.SearchModelsRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	response, err := h.searchService.SearchModels(h.GetDB(c), &req)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SearchHandler) SearchModelsAdvanced(c *gin.Context) {
	var req dto.AdvancedModelSearchRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	response, err := h.searchService.SearchModelsAdvanced(h.GetDB(c), &req)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SearchHandler) GetModelSearchSuggestions(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		apperrors.HandleError(c, apperrors.NewBadRequestError("query parameter is required"))
		return
	}

	limit := ParseQueryInt(c, "limit", 10)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	suggestions, err := h.searchService.GetModelSearchSuggestions(h.GetDB(c), query, limit)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions": suggestions,
		"total":       len(suggestions),
	})
}

// --- Employer search handlers ---

func (h *SearchHandler) SearchEmployers(c *gin.Context) {
	var req dto.SearchEmployersRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	response, err := h.searchService.SearchEmployers(h.GetDB(c), &req)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// --- Unified search handlers ---

func (h *SearchHandler) UnifiedSearch(c *gin.Context) {
	var req dto.UnifiedSearchRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 30
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	response, err := h.searchService.UnifiedSearch(h.GetDB(c), &req)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *SearchHandler) GetSearchAutoComplete(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		apperrors.HandleError(c, apperrors.NewBadRequestError("query parameter is required"))
		return
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	response, err := h.searchService.GetSearchAutoComplete(h.GetDB(c), query)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// --- Search analytics and features ---

func (h *SearchHandler) GetPopularSearches(c *gin.Context) {
	limit := ParseQueryInt(c, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	searches, err := h.searchService.GetPopularSearches(h.GetDB(c), limit)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"popular_searches": searches,
		"total":            len(searches),
	})
}

func (h *SearchHandler) GetSearchTrends(c *gin.Context) {
	days := ParseQueryInt(c, "days", 7)
	if days <= 0 || days > 365 {
		days = 7
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	trends, err := h.searchService.GetSearchTrends(h.GetDB(c), days)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, trends)
}

func (h *SearchHandler) GetSearchHistory(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	limit := ParseQueryInt(c, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	history, err := h.searchService.GetSearchHistory(h.GetDB(c), userID, limit)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   len(history),
	})
}

func (h *SearchHandler) ClearSearchHistory(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ▼▼▼ ИZМЕНЕНО ▼▼▼
	if err := h.searchService.ClearSearchHistory(h.GetDB(c), userID); err != nil {
		// ▲▲▲ ИЗМЕНЕНО ▲▲▲
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Search history cleared successfully"})
}

// --- Admin handlers ---

func (h *SearchHandler) GetSearchAnalytics(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	days := ParseQueryInt(c, "days", 30)
	if days <= 0 || days > 365 {
		days = 30
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	analytics, err := h.searchService.GetSearchAnalytics(h.GetDB(c), days)
	// ▲▲▲ ИЗМЕНЕНО ▲▲▲
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (h *SearchHandler) ReindexSearchData(c *gin.Context) {
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ▼▼▼ ИЗМЕНЕНО ▼▼▼
	if err := h.searchService.ReindexSearchData(h.GetDB(c), adminID); err != nil {
		// ▲▲▲ ИЗМЕНЕНО ▲▲▲
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Search data reindexing started successfully"})
}
