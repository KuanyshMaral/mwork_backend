package handlers

import (
	// "fmt" // <-- No longer needed
	"net/http"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/middleware" // <-- Still needed for RegisterRoutes
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

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

	if err := h.profileService.CreateModelProfile(&req); err != nil {
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

	if err := h.profileService.CreateEmployerProfile(&req); err != nil {
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

	profile, err := h.profileService.GetProfile(userID, requesterID)
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

	if err := h.profileService.UpdateProfile(userID, &req); err != nil {
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

	if err := h.profileService.ToggleProfileVisibility(userID, req.IsPublic); err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile visibility updated successfully"})
}

// --- Search handlers ---

func (h *ProfileHandler) SearchModels(c *gin.Context) {
	var criteria dto.ProfileSearchCriteria
	// 7. Use BindAndValidate_Query
	if !h.BindAndValidate_Query(c, &criteria) {
		return
	}

	// 8. Use ParsePagination
	criteria.Page, criteria.PageSize = ParsePagination(c)

	profiles, total, err := h.profileService.SearchModels(criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"profiles": profiles,
		"total":    total,
		"page":     criteria.Page,
		"pages":    (total + int64(criteria.PageSize) - 1) / int64(criteria.PageSize),
	})
}

func (h *ProfileHandler) SearchEmployers(c *gin.Context) {
	var criteria repositories.EmployerSearchCriteria
	if !h.BindAndValidate_Query(c, &criteria) {
		return
	}

	criteria.Page, criteria.PageSize = ParsePagination(c)

	profiles, total, err := h.profileService.SearchEmployers(criteria)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"profiles": profiles,
		"total":    total,
		"page":     criteria.Page,
		"pages":    (total + int64(criteria.PageSize) - 1) / int64(criteria.PageSize),
	})
}

// --- Stats handlers ---

func (h *ProfileHandler) GetMyStats(c *gin.Context) {
	userID, ok := h.GetAndAuthorizeUserID(c)
	if !ok {
		return
	}

	// Get user to determine role
	user, err := h.profileService.GetProfile(userID, userID)
	if err != nil {
		h.HandleServiceError(c, err)
		return
	}

	if user.Type == "model" {
		// Extract model ID from profile data
		if modelProfile, ok := user.Data.(*models.ModelProfile); ok {
			stats, err := h.profileService.GetModelStats(modelProfile.ID)
			if err != nil {
				h.HandleServiceError(c, err)
				return
			}
			c.JSON(http.StatusOK, stats)
			return
		}
	}

	// Use HandleServiceError for the final error case
	h.HandleServiceError(c, appErrors.NewBadRequestError("Stats not available for this profile type"))
}
