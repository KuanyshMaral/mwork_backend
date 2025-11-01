package handlers

import (
	"net/http"
	"strconv" // <-- Kept for ParseFloat

	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors" // <-- Added import

	"github.com/gin-gonic/gin"
)

type MatchingHandler struct {
	*BaseHandler    // <-- 1. Embed BaseHandler
	matchingService services.MatchingService
}

// 2. Update the constructor
func NewMatchingHandler(base *BaseHandler, matchingService services.MatchingService) *MatchingHandler {
	return &MatchingHandler{
		BaseHandler:     base, // <-- 3. Assign it
		matchingService: matchingService,
	}
}

// RegisterRoutes remains unchanged
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

// --- Core matching handlers ---

func (h *MatchingHandler) FindMatchingModels(c *gin.Context) {
	// 4. Use GetAndAuthorizeUserID
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	castingID := c.Param("castingId")

	// 5. Use ParseQueryInt
	limit := ParseQueryInt(c, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20 // Enforce bounds
	}

	minScore := 50.0
	if scoreParam := c.Query("min_score"); scoreParam != "" {
		// ParseFloat is not in BaseHandler, so we keep manual parsing
		if parsed, err := strconv.ParseFloat(scoreParam, 64); err == nil && parsed >= 0 && parsed <= 100 {
			minScore = parsed
		}
	}

	// ✅ DB: Используем h.GetDB(c)
	matches, err := h.matchingService.FindMatchingModels(h.GetDB(c), castingID, limit, minScore)
	if err != nil {
		// 6. Use HandleServiceError
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"matches": matches,
		"total":   len(matches),
	})
}

func (h *MatchingHandler) FindModelsByCriteria(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	var criteria dto.MatchCriteria
	// 7. Use BindAndValidate_JSON
	if !h.BindAndValidate_JSON(c, &criteria) {
		return
	}

	if criteria.Limit == 0 {
		criteria.Limit = 20
	}
	if criteria.MinScore == 0 {
		criteria.MinScore = 50.0
	}

	// ✅ DB: Используем h.GetDB(c)
	matches, err := h.matchingService.FindModelsByCriteria(h.GetDB(c), &criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"matches": matches,
		"total":   len(matches),
	})
}

func (h *MatchingHandler) GetModelCompatibility(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	modelID := c.Query("model_id")
	castingID := c.Query("casting_id")

	if modelID == "" || castingID == "" {
		// 8. Use appErrors
		apperrors.HandleError(c, apperrors.NewBadRequestError("model_id and casting_id are required"))
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	compatibility, err := h.matchingService.GetModelCompatibility(h.GetDB(c), modelID, castingID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, compatibility)
}

func (h *MatchingHandler) FindSimilarModels(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	modelID := c.Param("modelId")

	limit := ParseQueryInt(c, "limit", 10)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	// ✅ DB: Используем h.GetDB(c)
	similarModels, err := h.matchingService.FindSimilarModels(h.GetDB(c), modelID, limit)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"similar_models": similarModels,
		"total":          len(similarModels),
	})
}

func (h *MatchingHandler) GetMatchingWeights(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// ✅ DB: Веса глобальные, h.GetDB(c) не нужен
	weights, err := h.matchingService.GetMatchingWeights()
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, weights)
}

func (h *MatchingHandler) GetMatchingStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	castingID := c.Param("castingId")

	// ✅ DB: Используем h.GetDB(c)
	stats, err := h.matchingService.GetMatchingStats(h.GetDB(c), castingID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *MatchingHandler) GetModelMatchingStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	modelID := c.Param("modelId")

	// ✅ DB: Используем h.GetDB(c)
	stats, err := h.matchingService.GetModelMatchingStats(h.GetDB(c), modelID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// --- Admin handlers ---

func (h *MatchingHandler) UpdateMatchingWeights(c *gin.Context) {
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var weights dto.MatchingWeights
	if !h.BindAndValidate_JSON(c, &weights) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.matchingService.UpdateMatchingWeights(h.GetDB(c), adminID, &weights); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Matching weights updated successfully"})
}

func (h *MatchingHandler) GetPlatformMatchingStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	stats, err := h.matchingService.GetPlatformMatchingStats(h.GetDB(c))
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *MatchingHandler) RecalculateAllMatches(c *gin.Context) {
	adminID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.matchingService.RecalculateAllMatches(h.GetDB(c), adminID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Recalculation started successfully"})
}

func (h *MatchingHandler) GetMatchingLogs(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// 9. Use ParsePagination
	page, pageSize := ParsePagination(c)

	criteria := dto.MatchingLogCriteria{
		Page:     page,
		PageSize: pageSize,
	}

	// ✅ DB: Используем h.GetDB(c)
	logs, total, err := h.matchingService.GetMatchingLogs(h.GetDB(c), criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int(total) / pageSize
		if int(total)%pageSize != 0 {
			totalPages++
		}
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
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	var req struct {
		CastingIDs []string `json:"casting_ids" binding:"required"`
	}

	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	results, err := h.matchingService.BatchMatchModels(h.GetDB(c), req.CastingIDs)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(results),
	})
}
