package handlers

import (
	"github.com/gin-gonic/gin"
	"mwork_front_fn/backend/models"
	"mwork_front_fn/backend/services"
	"net/http"
)

type ModelProfileHandler struct {
	service *services.ModelProfileService
}

func NewModelProfileHandler(service *services.ModelProfileService) *ModelProfileHandler {
	return &ModelProfileHandler{service: service}
}

// POST /model-profiles
// CreateProfile godoc
// @Summary Создать модельный профиль
// @Description Создаёт новый профиль модели на основе переданных данных
// @Tags model-profiles
// @Accept json
// @Produce json
// @Param profile body models.ModelProfile true "Данные модельного профиля"
// @Success 201 {object} models.ModelProfile
// @Failure 400 {object} models.ErrorResponse "Некорректный JSON"
// @Failure 500 {object} models.ErrorResponse "Ошибка сервера при создании профиля"
// @Router /model-profiles [post]
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

// GET /model-profiles/:user_id
// GetProfile godoc
// @Summary Получить профиль модели по ID пользователя
// @Description Возвращает профиль модели, связанный с указанным user_id
// @Tags model-profiles
// @Accept json
// @Produce json
// @Param user_id path string true "ID пользователя"
// @Success 200 {object} models.ModelProfile
// @Failure 404 {object} models.ErrorResponse "Профиль не найден"
// @Router /model-profiles/{user_id} [get]
func (h *ModelProfileHandler) GetProfile(c *gin.Context) {
	userID := c.Param("user_id")
	profile, err := h.service.GetProfileByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}
