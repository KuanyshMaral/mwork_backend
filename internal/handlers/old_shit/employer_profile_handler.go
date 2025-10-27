package old_shit

import (
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type EmployerProfileHandler struct {
	service *services.EmployerProfileService
}

func NewEmployerProfileHandler(service *services.EmployerProfileService) *EmployerProfileHandler {
	return &EmployerProfileHandler{service: service}
}

func (h *EmployerProfileHandler) CreateProfile(c *gin.Context) {
	var profile models.EmployerProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if err := h.service.CreateProfile(c.Request.Context(), &profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create employer profile"})
		return
	}

	c.JSON(http.StatusCreated, profile)
}

func (h *EmployerProfileHandler) GetProfile(c *gin.Context) {
	userID := c.Param("user_id")
	profile, err := h.service.GetProfileByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}
