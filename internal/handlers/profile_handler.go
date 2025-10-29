package handlers

import (
	"fmt"
	"net/http"

	"mwork_backend/internal/appErrors"
	"mwork_backend/internal/middleware"
	"mwork_backend/internal/models"
	"mwork_backend/internal/repositories"
	"mwork_backend/internal/services"
	"mwork_backend/internal/services/dto"

	"github.com/gin-gonic/gin"
)

type ProfileHandler struct {
	profileService *services.ProfileService
}

func NewProfileHandler(profileService *services.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

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

// Profile creation handlers

func (h *ProfileHandler) CreateModelProfile(c *gin.Context) {
	var req dto.CreateModelProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.UserID = middleware.GetUserID(c)

	if err := h.profileService.CreateModelProfile(&req); err != nil {
		statusCode := http.StatusInternalServerError
		if appErrors.Is(err, appErrors.ErrInvalidUserRole) {
			statusCode = http.StatusForbidden
		} else if err.Error() == "age must be between 16 and 70" ||
			err.Error() == "height must be between 100 and 250 cm" ||
			err.Error() == "hourly rate cannot be negative" {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Model profile created successfully"})
}

func (h *ProfileHandler) CreateEmployerProfile(c *gin.Context) {
	var req dto.CreateEmployerProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.UserID = middleware.GetUserID(c)

	if err := h.profileService.CreateEmployerProfile(&req); err != nil {
		statusCode := http.StatusInternalServerError
		if appErrors.Is(err, appErrors.ErrInvalidUserRole) {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Employer profile created successfully"})
}

// Profile retrieval handlers

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID := c.Param("userId")
	requesterID := ""

	// Get requester ID if authenticated
	if authUserID, exists := c.Get("userID"); exists {
		requesterID = authUserID.(string)
	}

	profile, err := h.profileService.GetProfile(userID, requesterID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if appErrors.Is(err, appErrors.ErrProfileNotPublic) {
			statusCode = http.StatusForbidden
		} else if appErrors.Is(err, appErrors.ErrInvalidUserRole) {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// Profile update handlers

func (h *ProfileHandler) UpdateMyProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.profileService.UpdateProfile(userID, &req); err != nil {
		statusCode := http.StatusInternalServerError
		if appErrors.Is(err, appErrors.ErrInvalidUserRole) {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

func (h *ProfileHandler) ToggleVisibility(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		IsPublic bool `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.profileService.ToggleProfileVisibility(userID, req.IsPublic); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "only model profiles can toggle visibility" {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile visibility updated successfully"})
}

// Search handlers

func (h *ProfileHandler) SearchModels(c *gin.Context) {
	var criteria dto.ProfileSearchCriteria
	if err := c.ShouldBindQuery(&criteria); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	// Set defaults
	if criteria.Page == 0 {
		criteria.Page = 1
	}
	if criteria.PageSize == 0 {
		criteria.PageSize = 20
	}

	profiles, total, err := h.profileService.SearchModels(criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
	if err := c.ShouldBindQuery(&criteria); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	// Set defaults
	if criteria.Page == 0 {
		criteria.Page = 1
	}
	if criteria.PageSize == 0 {
		criteria.PageSize = 20
	}

	profiles, total, err := h.profileService.SearchEmployers(criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"profiles": profiles,
		"total":    total,
		"page":     criteria.Page,
		"pages":    (total + int64(criteria.PageSize) - 1) / int64(criteria.PageSize),
	})
}

// Stats handlers

func (h *ProfileHandler) GetMyStats(c *gin.Context) {
	userID := middleware.GetUserID(c)

	// Get user to determine role
	user, err := h.profileService.GetProfile(userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if user.Type == "model" {
		// Extract model ID from profile data
		if modelProfile, ok := user.Data.(*models.ModelProfile); ok {
			stats, err := h.profileService.GetModelStats(modelProfile.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, stats)
			return
		}
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "Stats not available for this profile type"})
}

// ============================================================================
// Helper functions
// ============================================================================

func parseIntParam(param string) (int, error) {
	var result int
	if _, err := fmt.Sscanf(param, "%d", &result); err != nil {
		return 0, err
	}
	return result, nil
}
