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
	*BaseHandler   // <-- 1. Embed BaseHandler
	castingService services.CastingService
}

// 2. Update the constructor
func NewCastingHandler(base *BaseHandler, castingService services.CastingService) *CastingHandler {
	return &CastingHandler{
		BaseHandler:    base, // <-- 3. Assign it
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

// 4. --- Helper functions removed ---

// --- Public handlers ---

func (h *CastingHandler) SearchCastings(c *gin.Context) {
	var criteria dto.CastingSearchCriteria
	// 5. Use BindAndValidate_Query
	if !h.BindAndValidate_Query(c, &criteria) {
		return
	}

	// 6. Use ParsePagination
	criteria.Page, criteria.PageSize = ParsePagination(c)

	castings, total, err := h.castingService.SearchCastings(criteria)
	if err != nil {
		// 7. Use HandleServiceError
		h.HandleServiceError(c, err)
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

	// Public route, so auth is optional
	if authUserID, exists := c.Get("userID"); exists {
		userID = authUserID.(string)
	}

	casting, err := h.castingService.GetCasting(castingID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, casting)
}

func (h *CastingHandler) GetActiveCastings(c *gin.Context) {
	// 8. Use ParseQueryInt
	limit := ParseQueryInt(c, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	castings, err := h.castingService.GetActiveCastings(limit)
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

	castings, err := h.castingService.GetCastingsByCity(city, limit)
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
	// 9. Use GetAndAuthorizeUserID
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.CreateCastingRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// Set employer ID from authenticated user
	req.EmployerID = employerID

	err := h.castingService.CreateCasting(&req)
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
	requesterID := employerID // For employer, requester ID is the same as employer ID

	castings, err := h.castingService.GetEmployerCastings(employerID, requesterID)
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

	if err := h.castingService.UpdateCasting(castingID, employerID, &req); err != nil {
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

	if err := h.castingService.DeleteCasting(castingID, employerID); err != nil {
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

	if err := h.castingService.UpdateCastingStatus(castingID, employerID, req.Status); err != nil {
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

	stats, err := h.castingService.GetCastingStatsForCasting(castingID, employerID)
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
	requesterID := employerID // For employer, requester ID is the same as employer ID

	stats, err := h.castingService.GetCastingStats(employerID, requesterID)
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

	castings, err := h.castingService.FindMatchingCastings(modelID, limit)
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

	if err := h.castingService.CloseExpiredCastings(); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expired castings closed successfully"})
}

func (h *CastingHandler) GetPlatformStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// 10. Use ParseQueryDateRange (30-day default)
	dateFrom, dateTo, err := ParseQueryDateRange(c, 30)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	stats, err := h.castingService.GetPlatformCastingStats(dateFrom, dateTo)
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

	stats, err := h.castingService.GetMatchingStats(dateFrom, dateTo)
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

	distribution, err := h.castingService.GetCastingDistributionByCity()
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

	count, err := h.castingService.GetActiveCastingsCount()
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

	categories, err := h.castingService.GetPopularCategories(limit)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}
