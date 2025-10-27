package old_shit

import (
	"mwork_backend/internal/repositories/old_bullshit"
	"mwork_backend/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	AnalyticsService *services.AnalyticsService
	ProfileRepo      old_bullshit.ModelProfileRepository // чтобы найти modelID по userID
}

func NewAnalyticsHandler(service *services.AnalyticsService, profileRepo old_bullshit.ModelProfileRepository) *AnalyticsHandler {
	return &AnalyticsHandler{
		AnalyticsService: service,
		ProfileRepo:      profileRepo,
	}
}

func (h *AnalyticsHandler) GetModelAnalytics(c *gin.Context) {
	userIDany, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDany.(string) // ✅ раньше был uint
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	profile, err := h.ProfileRepo.GetByUserID(c.Request.Context(), userID) // ✅ обязательно ctx + string
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model profile not found"})
		return
	}

	analytics, err := h.AnalyticsService.GetModelAnalytics(profile.ID) // ✅ profile.ID — тоже string
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Analytics failed"})
		return
	}

	c.JSON(http.StatusOK, analytics)
}
