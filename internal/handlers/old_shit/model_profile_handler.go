package old_shit

import (
	"mwork_backend/internal/models"
	"mwork_backend/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ModelProfileHandler struct {
	service *services.ModelProfileService
}

func NewModelProfileHandler(service *services.ModelProfileService) *ModelProfileHandler {
	return &ModelProfileHandler{service: service}
}

func (h *ModelProfileHandler) CreateProfile(c *gin.Context) {
	var profile models.ModelProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if err := h.service.CreateProfile(c.Request.Context(), &profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // ← временно выводим ошибку
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create profile"})
		return
	}

	c.JSON(http.StatusCreated, profile)
}

func (h *ModelProfileHandler) GetProfile(c *gin.Context) {
	userID := c.Param("user_id")
	profile, err := h.service.GetProfileByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}
