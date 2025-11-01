package handlers

import (
	"net/http"

	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type CastingHandler struct {
	*BaseHandler
	castingService services.CastingService
}

func NewCastingHandler(base *BaseHandler, castingService services.CastingService) *CastingHandler {
	return &CastingHandler{
		BaseHandler:    base,
		castingService: castingService,
	}
}

// RegisterRoutes remains unchanged
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

// --- Public handlers ---

func (h *CastingHandler) SearchCastings(c *gin.Context) {
	// ▼▼▼ ИСПРАВЛЕНО ЗДЕСЬ ▼▼▼
	var criteria dto.SearchCastingsRequest
	// ▲▲▲ ИСПРАВЛЕНО ЗДЕСЬ ▲▲▲

	if !h.BindAndValidate_Query(c, &criteria) {
		return
	}

	criteria.Page, criteria.PageSize = ParsePagination(c)

	castings, total, err := h.castingService.SearchCastings(h.GetDB(c), criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"castings": castings,
		"total":    total,
		"page":     criteria.Page,
		"pages": func() int64 {
			if criteria.PageSize == 0 {
				return 0
			}
			return (total + int64(criteria.PageSize) - 1) / int64(criteria.PageSize)
		}(),
	})
}

func (h *CastingHandler) GetCasting(c *gin.Context) {
	castingID := c.Param("castingId")
	userID := ""

	if authUserID, exists := c.Get("userID"); exists {
		userID = authUserID.(string)
	}

	casting, err := h.castingService.GetCasting(h.GetDB(c), castingID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, casting)
}

func (h *CastingHandler) GetActiveCastings(c *gin.Context) {
	limit := ParseQueryInt(c, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	castings, err := h.castingService.GetActiveCastings(h.GetDB(c), limit)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"castings": castings,
		"total":    len(castings),
	})
}

func (h *CastingHandler) GetCastingsByCity(c *gin.Context) {
	city := c.Param("city")
	limit := ParseQueryInt(c, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	castings, err := h.castingService.GetCastingsByCity(h.GetDB(c), city, limit)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"castings": castings,
		"total":    len(castings),
	})
}

// --- Employer handlers ---

func (h *CastingHandler) CreateCasting(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.CreateCastingRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	req.EmployerID = employerID

	err := h.castingService.CreateCasting(h.GetDB(c), &req)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Casting created successfully"})
}

func (h *CastingHandler) GetMyCastings(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	requesterID := employerID

	castings, err := h.castingService.GetEmployerCastings(h.GetDB(c), employerID, requesterID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"castings": castings,
		"total":    len(castings),
	})
}

func (h *CastingHandler) UpdateCasting(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	castingID := c.Param("castingId")

	var req dto.UpdateCastingRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if err := h.castingService.UpdateCasting(h.GetDB(c), castingID, employerID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Casting updated successfully"})
}

func (h *CastingHandler) DeleteCasting(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	castingID := c.Param("castingId")

	if err := h.castingService.DeleteCasting(h.GetDB(c), castingID, employerID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Casting deleted successfully"})
}

func (h *CastingHandler) UpdateCastingStatus(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	castingID := c.Param("castingId")

	var req struct {
		Status models.CastingStatus `json:"status" binding:"required"`
	}
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if err := h.castingService.UpdateCastingStatus(h.GetDB(c), castingID, employerID, req.Status); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Casting status updated successfully"})
}

func (h *CastingHandler) GetCastingStatsForCasting(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	castingID := c.Param("castingId")

	stats, err := h.castingService.GetCastingStatsForCasting(h.GetDB(c), castingID, employerID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *CastingHandler) GetMyStats(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	requesterID := employerID

	stats, err := h.castingService.GetCastingStats(h.GetDB(c), employerID, requesterID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// --- Model handlers ---

func (h *CastingHandler) GetMatchingCastings(c *gin.Context) {
	modelID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	limit := ParseQueryInt(c, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	castings, err := h.castingService.FindMatchingCastings(h.GetDB(c), modelID, limit)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"castings": castings,
		"total":    len(castings),
	})
}

// --- Admin handlers ---

func (h *CastingHandler) CloseExpiredCastings(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	if err := h.castingService.CloseExpiredCastings(h.GetDB(c)); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expired castings closed successfully"})
}

func (h *CastingHandler) GetPlatformStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	stats, err := h.castingService.GetPlatformCastingStats(h.GetDB(c), dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *CastingHandler) GetMatchingStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	stats, err := h.castingService.GetMatchingStats(h.GetDB(c), dateFrom, dateTo)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *CastingHandler) GetCastingDistributionByCity(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	distribution, err := h.castingService.GetCastingDistributionByCity(h.GetDB(c))
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"distribution": distribution})
}

func (h *CastingHandler) GetActiveCastingsCount(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	count, err := h.castingService.GetActiveCastingsCount(h.GetDB(c))
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"active_castings_count": count})
}

func (h *CastingHandler) GetPopularCategories(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	limit := ParseQueryInt(c, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	categories, err := h.castingService.GetPopularCategories(h.GetDB(c), limit)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}
