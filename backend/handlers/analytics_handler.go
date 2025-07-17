package handlers

import (
	"github.com/gin-gonic/gin"
	_ "mwork_front_fn/backend/dto"
	"mwork_front_fn/backend/repositories"
	"mwork_front_fn/backend/services"
	"net/http"
)

type AnalyticsHandler struct {
	AnalyticsService *services.AnalyticsService
	ProfileRepo      repositories.ModelProfileRepository // чтобы найти modelID по userID
}

func NewAnalyticsHandler(service *services.AnalyticsService, profileRepo repositories.ModelProfileRepository) *AnalyticsHandler {
	return &AnalyticsHandler{
		AnalyticsService: service,
		ProfileRepo:      profileRepo,
	}
}

// GetModelAnalytics godoc
// @Summary Получить аналитику модели
// @Description Возвращает статистику по просмотрам, рейтингу, доходу и откликам модели. UserID берётся из JWT.
// @Tags analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.ModelAnalytics
// @Failure 400 {object} models.ErrorResponse "Некорректный userID"
// @Failure 401 {object} models.ErrorResponse "Неавторизован или отсутствует токен"
// @Failure 404 {object} models.ErrorResponse "Профиль модели не найден"
// @Failure 500 {object} models.ErrorResponse "Ошибка при получении аналитики"
// @Router /analytics/model [get]
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
