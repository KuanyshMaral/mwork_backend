package handlers

import (
	"net/http"

	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	reviewService services.ReviewService
}

func NewReviewHandler(reviewService services.ReviewService) *ReviewHandler {
	return &ReviewHandler{
		reviewService: reviewService,
	}
}

func (h *ReviewHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Public routes
	public := r.Group("/reviews")
	{
		public.GET("/:reviewId", h.GetReview)
		public.GET("/models/:modelId", h.GetModelReviews)
		public.GET("/models/:modelId/stats", h.GetModelRatingStats)
		public.GET("/models/:modelId/summary", h.GetReviewSummary)
	}

	// Protected routes - Employer only
	reviews := r.Group("/reviews")
	reviews.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleEmployer))
	{
		reviews.POST("", h.CreateReview)
		reviews.GET("/my", h.GetMyReviews)
		reviews.PUT("/:reviewId", h.UpdateReview)
		reviews.DELETE("/:reviewId", h.DeleteReview)
		reviews.GET("/can-create", h.CanCreateReview)
	}

	// Admin routes
	admin := r.Group("/admin/reviews")
	admin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleAdmin))
	{
		admin.GET("/stats/platform", h.GetPlatformReviewStats)
		admin.GET("/recent", h.GetRecentReviews)
	}
}

// Public handlers

func (h *ReviewHandler) GetReview(c *gin.Context) {
	reviewID := c.Param("reviewId")

	review, err := h.reviewService.GetReview(reviewID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	c.JSON(http.StatusOK, review)
}

func (h *ReviewHandler) GetModelReviews(c *gin.Context) {
	modelID := c.Param("modelId")

	page := 1
	pageSize := 10

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

	reviews, err := h.reviewService.GetModelReviews(modelID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews": reviews.Reviews,
		"total":   reviews.Total,
		"page":    page,
		"pages":   reviews.TotalPages,
	})
}

func (h *ReviewHandler) GetModelRatingStats(c *gin.Context) {
	modelID := c.Param("modelId")

	stats, err := h.reviewService.GetModelRatingStats(modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *ReviewHandler) GetReviewSummary(c *gin.Context) {
	modelID := c.Param("modelId")

	summary, err := h.reviewService.GetReviewSummary(modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// Employer handlers

func (h *ReviewHandler) CreateReview(c *gin.Context) {
	employerID := middleware.GetUserID(c)

	var req dto.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Set employer ID from context
	req.EmployerID = employerID

	review, err := h.reviewService.CreateReview(employerID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "cannot create review for this casting" {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, review)
}

func (h *ReviewHandler) GetMyReviews(c *gin.Context) {
	employerID := middleware.GetUserID(c)

	page := 1
	pageSize := 10

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

	reviews, err := h.reviewService.GetEmployerReviews(employerID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews": reviews.Reviews,
		"total":   reviews.Total,
		"page":    page,
		"pages":   reviews.TotalPages,
	})
}

func (h *ReviewHandler) UpdateReview(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	reviewID := c.Param("reviewId")

	var req dto.UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.reviewService.UpdateReview(employerID, reviewID, &req); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Review updated successfully"})
}

func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	reviewID := c.Param("reviewId")

	if err := h.reviewService.DeleteReview(employerID, reviewID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Review deleted successfully"})
}

func (h *ReviewHandler) CanCreateReview(c *gin.Context) {
	employerID := middleware.GetUserID(c)
	modelID := c.Query("model_id")
	castingID := c.Query("casting_id")

	if modelID == "" || castingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_id and casting_id are required"})
		return
	}

	canCreate, err := h.reviewService.CanUserReview(employerID, modelID, castingID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"can_create": false,
			"reason":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"can_create": canCreate,
	})
}

// Admin handlers

func (h *ReviewHandler) GetPlatformReviewStats(c *gin.Context) {
	stats, err := h.reviewService.GetPlatformReviewStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *ReviewHandler) GetRecentReviews(c *gin.Context) {
	limit := 20
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	reviews, err := h.reviewService.GetRecentReviews(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews": reviews,
		"total":   len(reviews),
	})
}
