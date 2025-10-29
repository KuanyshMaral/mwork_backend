package handlers

import (
	"net/http"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type PortfolioHandler struct {
	portfolioService services.PortfolioService
}

func NewPortfolioHandler(portfolioService services.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioService: portfolioService,
	}
}

func (h *PortfolioHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Public routes
	public := r.Group("/portfolio")
	{
		public.GET("/:itemId", h.GetPortfolioItem)
		public.GET("/model/:modelId", h.GetModelPortfolio)
		public.GET("/featured", h.GetFeaturedPortfolio)
		public.GET("/recent", h.GetRecentPortfolio)
	}

	// Protected routes
	portfolio := r.Group("/portfolio")
	portfolio.Use(middleware.AuthMiddleware())
	{
		portfolio.POST("", h.CreatePortfolioItem)
		portfolio.PUT("/:itemId", h.UpdatePortfolioItem)
		portfolio.DELETE("/:itemId", h.DeletePortfolioItem)
		portfolio.PUT("/reorder", h.UpdatePortfolioOrder)
		portfolio.PUT("/:itemId/visibility", h.TogglePortfolioVisibility)
		portfolio.GET("/stats/:modelId", h.GetPortfolioStats)
	}

	// Upload routes
	uploads := r.Group("/uploads")
	uploads.Use(middleware.AuthMiddleware())
	{
		uploads.POST("", h.UploadFile)
		uploads.GET("/:uploadId", h.GetUpload)
		uploads.GET("/user/me", h.GetMyUploads)
		uploads.GET("/entity/:entityType/:entityId", h.GetEntityUploads)
		uploads.DELETE("/:uploadId", h.DeleteUpload)
		uploads.GET("/storage/usage", h.GetStorageUsage)
	}

	// Admin routes
	admin := r.Group("/admin/uploads")
	admin.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(models.UserRoleAdmin))
	{
		admin.POST("/clean-orphaned", h.CleanOrphanedUploads)
		admin.GET("/stats", h.GetPlatformUploadStats)
	}
}

// Portfolio handlers

func (h *PortfolioHandler) CreatePortfolioItem(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.CreatePortfolioRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}

	response, err := h.portfolioService.CreatePortfolioItem(userID, &req, file)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if appErrors.Is(err, appErrors.ErrFileTooLarge) ||
			appErrors.Is(err, appErrors.ErrInvalidFileType) ||
			appErrors.Is(err, appErrors.ErrInvalidUploadUsage) {
			statusCode = http.StatusBadRequest
		} else if appErrors.Is(err, appErrors.ErrStorageLimitExceeded) {
			statusCode = http.StatusPaymentRequired
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *PortfolioHandler) GetPortfolioItem(c *gin.Context) {
	itemID := c.Param("itemId")

	response, err := h.portfolioService.GetPortfolioItem(itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Portfolio item not found"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *PortfolioHandler) GetModelPortfolio(c *gin.Context) {
	modelID := c.Param("modelId")

	responses, err := h.portfolioService.GetModelPortfolio(modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": responses,
		"total": len(responses),
	})
}

func (h *PortfolioHandler) UpdatePortfolioItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	var req dto.UpdatePortfolioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.portfolioService.UpdatePortfolioItem(userID, itemID, &req); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Portfolio item updated successfully"})
}

func (h *PortfolioHandler) UpdatePortfolioOrder(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.ReorderPortfolioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.portfolioService.UpdatePortfolioOrder(userID, &req); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied for some items" || err.Error() == "model profile not found" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Portfolio order updated successfully"})
}

func (h *PortfolioHandler) DeletePortfolioItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	if err := h.portfolioService.DeletePortfolioItem(userID, itemID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Portfolio item deleted successfully"})
}

func (h *PortfolioHandler) GetPortfolioStats(c *gin.Context) {
	modelID := c.Param("modelId")

	stats, err := h.portfolioService.GetPortfolioStats(modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *PortfolioHandler) TogglePortfolioVisibility(c *gin.Context) {
	userID := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	var req dto.PortfolioVisibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.portfolioService.TogglePortfolioVisibility(userID, itemID, &req); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Portfolio visibility updated successfully"})
}

func (h *PortfolioHandler) GetFeaturedPortfolio(c *gin.Context) {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := parseIntParam(limitParam); err == nil {
			limit = parsedLimit
		}
	}

	response, err := h.portfolioService.GetFeaturedPortfolio(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *PortfolioHandler) GetRecentPortfolio(c *gin.Context) {
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := parseIntParam(limitParam); err == nil {
			limit = parsedLimit
		}
	}

	response, err := h.portfolioService.GetRecentPortfolio(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Upload handlers

func (h *PortfolioHandler) UploadFile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.UploadRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}

	response, err := h.portfolioService.UploadFile(userID, &req, file)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if appErrors.Is(err, appErrors.ErrFileTooLarge) ||
			appErrors.Is(err, appErrors.ErrInvalidFileType) ||
			appErrors.Is(err, appErrors.ErrInvalidUploadUsage) {
			statusCode = http.StatusBadRequest
		} else if appErrors.Is(err, appErrors.ErrStorageLimitExceeded) {
			statusCode = http.StatusPaymentRequired
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *PortfolioHandler) GetUpload(c *gin.Context) {
	uploadID := c.Param("uploadId")

	upload, err := h.portfolioService.GetUpload(uploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload not found"})
		return
	}

	c.JSON(http.StatusOK, upload)
}

func (h *PortfolioHandler) GetMyUploads(c *gin.Context) {
	userID := middleware.GetUserID(c)

	uploads, err := h.portfolioService.GetUserUploads(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uploads": uploads,
		"total":   len(uploads),
	})
}

func (h *PortfolioHandler) GetEntityUploads(c *gin.Context) {
	entityType := c.Param("entityType")
	entityID := c.Param("entityId")

	uploads, err := h.portfolioService.GetEntityUploads(entityType, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uploads": uploads,
		"total":   len(uploads),
	})
}

func (h *PortfolioHandler) DeleteUpload(c *gin.Context) {
	userID := middleware.GetUserID(c)
	uploadID := c.Param("uploadId")

	if err := h.portfolioService.DeleteUpload(userID, uploadID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "access denied" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Upload deleted successfully"})
}

func (h *PortfolioHandler) GetStorageUsage(c *gin.Context) {
	userID := middleware.GetUserID(c)

	usage, err := h.portfolioService.GetUserStorageUsage(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// Admin handlers

func (h *PortfolioHandler) CleanOrphanedUploads(c *gin.Context) {
	if err := h.portfolioService.CleanOrphanedUploads(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Orphaned uploads cleaned successfully"})
}

func (h *PortfolioHandler) GetPlatformUploadStats(c *gin.Context) {
	stats, err := h.portfolioService.GetPlatformUploadStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
