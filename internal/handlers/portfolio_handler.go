package handlers

import (
	"net/http"

	"mwork_backend/internal/middleware"
	// "mwork_backend/internal/models" // (Больше не нужен, т.к. admin middleware ушло)
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"

	"github.com/gin-gonic/gin"
)

type PortfolioHandler struct {
	*BaseHandler
	portfolioService services.PortfolioService
}

func NewPortfolioHandler(base *BaseHandler, portfolioService services.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{
		BaseHandler:      base,
		portfolioService: portfolioService,
	}
}

// RegisterRoutes - ОЧИЩЕНО
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
		portfolio.POST("", h.CreatePortfolioItem) // Этот маршрут остается, т.к. он создает Portfolio *с* файлом
		portfolio.PUT("/:itemId", h.UpdatePortfolioItem)
		portfolio.DELETE("/:itemId", h.DeletePortfolioItem)
		portfolio.PUT("/reorder", h.UpdatePortfolioOrder)
		portfolio.PUT("/:itemId/visibility", h.TogglePortfolioVisibility)
		portfolio.GET("/stats/:modelId", h.GetPortfolioStats)
	}

	// ▼▼▼ УДАЛЕНО: Все маршруты /uploads и /admin/uploads ▼▼▼
	// Группы /uploads и /admin/uploads удалены.
	// Они должны быть зарегистрированы в UploadHandler.
	// ▲▲▲ УДАЛЕНО ▲▲▲
}

// --- Portfolio handlers (Обновлены с Context) ---

func (h *PortfolioHandler) CreatePortfolioItem(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.CreatePortfolioRequest
	// Используем c.ShouldBind() для multipart-form
	if err := c.ShouldBind(&req); err != nil {
		h.HandleServiceError(c, apperrors.NewBadRequestError(err.Error()))
		return
	}
	// Валидация
	if err := h.validator.Validate(req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		apperrors.HandleError(c, apperrors.NewBadRequestError("File is required"))
		return
	}

	// ✅ DB + Context
	response, err := h.portfolioService.CreatePortfolioItem(c.Request.Context(), h.GetDB(c), userID, &req, file)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *PortfolioHandler) GetPortfolioItem(c *gin.Context) {
	itemID := c.Param("itemId")

	// ✅ DB + Context
	response, err := h.portfolioService.GetPortfolioItem(c.Request.Context(), h.GetDB(c), itemID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *PortfolioHandler) GetModelPortfolio(c *gin.Context) {
	modelID := c.Param("modelId")

	// ✅ DB + Context
	responses, err := h.portfolioService.GetModelPortfolio(c.Request.Context(), h.GetDB(c), modelID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": responses,
		"total": len(responses),
	})
}

func (h *PortfolioHandler) UpdatePortfolioItem(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	itemID := c.Param("itemId")

	var req dto.UpdatePortfolioRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	if err := h.portfolioService.UpdatePortfolioItem(c.Request.Context(), h.GetDB(c), userID, itemID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Portfolio item updated successfully"})
}

func (h *PortfolioHandler) UpdatePortfolioOrder(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.ReorderPortfolioRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	if err := h.portfolioService.UpdatePortfolioOrder(c.Request.Context(), h.GetDB(c), userID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Portfolio order updated successfully"})
}

func (h *PortfolioHandler) DeletePortfolioItem(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	itemID := c.Param("itemId")

	// ✅ DB + Context
	if err := h.portfolioService.DeletePortfolioItem(c.Request.Context(), h.GetDB(c), userID, itemID); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Portfolio item deleted successfully"})
}

func (h *PortfolioHandler) GetPortfolioStats(c *gin.Context) {
	if _, ok := h.GetAndAuthorizeUserID(c); !ok {
		return
	}
	modelID := c.Param("modelId")

	// ✅ DB + Context
	stats, err := h.portfolioService.GetPortfolioStats(c.Request.Context(), h.GetDB(c), modelID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *PortfolioHandler) TogglePortfolioVisibility(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	itemID := c.Param("itemId")

	var req dto.PortfolioVisibilityRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB + Context
	if err := h.portfolioService.TogglePortfolioVisibility(c.Request.Context(), h.GetDB(c), userID, itemID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Portfolio visibility updated successfully"})
}

func (h *PortfolioHandler) GetFeaturedPortfolio(c *gin.Context) {
	limit := ParseQueryInt(c, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	// ✅ DB + Context
	response, err := h.portfolioService.GetFeaturedPortfolio(c.Request.Context(), h.GetDB(c), limit)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *PortfolioHandler) GetRecentPortfolio(c *gin.Context) {
	limit := ParseQueryInt(c, "limit", 10)
	if limit <= 0 {
		limit = 10
	}

	// ✅ DB + Context
	response, err := h.portfolioService.GetRecentPortfolio(c.Request.Context(), h.GetDB(c), limit)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// --- ▼▼▼ УДАЛЕНО: Все обработчики Uploads ▼▼▼ ---
//
// func (h *PortfolioHandler) UploadFile(c *gin.Context) { ... }
// func (h *PortfolioHandler) GetUpload(c *gin.Context) { ... }
// func (h *PortfolioHandler) GetMyUploads(c *gin.Context) { ... }
// func (h *PortfolioHandler) GetEntityUploads(c *gin.Context) { ... }
// func (h *PortfolioHandler) DeleteUpload(c *gin.Context) { ... }
// func (h *PortfolioHandler) GetStorageUsage(c *gin.Context) { ... }
// func (h *PortfolioHandler) CleanOrphanedUploads(c *gin.Context) { ... }
// func (h *PortfolioHandler) GetPlatformUploadStats(c *gin.Context) { ... }
//
// --- ▲▲▲ УДАЛЕНО ▲▲▲ ---
