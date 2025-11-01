package handlers

import (
	// "fmt" // <-- No longer needed
	"net/http"

	"mwork_backend/internal/middleware" // <-- Still needed for RegisterRoutes
	"mwork_backend/internal/models"
	// "mwork_backend/internal/repositories" // <-- Больше не нужен здесь
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"
	"mwork_backend/pkg/apperrors"

	"github.com/gin-gonic/gin"
)

type ProfileHandler struct {
	*BaseHandler   // <-- 1. Embed BaseHandler
	profileService services.ProfileService
}

// 2. Update the constructor
func NewProfileHandler(base *BaseHandler, profileService services.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		BaseHandler:    base, // <-- 3. Assign it
		profileService: profileService,
	}
}

// RegisterRoutes remains unchanged
func (h *ProfileHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Public routes
	public := r.Group("/profiles")
	{
		public.GET("/:userId", h.GetProfile)
		public.GET("/models/search", h.SearchModels)
		public.GET("/employers/search", h.SearchEmployers)
	}

	// Protected routes
	profiles := r.Group("/profiles")
	profiles.Use(middleware.AuthMiddleware())
	{
		profiles.POST("/model", h.CreateModelProfile)
		profiles.POST("/employer", h.CreateEmployerProfile)
		profiles.PUT("/me", h.UpdateMyProfile)
		profiles.PUT("/me/visibility", h.ToggleVisibility)
		profiles.GET("/me/stats", h.GetMyStats)
	}
}

// --- Profile creation handlers ---

func (h *ProfileHandler) CreateModelProfile(c *gin.Context) {
	var req dto.CreateModelProfileRequest
	// 4. Use BindAndValidate_JSON
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// 5. Use GetAndAuthorizeUserID
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	req.UserID = userID

	// ✅ DB: Используем h.GetDB(c)
	if err := h.profileService.CreateModelProfile(h.GetDB(c), &req); err != nil {
		// 6. Use HandleServiceError
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Model profile created successfully"})
}

func (h *ProfileHandler) CreateEmployerProfile(c *gin.Context) {
	var req dto.CreateEmployerProfileRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}
	req.UserID = userID

	// ✅ DB: Используем h.GetDB(c)
	if err := h.profileService.CreateEmployerProfile(h.GetDB(c), &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Employer profile created successfully"})
}

// --- Profile retrieval handlers ---

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID := c.Param("userId")
	requesterID := ""

	// This is a public route, so we check for an *optional* user ID.
	// We do NOT use GetAndAuthorizeUserID here, as that would force a 401.
	if authUserID, exists := c.Get("userID"); exists {
		requesterID, _ = authUserID.(string)
	}

	// ✅ DB: Используем h.GetDB(c)
	profile, err := h.profileService.GetProfile(h.GetDB(c), userID, requesterID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

// --- Profile update handlers ---

func (h *ProfileHandler) UpdateMyProfile(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req dto.UpdateProfileRequest
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.profileService.UpdateProfile(h.GetDB(c), userID, &req); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

func (h *ProfileHandler) ToggleVisibility(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	var req struct {
		IsPublic bool `json:"is_public"`
	}
	if !h.BindAndValidate_JSON(c, &req) {
		return
	}

	// ✅ DB: Используем h.GetDB(c)
	if err := h.profileService.ToggleProfileVisibility(h.GetDB(c), userID, req.IsPublic); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile visibility updated successfully"})
}

// --- Search handlers ---

func (h *ProfileHandler) SearchModels(c *gin.Context) {
	// ⭐ ИСПРАВЛЕНИЕ: Тип изменен на dto.SearchModelsRequest
	var criteria dto.SearchModelsRequest
	// 7. Use BindAndValidate_Query
	if !h.BindAndValidate_Query(c, &criteria) {
		return
	}

	// 8. Use ParsePagination
	criteria.Page, criteria.PageSize = ParsePagination(c)

	// ✅ DB: Используем h.GetDB(c)
	// ⭐ ИСПРАВЛЕНИЕ: передаем &criteria
	paginatedResponse, err := h.profileService.SearchModels(h.GetDB(c), &criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// ⭐ ИСПРАВЛЕНИЕ: Ответ адаптирован под dto.PaginatedResponse
	c.JSON(http.StatusOK, gin.H{
		"profiles": paginatedResponse.Data,
		"total":    paginatedResponse.Total,
		"page":     paginatedResponse.Page,
		"pages":    paginatedResponse.TotalPages,
	})
}

func (h *ProfileHandler) SearchEmployers(c *gin.Context) {
	// ⭐ ИСПРАВЛЕНИЕ: Тип изменен на dto.SearchEmployersRequest
	var criteria dto.SearchEmployersRequest
	if !h.BindAndValidate_Query(c, &criteria) {
		return
	}

	criteria.Page, criteria.PageSize = ParsePagination(c)

	// ✅ DB: Используем h.GetDB(c)
	// ⭐ ИСПРАВЛЕНИЕ: передаем &criteria
	paginatedResponse, err := h.profileService.SearchEmployers(h.GetDB(c), &criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	// ⭐ ИСПРАВЛЕНИЕ: Ответ адаптирован под dto.PaginatedResponse
	c.JSON(http.StatusOK, gin.H{
		"profiles": paginatedResponse.Data,
		"total":    paginatedResponse.Total,
		"page":     paginatedResponse.Page,
		"pages":    paginatedResponse.TotalPages,
	})
}

// --- Stats handlers ---

func (h *ProfileHandler) GetMyStats(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// Get user to determine role
	// ✅ DB: Используем h.GetDB(c)
	user, err := h.profileService.GetProfile(h.GetDB(c), userID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	if user.Type == "model" {
		// Extract model ID from profile data
		if modelProfile, ok := user.Data.(*models.ModelProfile); ok {
			// ✅ DB: Используем h.GetDB(c)
			stats, err := h.profileService.GetModelStats(h.GetDB(c), modelProfile.ID)
			if err != nil {
				h.HandleServiceError(c, err)
				return
			}
			c.JSON(http.StatusOK, stats)
			return
		}
	}

	// Use HandleServiceError for the final error case
	h.HandleServiceError(c, apperrors.NewBadRequestError("Stats not available for this profile type"))
}
