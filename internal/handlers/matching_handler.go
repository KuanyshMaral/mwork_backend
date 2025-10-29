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

type MatchingHandler struct {
	matchingService services.MatchingService
}

func NewMatchingHandler(matchingService services.MatchingService) *MatchingHandler {
	return &MatchingHandler{
		matchingService: matchingService,
	}
}

func (h *MatchingHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Protected routes - All authenticated users
	matching := r.Group("/matching")
	matching.Use(middleware.AuthMiddleware())
	{
		matching.GET("/castings/:castingId/models", h.FindMatchingModels)
		matching.POST("/models/search", h.FindModelsByCriteria)
		matching.GET("/compatibility", h.GetModelCompatibility)
		matching.GET("/models/:modelId/similar", h.FindSimilarModels)
		matching.GET("/weights", h.GetMatchingWeights)
		matching.GET("/castings/:castingId/stats", h.GetMatchingStats)
		matching.GET("/models/:modelId/stats", h.GetModelMatchingStats)
	}

	// Admin routes
	admin := r.Group("/admin/matching")
	admin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleAdmin))
	{
		admin.PUT("/weights", h.UpdateMatchingWeights)
		admin.GET("/stats/platform", h.GetPlatformMatchingStats)
		admin.POST("/recalculate", h.RecalculateAllMatches)
		admin.GET("/logs", h.GetMatchingLogs)
		admin.POST("/batch", h.BatchMatchModels)
	}
}

// Core matching handlers

func (h *MatchingHandler) FindMatchingModels(c *gin.Context) {
	castingID := c.Param("castingId")

	limit := 20
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	minScore := 50.0
	if scoreParam := c.Query("min_score"); scoreParam != "" {
		if parsed, err := strconv.ParseFloat(scoreParam, 64); err == nil && parsed >= 0 && parsed <= 100 {
			minScore = parsed
		}
	}

	matches, err := h.matchingService.FindMatchingModels(castingID, limit, minScore)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "casting not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"matches": matches,
		"total":   len(matches),
	})
}

func (h *MatchingHandler) FindModelsByCriteria(c *gin.Context) {
	var criteria dto.MatchCriteria
	if err := c.ShouldBindJSON(&criteria); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if criteria.Limit == 0 {
		criteria.Limit = 20
	}
	if criteria.MinScore == 0 {
		criteria.MinScore = 50.0
	}

	matches, err := h.matchingService.FindModelsByCriteria(&criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"matches": matches,
		"total":   len(matches),
	})
}

func (h *MatchingHandler) GetModelCompatibility(c *gin.Context) {
	modelID := c.Query("model_id")
	castingID := c.Query("casting_id")

	if modelID == "" || castingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_id and casting_id are required"})
		return
	}

	compatibility, err := h.matchingService.GetModelCompatibility(modelID, castingID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "model not found" || err.Error() == "casting not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, compatibility)
}

func (h *MatchingHandler) FindSimilarModels(c *gin.Context) {
	modelID := c.Param("modelId")

	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	similarModels, err := h.matchingService.FindSimilarModels(modelID, limit)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "model not found" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"similar_models": similarModels,
		"total":          len(similarModels),
	})
}

func (h *MatchingHandler) GetMatchingWeights(c *gin.Context) {
	weights, err := h.matchingService.GetMatchingWeights()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, weights)
}

func (h *MatchingHandler) GetMatchingStats(c *gin.Context) {
	castingID := c.Param("castingId")

	stats, err := h.matchingService.GetMatchingStats(castingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *MatchingHandler) GetModelMatchingStats(c *gin.Context) {
	modelID := c.Param("modelId")

	stats, err := h.matchingService.GetModelMatchingStats(modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Admin handlers

func (h *MatchingHandler) UpdateMatchingWeights(c *gin.Context) {
	adminID := middleware.GetUserID(c)

	var weights dto.MatchingWeights
	if err := c.ShouldBindJSON(&weights); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.matchingService.UpdateMatchingWeights(adminID, &weights); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "insufficient permissions" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "weights must sum to 1.0" {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Matching weights updated successfully"})
}

func (h *MatchingHandler) GetPlatformMatchingStats(c *gin.Context) {
	stats, err := h.matchingService.GetPlatformMatchingStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *MatchingHandler) RecalculateAllMatches(c *gin.Context) {
	adminID := middleware.GetUserID(c)

	if err := h.matchingService.RecalculateAllMatches(adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Recalculation started successfully"})
}

func (h *MatchingHandler) GetMatchingLogs(c *gin.Context) {
	page := 1
	pageSize := 50

	if pageParam := c.Query("page"); pageParam != "" {
		if parsed, err := strconv.Atoi(pageParam); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if pageSizeParam := c.Query("page_size"); pageSizeParam != "" {
		if parsed, err := strconv.Atoi(pageSizeParam); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	criteria := dto.MatchingLogCriteria{
		Page:     page,
		PageSize: pageSize,
	}

	logs, total, err := h.matchingService.GetMatchingLogs(criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":        logs,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

func (h *MatchingHandler) BatchMatchModels(c *gin.Context) {
	var req struct {
		CastingIDs []string `json:"casting_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	results, err := h.matchingService.BatchMatchModels(req.CastingIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(results),
	})
}
