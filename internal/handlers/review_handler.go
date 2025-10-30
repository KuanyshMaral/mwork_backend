package handlers

import (
	"net/http"

	"mwork_backend/internal/appErrors"  // <-- Добавлен импорт
	"mwork_backend/internal/middleware" // <-- Все еще нужен для RegisterRoutes
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	// "strconv" // <-- Больше не нужен

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	*BaseHandler  // <-- 1. Встраиваем BaseHandler
	reviewService services.ReviewService
}

// 2. Обновляем конструктор
func NewReviewHandler(base *BaseHandler, reviewService services.ReviewService) *ReviewHandler {
	return &ReviewHandler{
		BaseHandler:   base, // <-- 3. Сохраняем его
		reviewService: reviewService,
	}
}

// RegisterRoutes не требует изменений
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

// --- Public handlers ---

func (h *ReviewHandler) GetReview(c *gin.Context) {
	reviewID := c.Param("reviewId")

	review, err := h.reviewService.GetReview(reviewID)
	if err != nil {
		h.HandleServiceError(c, err) // <-- 4. Используем HandleServiceError
		return
	}

	c.JSON(http.StatusOK, review)
}

func (h *ReviewHandler) GetModelReviews(c *gin.Context) {
	modelID := c.Param("modelId")

	// 5. Используем ParsePagination
	page, pageSize := ParsePagination(c)

	reviews, err := h.reviewService.GetModelReviews(modelID, page, pageSize)
	if err != nil {
		h.HandleServiceError(c, err)
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
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *ReviewHandler) GetReviewSummary(c *gin.Context) {
	modelID := c.Param("modelId")

	summary, err := h.reviewService.GetReviewSummary(modelID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, summary)
}

// --- Employer handlers ---

func (h *ReviewHandler) CreateReview(c *gin.Context) {
	// 6. Используем GetAndAuthorizeUserID
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.CreateReviewRequest
	// 7. Используем BindAndValidate_JSON
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// Set employer ID from context
	req.EmployerID = employerID

	review, err := h.reviewService.CreateReview(employerID, &req)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, review)
}

func (h *ReviewHandler) GetMyReviews(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	page, pageSize := ParsePagination(c)

	reviews, err := h.reviewService.GetEmployerReviews(employerID, page, pageSize)
	if err != nil {
		h.HandleServiceError(c, err)
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
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	reviewID := c.Param("reviewId")

	var req dto.UpdateReviewRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	if err := h.reviewService.UpdateReview(employerID, reviewID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Review updated successfully"})
}

func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	reviewID := c.Param("reviewId")

	if err := h.reviewService.DeleteReview(employerID, reviewID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Review deleted successfully"})
}

func (h *ReviewHandler) CanCreateReview(c *gin.Context) {
	employerID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	modelID := c.Query("model_id")
	castingID := c.Query("casting_id")

	if modelID == "" || castingID == "" {
		// 8. Используем appErrors
		appErrors.HandleError(c, appErrors.NewBadRequestError("model_id and casting_id are required"))
		return
	}

	canCreate, err := h.reviewService.CanUserReview(employerID, modelID, castingID)
	if err != nil {
		// Особый случай: отправляем ошибку как часть ответа, а не как HTTP-ошибку
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

// --- Admin handlers ---

func (h *ReviewHandler) GetPlatformReviewStats(c *gin.Context) {
	// 9. Добавляем проверку авторизации для админских ручек
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	stats, err := h.reviewService.GetPlatformReviewStats()
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *ReviewHandler) GetRecentReviews(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}

	// 10. Используем ParseQueryInt
	limit := ParseQueryInt(c, "limit", 20)
	if limit <= 0 {
		limit = 20
	}

	reviews, err := h.reviewService.GetRecentReviews(limit)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews": reviews,
		"total":   len(reviews),
	})
}
